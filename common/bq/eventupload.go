// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bq

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
)

// ID is the global InsertIDGenerator
var ID InsertIDGenerator

// fieldInfo is metadata of a proto field.
// Retrieve field infos using getFieldInfos.
//
// For oneof, one oneof declaration is mapped to one fieldinfo,
// as opposed to one fieldinfo per oneof member.
type fieldInfo struct {
	*proto.Properties
	structIndex []int
	// oneOfFields maps a oneof struct type to its metadata.
	// Initialized only for oneof declaration fields.
	oneOfFields map[reflect.Type]oneOfFieldInfo
}

type oneOfFieldInfo struct {
	*proto.Properties
	valueFieldIndex []int // index of the field within a oneof struct
}

var bqFields = map[reflect.Type][]fieldInfo{}
var bqFieldsLock = sync.RWMutex{}

var protoMessageType = reflect.TypeOf((*proto.Message)(nil)).Elem()

const insertLimit = 10000
const batchDefault = 500

// Uploader contains the necessary data for streaming data to BigQuery.
type Uploader struct {
	*bigquery.Uploader
	// Uploader is bound to a specific table. DatasetID and Table ID are
	// provided for reference.
	DatasetID string
	TableID   string
	// UploadsMetricName is a string used to create a tsmon Counter metric
	// for event upload attempts via Put, e.g.
	// "/chrome/infra/commit_queue/events/count". If unset, no metric will
	// be created.
	UploadsMetricName string
	// uploads is the Counter metric described by UploadsMetricName. It
	// contains a field "status" set to either "success" or "failure."
	uploads        metric.Counter
	initMetricOnce sync.Once
	// BatchSize is the max number of rows to send to BigQuery at a time.
	// The default is 500.
	BatchSize int
}

// Row implements bigquery.ValueSaver
type Row struct {
	proto.Message // embedded

	// InsertID is unique per insert operation to handle deduplication.
	InsertID string
}

// Save is used by bigquery.Uploader.Put when inserting values into a table.
func (r *Row) Save() (map[string]bigquery.Value, string, error) {
	m, err := mapFromMessage(r.Message, nil)
	return m, r.InsertID, err
}

// mapFromMessage returns a {BQ Field name: BQ value} map.
// path is a slice of Go field names leading to m.
func mapFromMessage(m proto.Message, path []string) (map[string]bigquery.Value, error) {
	sPtr := reflect.ValueOf(m)
	switch {
	case sPtr.Kind() != reflect.Ptr:
		return nil, fmt.Errorf("type %T implementing proto.Message is not a pointer", m)
	case sPtr.IsNil():
		return nil, nil
	}

	s := sPtr.Elem()
	if s.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type %T implementing proto.Message is not a pointer to a struct", m)
	}

	t := s.Type()
	infos, err := getFieldInfos(t)
	if err != nil {
		return nil, errors.Annotate(err, "could not populate bqFields for type %v", t).Err()
	}
	path = append(path, "")

	var row map[string]bigquery.Value // keep it nil unless there are values
	for _, fi := range infos {
		var bqField string
		var bqValue interface{}
		path[len(path)-1] = fi.Name

		switch {
		case len(fi.oneOfFields) != 0:
			val := s.FieldByIndex(fi.structIndex)
			if val.IsNil() {
				continue
			}
			structPtr := val.Elem()
			oof := fi.oneOfFields[structPtr.Type()]
			bqField = oof.OrigName
			rawValue := structPtr.Elem().FieldByIndex(oof.valueFieldIndex).Interface()
			if bqValue, err = getValue(rawValue, path, oof.Properties); err != nil {
				return nil, errors.Annotate(err, "%s", fi.OrigName).Err()
			} else if bqValue == nil {
				// Omit NULL values.
				continue
			}

		case fi.Repeated:
			f := s.FieldByIndex(fi.structIndex)
			// init value only if there are elements
			n := f.Len()
			if n == 0 {
				// omit a repeated field with no elements.
				continue
			}

			elems := make([]interface{}, n)
			vPath := append(path, "")
			switch f.Kind() {
			case reflect.Slice:
				for i := 0; i < len(elems); i++ {
					vPath[len(vPath)-1] = strconv.Itoa(i)
					elems[i], err = getValue(f.Index(i).Interface(), vPath, fi.Properties)
					if err != nil {
						return nil, errors.Annotate(err, "%s[%d]", fi.OrigName, i).Err()
					}
				}

			case reflect.Map:
				if f.Type().Key().Kind() != reflect.String {
					return nil, fmt.Errorf("map key must be a string")
				}

				keys := f.MapKeys()
				sort.Slice(keys, func(i, j int) bool {
					return keys[i].String() < keys[j].String()
				})

				for i, k := range keys {
					kStr := k.String()
					vPath[len(vPath)-1] = kStr
					elemValue, err := getValue(f.MapIndex(k).Interface(), vPath, fi.Properties)
					if err != nil {
						return nil, errors.Annotate(err, "%s[%s]", fi.OrigName, kStr).Err()
					}
					elems[i] = map[string]bigquery.Value{
						"key":   kStr,
						"value": elemValue,
					}
				}

			default:
				return nil, fmt.Errorf("kind %s not supported as a repeated field", f.Kind())
			}
			bqField = fi.OrigName
			bqValue = elems

		default:
			bqField = fi.OrigName
			if bqValue, err = getValue(s.FieldByIndex(fi.structIndex).Interface(), path, fi.Properties); err != nil {
				return nil, errors.Annotate(err, "%s", fi.OrigName).Err()
			} else if bqValue == nil {
				// Omit NULL values.
				continue
			}
		}

		if row == nil {
			row = map[string]bigquery.Value{}
		}
		row[bqField] = bigquery.Value(bqValue)
	}
	return row, nil
}

