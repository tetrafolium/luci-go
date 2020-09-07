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

package rpc

import (
	"context"
	"strings"

	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Masterminds/squirrel"
	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"

	"github.com/tetrafolium/luci-go/common/errors"

	states "github.com/tetrafolium/luci-go/machine-db/api/common/v1"
	"github.com/tetrafolium/luci-go/machine-db/api/crimson/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/database"
	"github.com/tetrafolium/luci-go/machine-db/appengine/model"
	"github.com/tetrafolium/luci-go/machine-db/common"
)

// CreateVM handles a request to create a new VM.
func (*Service) CreateVM(c context.Context, req *crimson.CreateVMRequest) (*crimson.VM, error) {
	return createVM(c, req.Vm)
}

// ListVMs handles a request to list VMs.
func (*Service) ListVMs(c context.Context, req *crimson.ListVMsRequest) (*crimson.ListVMsResponse, error) {
	vms, err := listVMs(c, database.Get(c), req)
	if err != nil {
		return nil, err
	}
	return &crimson.ListVMsResponse{
		Vms: vms,
	}, nil
}

// UpdateVM handles a request to update an existing VM.
func (*Service) UpdateVM(c context.Context, req *crimson.UpdateVMRequest) (*crimson.VM, error) {
	return updateVM(c, req.Vm, req.UpdateMask)
}

// createVM creates a new VM in the database.
func createVM(c context.Context, v *crimson.VM) (*crimson.VM, error) {
	if err := validateVMForCreation(v); err != nil {
		return nil, err
	}
	ip, _ := common.ParseIPv4(v.Ipv4)
	tx, err := database.Begin(c)
	if err != nil {
		return nil, errors.Annotate(err, "failed to begin transaction").Err()
	}
	defer tx.MaybeRollback(c)

	hostnameID, err := model.AssignHostnameAndIP(c, tx, v.Name, ip)
	if err != nil {
		return nil, err
	}

	// vms.hostname_id, vms.physical_host_id, and vms.os_id are NOT NULL as above.
	_, err = tx.ExecContext(c, `
		INSERT INTO vms (hostname_id, physical_host_id, os_id, description, deployment_ticket, state)
		VALUES (
			?,
			(SELECT p.id FROM physical_hosts p, hostnames h WHERE p.hostname_id = h.id AND h.name = ?),
			(SELECT id FROM oses WHERE name = ?),
			?,
			?,
			?
		)
	`, hostnameID, v.Host, v.Os, v.Description, v.DeploymentTicket, v.State)
	if err != nil {
		switch e, ok := err.(*mysql.MySQLError); {
		case !ok:
			// Type assertion failed.
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'physical_host_id'"):
			// e.g. "Error 1048: Column 'physical_host_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "physical host %q does not exist", v.Host)
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'os_id'"):
			// e.g. "Error 1048: Column 'os_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "operating system %q does not exist", v.Os)
		}
		return nil, errors.Annotate(err, "failed to create VM").Err()
	}

	vms, err := listVMs(c, tx, &crimson.ListVMsRequest{
		Names: []string{v.Name},
	})
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch created VM").Err()
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Annotate(err, "failed to commit transaction").Err()
	}
	return vms[0], nil
}

// listVMs returns a slice of VMs in the database.
func listVMs(c context.Context, q database.QueryerContext, req *crimson.ListVMsRequest) ([]*crimson.VM, error) {
	ipv4s, err := parseIPv4s(req.Ipv4S)
	if err != nil {
		return nil, err
	}

	stmt := squirrel.Select(
		"hv.name",
		"hv.vlan_id",
		"hp.name",
		"hp.vlan_id",
		"o.name",
		"v.description",
		"v.deployment_ticket",
		"i.ipv4",
		"v.state",
	)
	stmt = stmt.From("vms v, hostnames hv, physical_hosts p, hostnames hp, oses o, ips i").
		Where("v.hostname_id = hv.id").
		Where("v.physical_host_id = p.id").
		Where("p.hostname_id = hp.id").
		Where("v.os_id = o.id").
		Where("i.hostname_id = hv.id")
	stmt = selectInString(stmt, "hv.name", req.Names)
	stmt = selectInInt64(stmt, "hv.vlan_id", req.Vlans)
	stmt = selectInInt64(stmt, "i.ipv4", ipv4s)
	stmt = selectInString(stmt, "hp.name", req.Hosts)
	stmt = selectInInt64(stmt, "hp.vlan_id", req.HostVlans)
	stmt = selectInString(stmt, "o.name", req.Oses)
	stmt = selectInState(stmt, "v.state", req.States)
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Annotate(err, "failed to generate statement").Err()
	}

	rows, err := q.QueryContext(c, query, args...)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch VMs").Err()
	}
	defer rows.Close()
	var vms []*crimson.VM
	for rows.Next() {
		v := &crimson.VM{}
		var ipv4 common.IPv4
		if err = rows.Scan(
			&v.Name,
			&v.Vlan,
			&v.Host,
			&v.HostVlan,
			&v.Os,
			&v.Description,
			&v.DeploymentTicket,
			&ipv4,
			&v.State,
		); err != nil {
			return nil, errors.Annotate(err, "failed to fetch VM").Err()
		}
		v.Ipv4 = ipv4.String()
		vms = append(vms, v)
	}
	return vms, nil
}

