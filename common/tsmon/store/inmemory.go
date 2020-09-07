// Copyright 2015 The LUCI Authors.
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

package store

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/target"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
)

type inMemoryStore struct {
	defaultTarget atomic.Value

	data     map[dataKey]*metricData
	dataLock sync.RWMutex
}

type cellKey struct {
	fieldValuesHash, targetHash uint64
}

type dataKey struct {
	metricName string
	targetType types.TargetType
}

type metricData struct {
	types.MetricInfo
	types.MetricMetadata

	cells map[cellKey][]*types.CellData
	lock  sync.Mutex
}

// findTarget returns the target instance for the given metric.
func (s *inMemoryStore) findTarget(ctx context.Context, m *metricData) types.Target {
	dt := s.DefaultTarget()

	/* If the type of the metric is NilType or the type of the default target,
	   then this function returns the target, in the context, matched with
	   the type of the default target.

	   If there is no target found in the context matched with the type, then
	   nil will be returned, and therefore, the target of newly added cells
	   will be determined at the time of store.GetAll() being invoked.

	   e.g.,
	   // create different targets.
	   target1, target2, target3 := target.Task(...), ....
	   metric := NewInt(...)

	   // Set the default target with target1.
	   store.SetDefaultTarget(target1)
	   // This creates a new cell with Nil target. It means that the target of
	   // the new cell has not been determined yet. In other words,
	   // SetDefaultTarget() doesn't always guarantee that all the new cells
	   // created after the SetDefaultTarget() invocation will have the target.
	   metric.Set(ctx, 42)

	   // Create a target context with target2.
	   tctx := target.Set(target2)
	   // This creates a new cell with target2.
	   metric.Incr(tctx, 43)

	   // Set the default target with target3.
	   SetDefaultTarget(target3)

	   // This will return cells with the following elements.
	   // - value(42), target(target3)
	   // - value(43), target(target2)
	   cells := store.GetAll()
	*/
	if m.TargetType == target.NilType || m.TargetType == dt.Type() {
		return target.Get(ctx, dt.Type())
	}

	ct := target.Get(ctx, m.TargetType)
	if ct != nil {
		return ct
	}
	panic(fmt.Sprintf(
		"Missing target for Metric %s with TargetType %s", m.Name, m.TargetType,
	))
}

func (m *metricData) get(fieldVals []interface{}, t types.Target, resetTime time.Time) *types.CellData {
	fieldVals, err := field.Canonicalize(m.Fields, fieldVals)
	if err != nil {
		panic(err) // bad field types, can only happen if the code is wrong
	}

	key := cellKey{fieldValuesHash: field.Hash(fieldVals)}
	if t != nil {
		key.targetHash = t.Hash()
	}

	cells, ok := m.cells[key]
	if ok {
		for _, cell := range cells {
			if reflect.DeepEqual(fieldVals, cell.FieldVals) &&
				reflect.DeepEqual(t, cell.Target) {
				return cell
			}
		}
	}

	cell := &types.CellData{fieldVals, t, resetTime, nil}
	m.cells[key] = append(cells, cell)
	return cell
}

func (m *metricData) del(fieldVals []interface{}, t types.Target) {
	fieldVals, err := field.Canonicalize(m.Fields, fieldVals)
	if err != nil {
		panic(err) // bad field types, can only happen if the code is wrong
	}

	key := cellKey{fieldValuesHash: field.Hash(fieldVals)}
	if t != nil {
		key.targetHash = t.Hash()
	}

	cells, ok := m.cells[key]
	if ok {
		for i, cell := range cells {
			if reflect.DeepEqual(fieldVals, cell.FieldVals) &&
				reflect.DeepEqual(t, cell.Target) {
				// if the length is 1, then simply delete the map entry.
				if len(cells) == 1 {
					delete(m.cells, key)
					return
				}
				cells[i] = cells[len(cells)-1]
				m.cells[key] = cells[:len(cells)-1]
				return
			}
		}
	}
	return
}

// NewInMemory creates a new metric store that holds metric data in this
// process' memory.
func NewInMemory(defaultTarget types.Target) Store {
	s := &inMemoryStore{
		data: map[dataKey]*metricData{},
	}
	s.SetDefaultTarget(defaultTarget)
	return s
}

func (s *inMemoryStore) getOrCreateData(m types.Metric) *metricData {
	dk := dataKey{m.Info().Name, m.Info().TargetType}
	s.dataLock.RLock()
	d, ok := s.data[dk]
	s.dataLock.RUnlock()
	if ok {
		return d
	}

	s.dataLock.Lock()
	defer s.dataLock.Unlock()

	// Check again in case another goroutine got the lock before us.
	if d, ok = s.data[dk]; ok {
		return d
	}

	d = &metricData{
		MetricInfo: m.Info(),
		cells:      map[cellKey][]*types.CellData{},
	}

	s.data[dk] = d
	return d
}

