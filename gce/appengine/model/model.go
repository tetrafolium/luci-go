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

// Package model contains datastore model definitions.
package model

import (
	"fmt"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/gce/api/config/v1"
	"github.com/tetrafolium/luci-go/gce/api/projects/v1"
)

// ConfigKind is a config entity's kind in the datastore.
const ConfigKind = "Config"

// Config is a root entity representing a config for one type of VMs.
// VM entities should be created for each config entity.
type Config struct {
	// _extra is where unknown properties are put into memory.
	// Extra properties are not written to the datastore.
	_extra datastore.PropertyMap `gae:"-,extra"`
	// _kind is the entity's kind in the datastore.
	_kind string `gae:"$kind,Config"`
	// ID is the unique identifier for this config.
	ID string `gae:"$id"`
	// Config is the config.Config representation of this entity.
	// Indexing is not useful here since this field contains textproto.
	// Additionally, indexed string fields are limited to 1500 bytes.
	// https://cloud.google.com/datastore/docs/concepts/limits.
	// noindex is not respected here. See config.Config.ToProperty.
	Config config.Config `gae:"binary_config,noindex"`
}

// ProjectKind is a project entity's kind in the datastore.
const ProjectKind = "Project"

// Project is a root entity representing a GCP project.
// GCE quota utilization is reported for each metric in each region.
type Project struct {
	// _extra is where unknown properties are put into memory.
	// Extra properties are not written to the datastore.
	_extra datastore.PropertyMap `gae:"-,extra"`
	// _kind is the entity's kind in the datastore.
	_kind string `gae:"$kind,Project"`
	// ID is the unique identifier for this project.
	ID string `gae:"$id"`
	// Config is the projects.Config representation of this entity.
	// noindex is not respected here. See projects.Config.ToProperty.
	Config projects.Config `gae:"binary_config,noindex"`
}

// VMKind is a VM entity's kind in the datastore.
const VMKind = "VM"

// NetworkInterface is a network interface attached to a GCE instance.
type NetworkInterface struct {
	// ExternalIP is an external network address assigned to a GCE instance.
	// GCE currently supports at most one external IP address per network
	// interface.
	ExternalIP string
	// Internal is an internal network address assigned to a GCE instance.
	InternalIP string
}

// VM is a root entity representing a configured VM.
// GCE instances should be created for each VM entity.
type VM struct {
	// _extra is where unknown properties are put into memory.
	// Extra properties are not written to the datastore.
	_extra datastore.PropertyMap `gae:"-,extra"`
	// _kind is the entity's kind in the datastore.
	_kind string `gae:"$kind,VM"`
	// ID is the unique identifier for this VM.
	ID string `gae:"$id"`
	// Attributes is the config.VM describing the GCE instance to create.
	// Indexing is not useful here since this field contains textproto.
	// noindex is not respected here. See config.VM.ToProperty.
	Attributes config.VM `gae:"binary_attributes,noindex"`
	// AttributesIndexed is a slice of strings in "key:value" form where the key is
	// the path to a field in Attributes and the value is its associated value.
	// Allows fields from Attributes to be indexed.
	AttributesIndexed []string `gae:"attributes_indexed"`
	// Config is the ID of the config this VM was created from.
	Config string `gae:"config"`
	// Configured is the Unix time when the GCE instance was configured.
	Configured int64 `gae:"configured"`
	// Connected is the Unix time when the GCE instance connected to Swarming.
	Connected int64 `gae:"connected"`
	// Created is the Unix time when the GCE instance was created.
	Created int64 `gae:"created"`
	// Drained indicates whether or not this VM is drained.
	// A GCE instance should not be created for a drained VM.
	// Any existing GCE instance should be deleted regardless of deadline.
	Drained bool `gae:"drained"`
	// Hostname is the short hostname of the GCE instance to create.
	Hostname string `gae:"hostname"`
	// Image is the source image for the boot disk of the GCE instance.
	Image string `gae:"image"`
	// Index is this VM's number with respect to its config.
	Index int32 `gae:"index"`
	// Lifetime is the number of seconds the GCE instance should live for.
	Lifetime int64 `gae:"lifetime"`
	// NetworkInterfaces is a slice of network interfaces attached to this created
	// GCE instance. Empty if the instance is not yet created.
	NetworkInterfaces []NetworkInterface `gae:"network_interfaces"`
	// Prefix is the prefix to use when naming the GCE instance.
	Prefix string `gae:"prefix"`
	// Revision is the config revision this VM was created from.
	Revision string `gae:"revision"`
	// Swarming is hostname of the Swarming server the GCE instance connects to.
	Swarming string `gae:"swarming"`
	// Timeout is the number of seconds the GCE instance has to connect to Swarming.
	Timeout int64 `gae:"timeout"`
	// URL is the URL of the created GCE instance.
	URL string `gae:"url"`
}