// updateVM updates an existing VM in the database.
func updateVM(c context.Context, v *crimson.VM, mask *field_mask.FieldMask) (*crimson.VM, error) {
	if err := validateVMForUpdate(v, mask); err != nil {
		return nil, err
	}
	stmt := squirrel.Update("vms")
	updatedHost := false
	for _, path := range mask.Paths {
		switch path {
		case "host":
			if !updatedHost {
				stmt = stmt.Set("physical_host_id", squirrel.Expr("(SELECT id FROM physical_hosts WHERE hostname_id = (SELECT id FROM hostnames WHERE name = ?))", v.Host))
			}
			updatedHost = true
		case "os":
			stmt = stmt.Set("os_id", squirrel.Expr("(SELECT id FROM oses WHERE name = ?)", v.Os))
		case "state":
			stmt = stmt.Set("state", v.State)
		case "description":
			stmt = stmt.Set("description", v.Description)
		case "deployment_ticket":
			stmt = stmt.Set("deployment_ticket", v.DeploymentTicket)
		}
	}
	stmt = stmt.Where("hostname_id = (SELECT id FROM hostnames WHERE name = ?)", v.Name)
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Annotate(err, "failed to generate statement").Err()
	}

	tx, err := database.Begin(c)
	if err != nil {
		return nil, errors.Annotate(err, "failed to begin transaction").Err()
	}
	defer tx.MaybeRollback(c)

	_, err = tx.ExecContext(c, query, args...)
	if err != nil {
		switch e, ok := err.(*mysql.MySQLError); {
		case !ok:
			// Type assertion failed.
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'physical_host_id'"):
			// e.g. "Error 1048: Column 'physical_host_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "physical host %q does not exist", v.Host)
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'os_id'"):
			// e.g. "Error 1048: Column 'os_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "operating system %q does not exist", v.Os)
		}
		return nil, errors.Annotate(err, "failed to update VM").Err()
	}
	// The number of rows affected cannot distinguish between zero because the VM didn't exist
	// and zero because the row already matched, so skip looking at the number of rows affected.

	vms, err := listVMs(c, tx, &crimson.ListVMsRequest{
		Names: []string{v.Name},
	})
	switch {
	case err != nil:
		return nil, errors.Annotate(err, "failed to fetch updated VM").Err()
	case len(vms) == 0:
		return nil, status.Errorf(codes.NotFound, "VM %q does not exist", v.Name)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Annotate(err, "failed to commit transaction").Err()
	}
	return vms[0], nil
}

// validateVMForCreation validates a VM for creation.
func validateVMForCreation(v *crimson.VM) error {
	switch {
	case v == nil:
		return status.Error(codes.InvalidArgument, "VM specification is required")
	case v.Name == "":
		return status.Error(codes.InvalidArgument, "hostname is required and must be non-empty")
	case v.Vlan != 0:
		return status.Error(codes.InvalidArgument, "VLAN must not be specified, use IP address instead")
	case v.Host == "":
		return status.Error(codes.InvalidArgument, "physical hostname is required and must be non-empty")
	case v.HostVlan != 0:
		return status.Error(codes.InvalidArgument, "host VLAN must not be specified, use physical hostname instead")
	case v.Os == "":
		return status.Error(codes.InvalidArgument, "operating system is required and must be non-empty")
	case v.Ipv4 == "":
		return status.Error(codes.InvalidArgument, "IPv4 address is required and must be non-empty")
	case v.State == states.State_STATE_UNSPECIFIED:
		return status.Error(codes.InvalidArgument, "state is required")
	default:
		_, err := common.ParseIPv4(v.Ipv4)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid IPv4 address %q", v.Ipv4)
		}
		return nil
	}
}

// validateVMForUpdate validates a VM for update.
func validateVMForUpdate(v *crimson.VM, mask *field_mask.FieldMask) error {
	switch err := validateUpdateMask(mask); {
	case v == nil:
		return status.Error(codes.InvalidArgument, "VM specification is required")
	case v.Name == "":
		return status.Error(codes.InvalidArgument, "hostname is required and must be non-empty")
	case err != nil:
		return err
	}
	for _, path := range mask.Paths {
		// TODO(smut): Allow IPv4 address to be updated.
		switch path {
		case "name":
			return status.Error(codes.InvalidArgument, "hostname cannot be updated, delete and create a new VM instead")
		case "vlan":
			return status.Error(codes.InvalidArgument, "VLAN cannot be updated, delete and create a new VM instead")
		case "host":
			if v.Host == "" {
				return status.Error(codes.InvalidArgument, "physical hostname is required and must be non-empty")
			}
		case "host_vlan":
			return status.Error(codes.InvalidArgument, "host VLAN cannot be updated, update the host instead")
		case "os":
			if v.Os == "" {
				return status.Error(codes.InvalidArgument, "operating system is required and must be non-empty")
			}
		case "state":
			if v.State == states.State_STATE_UNSPECIFIED {
				return status.Error(codes.InvalidArgument, "state is required")
			}
		case "description":
			// Empty description is allowed, nothing to validate.
		case "deployment_ticket":
			// Empty deployment ticket is allowed, nothing to validate.
		default:
			return status.Errorf(codes.InvalidArgument, "unsupported update mask path %q", path)
		}
	}
	return nil
}