// getFieldInfos returns field metadata for a given proto go type.
// Caches results.
func getFieldInfos(t reflect.Type) ([]fieldInfo, error) {
	bqFieldsLock.RLock()
	f := bqFields[t]
	bqFieldsLock.RUnlock()
	if f != nil {
		return f, nil
	}

	bqFieldsLock.Lock()
	defer bqFieldsLock.Unlock()
	return getFieldInfosLocked(t)
}

func getFieldInfosLocked(t reflect.Type) ([]fieldInfo, error) {
	if f := bqFields[t]; f != nil {
		return f, nil
	}

	structProp := proto.GetProperties(t)

	oneOfs := map[int]map[reflect.Type]oneOfFieldInfo{}
	for _, of := range structProp.OneofTypes {
		f, ok := of.Type.Elem().FieldByName(of.Prop.Name)
		if !ok {
			return nil, fmt.Errorf("field %q not found in %q", of.Prop.Name, of.Type)
		}

		typeMap := oneOfs[of.Field]
		if typeMap == nil {
			typeMap = map[reflect.Type]oneOfFieldInfo{}
			oneOfs[of.Field] = typeMap
		}
		typeMap[of.Type] = oneOfFieldInfo{
			Properties:      of.Prop,
			valueFieldIndex: f.Index,
		}
	}

	fields := make([]fieldInfo, 0, len(structProp.Prop))
	for _, p := range structProp.Prop {
		if strings.HasPrefix(p.Name, "XXX_") {
			continue
		}

		f, ok := t.FieldByName(p.Name)
		if !ok {
			return nil, fmt.Errorf("field %q not found in %q", p.Name, t)
		}

		ft := f.Type
		if ft.Kind() == reflect.Slice {
			ft = ft.Elem()
		}
		if ft.Implements(protoMessageType) && ft.Kind() == reflect.Ptr {
			if st := ft.Elem(); st.Kind() == reflect.Struct {
				// Note: this will crash with a stack overflow if the protobuf
				// message is recursive, but bqschemaupdater should catch that
				// earlier.
				subfields, err := getFieldInfosLocked(st)
				if err != nil {
					return nil, err
				}
				if len(subfields) == 0 {
					// Skip RECORD fields with no sub-fields.
					continue
				}
			}
		}
		fields = append(fields, fieldInfo{
			Properties:  p,
			structIndex: f.Index,
			oneOfFields: oneOfs[f.Index[0]],
		})
	}

	bqFields[t] = fields
	return fields, nil
}

