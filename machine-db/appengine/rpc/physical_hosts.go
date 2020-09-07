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

// CreatePhysicalHost handles a request to create a new physical host.
func (*Service) CreatePhysicalHost(c context.Context, req *crimson.CreatePhysicalHostRequest) (*crimson.PhysicalHost, error) {
	return createPhysicalHost(c, req.Host)
}

// ListPhysicalHosts handles a request to list physical hosts.
func (*Service) ListPhysicalHosts(c context.Context, req *crimson.ListPhysicalHostsRequest) (*crimson.ListPhysicalHostsResponse, error) {
	hosts, err := listPhysicalHosts(c, database.Get(c), req)
	if err != nil {
		return nil, err
	}
	return &crimson.ListPhysicalHostsResponse{
		Hosts: hosts,
	}, nil
}

// UpdatePhysicalHost handles a request to update an existing physical host.
func (*Service) UpdatePhysicalHost(c context.Context, req *crimson.UpdatePhysicalHostRequest) (*crimson.PhysicalHost, error) {
	return updatePhysicalHost(c, req.Host, req.UpdateMask)
}

// createPhysicalHost creates a new physical host in the database.
func createPhysicalHost(c context.Context, h *crimson.PhysicalHost) (*crimson.PhysicalHost, error) {
	if err := validatePhysicalHostForCreation(h); err != nil {
		return nil, err
	}
	ip, _ := common.ParseIPv4(h.Ipv4)
	tx, err := database.Begin(c)
	if err != nil {
		return nil, errors.Annotate(err, "failed to begin transaction").Err()
	}
	defer tx.MaybeRollback(c)

	// TODO(smut): Support the case where the NIC already has a hostname and IP assigned.
	hostnameID, err := model.AssignHostnameAndIP(c, tx, h.Name, ip)
	if err != nil {
		return nil, err
	}

	// By setting hostname_id, machine_id, nic_id, and os_id as FOREIGN KEY and NOT NULL when setting up the
	// database, we can avoid checking if the given values are valid. MySQL will ensure the given values exist.
	res, err := tx.ExecContext(c, `
		INSERT INTO physical_hosts (
			hostname_id,
			machine_id,
			nic_id,
			os_id,
			vm_slots,
			virtual_datacenter,
			description,
			deployment_ticket
		)
		VALUES (
			?,
			(SELECT id FROM machines WHERE name = ?),
			(SELECT n.id FROM machines m, nics n WHERE n.machine_id = m.id AND m.name = ? AND n.name = ? AND n.hostname_id IS NULL),
			(SELECT id FROM oses WHERE name = ?),
			?,
			?,
			?,
			?
		)
	`, hostnameID, h.Machine, h.Machine, h.Nic, h.Os, h.VmSlots, h.VirtualDatacenter, h.Description, h.DeploymentTicket)
	if err != nil {
		switch e, ok := err.(*mysql.MySQLError); {
		case !ok:
			// Type assertion failed.
		case e.Number == mysqlerr.ER_DUP_ENTRY && strings.Contains(e.Message, "'machine_id'"):
			// e.g. "Error 1062: Duplicate entry '1' for key 'machine_id'".
			return nil, status.Errorf(codes.AlreadyExists, "duplicate physical host for machine %q", h.Machine)
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'machine_id'"):
			// e.g. "Error 1048: Column 'machine_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "machine %q does not exist", h.Machine)
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'nic_id'"):
			// e.g. "Error 1048: Column 'nic_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "NIC %q of machine %q does not exist or already has a hostname", h.Nic, h.Machine)
		case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'os_id'"):
			// e.g. "Error 1048: Column 'os_id' cannot be null".
			return nil, status.Errorf(codes.NotFound, "operating system %q does not exist", h.Os)
		}
		return nil, errors.Annotate(err, "failed to create physical host").Err()
	}
	hostID, err := res.LastInsertId()
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch physical host").Err()
	}

	// Hostname is also stored with the backing NIC.
	_, err = tx.ExecContext(c, `
		UPDATE nics
		SET hostname_id = ?
		WHERE id = (SELECT nic_id FROM physical_hosts WHERE id = ?)
	`, hostnameID, hostID)
	if err != nil {
		return nil, errors.Annotate(err, "failed to update NIC").Err()
	}

	// Physical host state is stored with the backing machine. Update if necessary.
	if h.State != states.State_STATE_UNSPECIFIED {
		_, err := tx.ExecContext(c, `
			UPDATE machines
			SET state = ?
			WHERE name = ?
		`, h.State, h.Machine)
		if err != nil {
			return nil, errors.Annotate(err, "failed to update machine").Err()
		}
	}

	hosts, err := listPhysicalHosts(c, tx, &crimson.ListPhysicalHostsRequest{
		Names: []string{h.Name},
	})
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch created physical host").Err()
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Annotate(err, "failed to commit transaction").Err()
	}
	return hosts[0], nil
}