func (s *inMemoryStore) DefaultTarget() types.Target {
	return s.defaultTarget.Load().(types.Target)
}

func (s *inMemoryStore) SetDefaultTarget(t types.Target) {
	s.defaultTarget.Store(t.Clone())
}

// Get returns the value for a given metric cell.
func (s *inMemoryStore) Get(ctx context.Context, h types.Metric, resetTime time.Time, fieldVals []interface{}) interface{} {
	if resetTime.IsZero() {
		resetTime = clock.Now(ctx)
	}

	m := s.getOrCreateData(h)
	t := s.findTarget(ctx, m)
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.get(fieldVals, t, resetTime).Value
}

func isLessThan(a, b interface{}) bool {
	if a == nil || b == nil {
		return false
	}
	switch a.(type) {
	case int64:
		return a.(int64) < b.(int64)
	case float64:
		return a.(float64) < b.(float64)
	}
	return false
}

// Set writes the value into the given metric cell.
func (s *inMemoryStore) Set(ctx context.Context, h types.Metric, resetTime time.Time, fieldVals []interface{}, value interface{}) {
	if resetTime.IsZero() {
		resetTime = clock.Now(ctx)
	}
	m := s.getOrCreateData(h)
	t := s.findTarget(ctx, m)
	m.lock.Lock()
	defer m.lock.Unlock()

	c := m.get(fieldVals, t, resetTime)

	if m.ValueType.IsCumulative() && isLessThan(value, c.Value) {
		logging.Errorf(ctx,
			"Attempted to set cumulative metric %q to %v, which is lower than the previous value %v",
			h.Info().Name, value, c.Value)
		return
	}

	c.Value = value
}

// Del deletes the metric cell.
func (s *inMemoryStore) Del(ctx context.Context, h types.Metric, fieldVals []interface{}) {
	m := s.getOrCreateData(h)
	t := s.findTarget(ctx, m)
	m.lock.Lock()
	defer m.lock.Unlock()
	m.del(fieldVals, t)
	return
}

// Incr increments the value in a given metric cell by the given delta.
func (s *inMemoryStore) Incr(ctx context.Context, h types.Metric, resetTime time.Time, fieldVals []interface{}, delta interface{}) {
	if resetTime.IsZero() {
		resetTime = clock.Now(ctx)
	}
	m := s.getOrCreateData(h)
	t := s.findTarget(ctx, m)
	m.lock.Lock()
	defer m.lock.Unlock()

	c := m.get(fieldVals, t, resetTime)

	switch m.ValueType {
	case types.CumulativeDistributionType:
		d, ok := delta.(float64)
		if !ok {
			panic(fmt.Errorf("Incr got a delta of unsupported type (%v)", delta))
		}
		v, ok := c.Value.(*distribution.Distribution)
		if !ok {
			v = distribution.New(h.(types.DistributionMetric).Bucketer())
			c.Value = v
		}
		v.Add(float64(d))

	case types.CumulativeIntType:
		if v, ok := c.Value.(int64); ok {
			c.Value = v + delta.(int64)
		} else {
			c.Value = delta.(int64)
		}

	case types.CumulativeFloatType:
		if v, ok := c.Value.(float64); ok {
			c.Value = v + delta.(float64)
		} else {
			c.Value = delta.(float64)
		}

	default:
		panic(fmt.Errorf("attempted to increment non-cumulative metric %s by %v", m.Name, delta))
	}
}

// GetAll efficiently returns all cells in the store.
func (s *inMemoryStore) GetAll(ctx context.Context) []types.Cell {
	s.dataLock.Lock()
	defer s.dataLock.Unlock()

	defaultTarget := s.DefaultTarget()
	ret := []types.Cell{}
	for _, m := range s.data {
		m.lock.Lock()
		for _, cells := range m.cells {
			for _, cell := range cells {
				// Add the default target to the cell if it doesn't have one set.
				cellCopy := *cell
				if cellCopy.Target == nil {
					cellCopy.Target = defaultTarget
				}
				ret = append(ret, types.Cell{m.MetricInfo, m.MetricMetadata, cellCopy})
			}
		}
		m.lock.Unlock()
	}
	return ret
}

func (s *inMemoryStore) Reset(ctx context.Context, h types.Metric) {
	m := s.getOrCreateData(h)

	m.lock.Lock()
	m.cells = make(map[cellKey][]*types.CellData)
	m.lock.Unlock()
}