func getValue(value interface{}, path []string, prop *proto.Properties) (interface{}, error) {
	if prop.Enum != "" {
		stringer, ok := value.(fmt.Stringer)
		if !ok {
			return nil, fmt.Errorf("could not convert enum value to string")
		}
		return stringer.String(), nil
	} else if dpb, ok := value.(*duration.Duration); ok {
		if dpb == nil {
			return nil, nil
		}
		value, err := ptypes.Duration(dpb)
		if err != nil {
			return nil, fmt.Errorf("tried to write an invalid duration for [%+v] for field %q", dpb, strings.Join(path, "."))
		}
		// Convert to FLOAT64.
		return value.Seconds(), nil
	} else if tspb, ok := value.(*timestamp.Timestamp); ok {
		if tspb == nil {
			return nil, nil
		}
		value, err := ptypes.Timestamp(tspb)
		if err != nil {
			return nil, fmt.Errorf("tried to write an invalid timestamp for [%+v] for field %q", tspb, strings.Join(path, "."))
		}
		return value, nil
	} else if s, ok := value.(*structpb.Struct); ok {
		if s == nil {
			return nil, nil
		}
		// Structs are persisted as JSONPB strings.
		// See also https://bit.ly/chromium-bq-struct
		var buf bytes.Buffer
		if err := (&jsonpb.Marshaler{}).Marshal(&buf, s); err != nil {
			return nil, err
		}
		return buf.String(), nil
	} else if nested, ok := value.(proto.Message); ok {
		if nested == nil {
			return nil, nil
		}
		m, err := mapFromMessage(nested, path)
		if m == nil {
			// a nil map is not nil when converted to interface{},
			// so return nil explicitly.
			return nil, err
		}
		return m, err
	} else {
		return value, nil
	}
}

// NewUploader constructs a new Uploader struct.
//
// DatasetID and TableID are provided to the BigQuery client to
// gain access to a particular table.
//
// You may want to change the default configuration of the bigquery.Uploader.
// Check the documentation for more details.
//
// Set UploadsMetricName on the resulting Uploader to use the default counter
// metric.
//
// Set BatchSize to set a custom batch size.
func NewUploader(ctx context.Context, c *bigquery.Client, datasetID, tableID string) *Uploader {
	return &Uploader{
		DatasetID: datasetID,
		TableID:   tableID,
		Uploader:  c.Dataset(datasetID).Table(tableID).Uploader(),
	}
}

func (u *Uploader) batchSize() int {
	switch {
	case u.BatchSize > insertLimit:
		return insertLimit
	case u.BatchSize <= 0:
		return batchDefault
	default:
		return u.BatchSize
	}
}

func (u *Uploader) getCounter() metric.Counter {
	u.initMetricOnce.Do(func() {
		if u.UploadsMetricName != "" {
			desc := "Upload attempts; status is 'success' or 'failure'"
			field := field.String("status")
			u.uploads = metric.NewCounter(u.UploadsMetricName, desc, nil, field)
		}
	})
	return u.uploads
}

func (u *Uploader) updateUploads(ctx context.Context, count int64, status string) {
	if uploads := u.getCounter(); uploads != nil && count != 0 {
		uploads.Add(ctx, count, status)
	}
}

// Put uploads one or more rows to the BigQuery service. Put takes care of
// adding InsertIDs, used by BigQuery to deduplicate rows.
//
// If any rows do now match one of the expected types, Put will not attempt to
// upload any rows and returns an InvalidTypeError.
//
// Put returns a PutMultiError if one or more rows failed to be uploaded.
// The PutMultiError contains a RowInsertionError for each failed row.
//
// Put will retry on temporary errors. If the error persists, the call will
// run indefinitely. Because of this, if ctx does not have a timeout, Put will
// add one.
//
// See bigquery documentation and source code for detailed information on how
// struct values are mapped to rows.
func (u *Uploader) Put(ctx context.Context, messages ...proto.Message) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}
	rows := make([]*Row, len(messages))
	for i, m := range messages {
		rows[i] = &Row{
			Message:  m,
			InsertID: ID.Generate(),
		}
	}

	return parallel.WorkPool(16, func(workC chan<- func() error) {
		for _, rowSet := range batch(rows, u.batchSize()) {
			rowSet := rowSet
			workC <- func() error {
				var failed int
				err := u.Uploader.Put(ctx, rowSet)
				if err != nil {
					logging.WithError(err).Errorf(ctx, "eventupload: Uploader.Put failed")
					if merr, ok := err.(bigquery.PutMultiError); ok {
						if failed = len(merr); failed > len(rowSet) {
							logging.Errorf(ctx, "eventupload: %v failures trying to insert %v rows", failed, len(rowSet))
						}
					} else {
						failed = len(rowSet)
					}
					u.updateUploads(ctx, int64(failed), "failure")
				}
				succeeded := len(rowSet) - failed
				u.updateUploads(ctx, int64(succeeded), "success")
				return err
			}
		}
	})
}

func batch(rows []*Row, batchSize int) [][]*Row {
	rowSetsLen := int(math.Ceil(float64(len(rows) / batchSize)))
	rowSets := make([][]*Row, 0, rowSetsLen)
	for len(rows) > 0 {
		batch := rows
		if len(batch) > batchSize {
			batch = batch[:batchSize]
		}
		rowSets = append(rowSets, batch)
		rows = rows[len(batch):]
	}
	return rowSets
}