// listPhysicalHosts returns a slice of physical hosts in the database.
func listPhysicalHosts(c context.Context, q database.QueryerContext, req *crimson.ListPhysicalHostsRequest) ([]*crimson.PhysicalHost, error) {
	mac48s, err := parseMAC48s(req.MacAddresses)
	if err != nil {
		return nil, err
	}
	ipv4s, err := parseIPv4s(req.Ipv4S)
	if err != nil {
		return nil, err
	}

	stmt := squirrel.Select(
		"hp.name",
		"hp.vlan_id",
		"m.name",
		"n.name",
		"n.mac_address",
		"o.name",
		"h.vm_slots",
		"h.virtual_datacenter",
		"h.description",
		"h.deployment_ticket",
		"i.ipv4",
		"m.state",
	)
	stmt = stmt.From("(physical_hosts h, hostnames hp, machines m, nics n, oses o, ips i)")
	if len(req.Datacenters) > 0 {
		stmt = stmt.Join("racks r ON m.rack_id = r.id")
		stmt = stmt.Join("datacenters d ON r.datacenter_id = d.id")
	} else if len(req.Racks) > 0 {
		stmt = stmt.Join("racks r ON m.rack_id = r.id")
	}
	if len(req.Platforms) > 0 {
		stmt = stmt.Join("platforms p ON m.platform_id = p.id")
	}
	stmt = stmt.Where("n.hostname_id = hp.id").
		Where("h.machine_id = m.id").
		Where("h.nic_id = n.id").
		Where("h.os_id = o.id").
		Where("i.hostname_id = hp.id")
	stmt = selectInString(stmt, "hp.name", req.Names)
	stmt = selectInInt64(stmt, "hp.vlan_id", req.Vlans)
	stmt = selectInString(stmt, "m.name", req.Machines)
	stmt = selectInString(stmt, "n.name", req.Nics)
	stmt = selectInUint64(stmt, "n.mac_address", mac48s)
	stmt = selectInString(stmt, "o.name", req.Oses)
	stmt = selectInString(stmt, "h.virtual_datacenter", req.VirtualDatacenters)
	stmt = selectInInt64(stmt, "i.ipv4", ipv4s)
	stmt = selectInState(stmt, "m.state", req.States)
	stmt = selectInString(stmt, "d.name", req.Datacenters)
	stmt = selectInString(stmt, "r.name", req.Racks)
	stmt = selectInString(stmt, "p.name", req.Platforms)
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Annotate(err, "failed to generate statement").Err()
	}

	rows, err := q.QueryContext(c, query, args...)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch physical hosts").Err()
	}
	defer rows.Close()
	var hosts []*crimson.PhysicalHost
	for rows.Next() {
		h := &crimson.PhysicalHost{}
		var mac48 common.MAC48
		var ipv4 common.IPv4
		if err = rows.Scan(
			&h.Name,
			&h.Vlan,
			&h.Machine,
			&h.Nic,
			&mac48,
			&h.Os,
			&h.VmSlots,
			&h.VirtualDatacenter,
			&h.Description,
			&h.DeploymentTicket,
			&ipv4,
			&h.State,
		); err != nil {
			return nil, errors.Annotate(err, "failed to fetch physical host").Err()
		}
		h.MacAddress = mac48.String()
		h.Ipv4 = ipv4.String()
		hosts = append(hosts, h)
	}
	return hosts, nil
}