// IndexAttributes sets indexable fields of vm.Attributes in AttributesIndexed.
func (vm *VM) IndexAttributes() {
	vm.AttributesIndexed = make([]string, len(vm.Attributes.Disk))
	for i, d := range vm.Attributes.Disk {
		vm.AttributesIndexed[i] = fmt.Sprintf("disk.image:%s", d.GetImageBase())
	}
}

// getDisks returns a []*compute.AttachedDisk representation of this VM's disks.
func (vm *VM) getDisks() []*compute.AttachedDisk {
	if len(vm.Attributes.GetDisk()) == 0 {
		return nil
	}
	disks := make([]*compute.AttachedDisk, len(vm.Attributes.Disk))
	for i, disk := range vm.Attributes.Disk {
		disks[i] = &compute.AttachedDisk{
			// AutoDelete deletes the disk when the instance is deleted.
			AutoDelete: true,
			InitializeParams: &compute.AttachedDiskInitializeParams{
				DiskSizeGb:  disk.Size,
				DiskType:    disk.Type,
				SourceImage: disk.Image,
			},
			Interface: disk.GetInterface().String(),
		}
		if disk.IsScratchDisk() {
			disks[i].Type = "SCRATCH"
		}
	}
	// GCE requires the first disk to be the boot disk.
	disks[0].Boot = true
	return disks
}

// getMetadata returns a *compute.Metadata representation of this VM's metadata.
func (vm *VM) getMetadata() *compute.Metadata {
	if len(vm.Attributes.GetMetadata()) == 0 {
		return nil
	}
	meta := &compute.Metadata{
		Items: make([]*compute.MetadataItems, len(vm.Attributes.Metadata)),
	}
	for i, data := range vm.Attributes.Metadata {
		// Implicitly rejects FromFile, which is only supported in configs.
		spl := strings.SplitN(data.GetFromText(), ":", 2)
		// Per strings.SplitN semantics, len(spl) > 0 when splitting on a non-empty separator.
		// Therefore we can be sure the spl[0] exists (even if it's an empty string).
		key := spl[0]
		var val *string
		if len(spl) > 1 {
			val = &spl[1]
		}
		meta.Items[i] = &compute.MetadataItems{
			Key:   key,
			Value: val,
		}
	}
	return meta
}

// getNetworkInterfaces returns a []*compute.NetworkInterface representation of this VM's network interfaces.
func (vm *VM) getNetworkInterfaces() []*compute.NetworkInterface {
	if len(vm.Attributes.GetNetworkInterface()) == 0 {
		return nil
	}
	nics := make([]*compute.NetworkInterface, len(vm.Attributes.NetworkInterface))
	for i, nic := range vm.Attributes.NetworkInterface {
		nics[i] = &compute.NetworkInterface{
			Network: nic.Network,
		}
		if len(nic.GetAccessConfig()) > 0 {
			nics[i].AccessConfigs = make([]*compute.AccessConfig, len(nic.AccessConfig))
			for j, cfg := range nic.AccessConfig {
				nics[i].AccessConfigs[j] = &compute.AccessConfig{
					Type: cfg.Type.String(),
				}
			}
		}
	}
	return nics
}

// getServiceAccounts returns a []*compute.ServiceAccount representation of this VM's service accounts.
func (vm *VM) getServiceAccounts() []*compute.ServiceAccount {
	if len(vm.Attributes.GetServiceAccount()) == 0 {
		return nil
	}
	accts := make([]*compute.ServiceAccount, len(vm.Attributes.ServiceAccount))
	for i, sa := range vm.Attributes.ServiceAccount {
		accts[i] = &compute.ServiceAccount{
			Email: sa.Email,
		}
		if len(sa.GetScope()) > 0 {
			accts[i].Scopes = make([]string, len(sa.Scope))
			for j, s := range sa.Scope {
				accts[i].Scopes[j] = s
			}
		}
	}
	return accts
}

// getTags returns a *compute.Tags representation of this VM's tags.
func (vm *VM) getTags() *compute.Tags {
	if len(vm.Attributes.GetTag()) == 0 {
		return nil
	}
	tags := &compute.Tags{
		Items: make([]string, len(vm.Attributes.Tag)),
	}
	for i, tag := range vm.Attributes.Tag {
		tags.Items[i] = tag
	}
	return tags
}

// GetInstance returns a *compute.Instance representation of this VM.
func (vm *VM) GetInstance() *compute.Instance {
	inst := &compute.Instance{
		Name:              vm.Hostname,
		Disks:             vm.getDisks(),
		MachineType:       vm.Attributes.GetMachineType(),
		Metadata:          vm.getMetadata(),
		MinCpuPlatform:    vm.Attributes.GetMinCpuPlatform(),
		NetworkInterfaces: vm.getNetworkInterfaces(),
		ServiceAccounts:   vm.getServiceAccounts(),
		Tags:              vm.getTags(),
	}
	return inst
}