// updatePhysicalHost updates an existing physical host in the database.
func updatePhysicalHost(c context.Context, h *crimson.PhysicalHost, mask *field_mask.FieldMask) (*crimson.PhysicalHost, error) {
	if err := validatePhysicalHostForUpdate(h, mask); err != nil {
		return nil, err
	}
	update := false
	updateState := false
	stmt := squirrel.Update("physical_hosts")
	for _, path := range mask.Paths {
		switch path {
		case "machine":
			stmt = stmt.Set("machine_id", squirrel.Expr("(SELECT id FROM machines WHERE name = ?)", h.Machine))
			update = true
		case "os":
			stmt = stmt.Set("os_id", squirrel.Expr("(SELECT id FROM oses WHERE name = ?)", h.Os))
			update = true
		case "state":
			updateState = true
		case "vm_slots":
			stmt = stmt.Set("vm_slots", h.VmSlots)
			update = true
		case "virtual_datacenter":
			stmt = stmt.Set("virtual_datacenter", h.VirtualDatacenter)
			update = true
		case "description":
			stmt = stmt.Set("description", h.Description)
			update = true
		case "deployment_ticket":
			stmt = stmt.Set("deployment_ticket", h.DeploymentTicket)
			update = true
		}
	}
	var query string
	var args []interface{}
	var err error
	if update {
		stmt = stmt.Where("hostname_id = (SELECT id FROM hostnames WHERE name = ?)", h.Name)
		query, args, err = stmt.ToSql()
		if err != nil {
			return nil, errors.Annotate(err, "failed to generate statement").Err()
		}
	}

	tx, err := database.Begin(c)
	if err != nil {
		return nil, errors.Annotate(err, "failed to begin transaction").Err()
	}
	defer tx.MaybeRollback(c)

	if query != "" && len(args) > 0 {
		_, err = tx.ExecContext(c, query, args...)
		if err != nil {
			switch e, ok := err.(*mysql.MySQLError); {
			case !ok:
				// Type assertion failed.
			case e.Number == mysqlerr.ER_DUP_ENTRY && strings.Contains(e.Message, "'machine_id'"):
				// e.g. "Error 1062: Duplicate entry '1' for key 'machine_id'".
				return nil, status.Errorf(codes.AlreadyExists, "duplicate physical host for machine %q", h.Machine)
			case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'machine_id'"):
				// e.g. "Error 1048: Column 'machine_id' cannot be null".
				return nil, status.Errorf(codes.NotFound, "machine %q does not exist", h.Machine)
			case e.Number == mysqlerr.ER_BAD_NULL_ERROR && strings.Contains(e.Message, "'os_id'"):
				// e.g. "Error 1048: Column 'os_id' cannot be null".
				return nil, status.Errorf(codes.NotFound, "operating system %q does not exist", h.Os)
			}
			return nil, errors.Annotate(err, "failed to update physical host").Err()
		}
		// The number of rows affected cannot distinguish between zero because the physical host didn't exist
		// and zero because the row already matched, so skip looking at the number of rows affected.
	}
	if updateState {
		_, err = tx.ExecContext(c, `
			UPDATE machines
			SET state = ?
			WHERE id = (SELECT machine_id FROM physical_hosts WHERE hostname_id = (SELECT id FROM hostnames WHERE name = ?))
		`, h.State, h.Name)
		if err != nil {
			return nil, errors.Annotate(err, "failed to update machine").Err()
		}
	}

	hosts, err := listPhysicalHosts(c, tx, &crimson.ListPhysicalHostsRequest{
		Names: []string{h.Name},
	})
	switch {
	case err != nil:
		return nil, errors.Annotate(err, "failed to fetch updated physical host").Err()
	case len(hosts) == 0:
		return nil, status.Errorf(codes.NotFound, "physical host %q does not exist", h.Name)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Annotate(err, "failed to commit transaction").Err()
	}
	return hosts[0], nil
}

// validatePhysicalHostForCreation validates a physical host for creation.
func validatePhysicalHostForCreation(h *crimson.PhysicalHost) error {
	switch {
	case h == nil:
		return status.Error(codes.InvalidArgument, "physical host specification is required")
	case h.Name == "":
		return status.Error(codes.InvalidArgument, "hostname is required and must be non-empty")
	case h.Vlan != 0:
		return status.Error(codes.InvalidArgument, "VLAN must not be specified, use IP address instead")
	case h.Machine == "":
		return status.Error(codes.InvalidArgument, "machine is required and must be non-empty")
	case h.Nic == "":
		return status.Error(codes.InvalidArgument, "NIC is required and must be non-empty")
	case h.MacAddress != "":
		return status.Error(codes.InvalidArgument, "MAC address must not be specified, use NIC instead")
	case h.Os == "":
		return status.Error(codes.InvalidArgument, "operating system is required and must be non-empty")
	case h.VmSlots < 0:
		return status.Error(codes.InvalidArgument, "VM slots must be non-negative")
	default:
		_, err := common.ParseIPv4(h.Ipv4)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid IPv4 address %q", h.Ipv4)
		}
		return nil
	}
}

// validatePhysicalHostForUpdate validates a physical host for update.
func validatePhysicalHostForUpdate(h *crimson.PhysicalHost, mask *field_mask.FieldMask) error {
	switch err := validateUpdateMask(mask); {
	case h == nil:
		return status.Error(codes.InvalidArgument, "physical host specification is required")
	case h.Name == "":
		return status.Error(codes.InvalidArgument, "hostname is required and must be non-empty")
	case err != nil:
		return err
	}
	for _, path := range mask.Paths {
		// TODO(smut): Allow NIC, IPv4 address and state to be updated.
		switch path {
		case "name":
			return status.Error(codes.InvalidArgument, "hostname cannot be updated, delete and create a new physical host instead")
		case "vlan":
			return status.Error(codes.InvalidArgument, "VLAN cannot be updated, delete and create a new physical host instead")
		case "machine":
			if h.Machine == "" {
				return status.Error(codes.InvalidArgument, "machine is required and must be non-empty")
			}
		case "nic":
			return status.Error(codes.InvalidArgument, "NIC cannot be updated, delete and create a new physical host instead")
		case "mac_address":
			return status.Error(codes.InvalidArgument, "MAC address cannot be updated, update NIC instead")
		case "os":
			if h.Os == "" {
				return status.Error(codes.InvalidArgument, "operating system is required and must be non-empty")
			}
		case "state":
			if h.State == states.State_STATE_UNSPECIFIED {
				return status.Error(codes.InvalidArgument, "state is required")
			}
		case "vm_slots":
			if h.VmSlots < 0 {
				return status.Error(codes.InvalidArgument, "VM slots must be non-negative")
			}
		case "virtual_datacenter":
			// Empty virtual datacenter is allowed, nothing to validate.
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
