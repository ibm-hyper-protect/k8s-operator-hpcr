// Copyright 2023 IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package datasource

package onprem

import (
	"fmt"
	"log"

	libvirt "github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

func createDefaultVolume() libvirtxml.StorageVolume {
	return libvirtxml.StorageVolume{
		Target: &libvirtxml.StorageVolumeTarget{
			Format: &libvirtxml.StorageVolumeTargetFormat{
				Type: "qcow2",
			},
			Permissions: &libvirtxml.StorageVolumeTargetPermissions{
				Mode: "644",
			},
		},
		Capacity: &libvirtxml.StorageVolumeSize{
			Unit:  "bytes",
			Value: 1,
		},
	}
}

func deleteStorageVol(conn *libvirt.Libvirt) func(pool libvirt.StoragePool, name string) (*libvirt.StorageVol, error) {
	return func(pool libvirt.StoragePool, name string) (*libvirt.StorageVol, error) {
		existing, err := conn.StorageVolLookupByName(pool, name)
		if err != nil {
			return nil, err
		}
		log.Printf("Deleting volume [%s] from pool [%s]...", name, pool.Name)
		return &existing, conn.StorageVolDelete(existing, 0)
	}
}

func getStorageVolByNameXMLDesc(conn *libvirt.Libvirt) func(pool libvirt.StoragePool, name string) (*libvirtxml.StorageVolume, error) {
	storageVolXMLDesc := getStorageVolXMLDesc(conn)
	return func(pool libvirt.StoragePool, name string) (*libvirtxml.StorageVolume, error) {
		existing, err := conn.StorageVolLookupByName(pool, name)
		if err != nil {
			return nil, err
		}
		return storageVolXMLDesc(&existing)
	}
}

func getStorageVolXMLDesc(conn *libvirt.Libvirt) func(vol *libvirt.StorageVol) (*libvirtxml.StorageVolume, error) {
	return func(vol *libvirt.StorageVol) (*libvirtxml.StorageVolume, error) {
		// try to get more info
		xmlDef, err := conn.StorageVolGetXMLDesc(*vol, 0)
		if err != nil {
			return nil, err
		}
		// parse
		return parseStorageVolumeXML(xmlDef)
	}
}

func getSizeFromStorageVolumeSize(s *libvirtxml.StorageVolumeSize) (uint64, bool) {
	if s == nil {
		return 0, false
	}
	unit := s.Unit
	if unit == "bytes" {
		return s.Value, true
	}
	log.Printf("Unknown unit [%s]\n", unit)
	return 0, false
}

func getVolumeSize(vol *libvirtxml.StorageVolume) (uint64, bool) {
	phys, ok := getSizeFromStorageVolumeSize(vol.Physical)
	if ok {
		return phys, ok
	}
	alloc, ok := getSizeFromStorageVolumeSize(vol.Allocation)
	if ok {
		return alloc, ok
	}
	return getSizeFromStorageVolumeSize(vol.Capacity)
}

// deleteVolumeByKey removes the volume identified by key from libvirt.
func deleteVolumeByKey(client *LivirtClient) func(key string) error {
	conn := client.LibVirt
	return func(key string) error {
		volume, err := conn.StorageVolLookupByKey(key)
		if err != nil {
			if isError(err, libvirt.ErrNoStorageVol) {
				// Volume already deleted.
				return nil
			}
			return err
		}

		// Refresh the pool of the volume so that libvirt knows it is
		// no longer in use.
		volPool, err := conn.StoragePoolLookupByVolume(volume)
		if err != nil {
			return err
		}

		if err := waitForSuccess("error refreshing pool for volume", func() error {
			return conn.StoragePoolRefresh(volPool, 0)
		}); err != nil {
			return err
		}

		// Workaround for redhat#1293804
		// https://bugzilla.redhat.com/show_bug.cgi?id=1293804#c12
		// Does not solve the problem but it makes it happen less often.
		_, err = conn.StorageVolGetXMLDesc(volume, 0)
		if err != nil {
			if !isError(err, libvirt.ErrNoStorageVol) {
				return err
			}
			// Volume is probably gone already, getting its XML description is pointless
		}

		err = conn.StorageVolDelete(volume, 0)
		if err != nil {
			if !isError(err, libvirt.ErrNoStorageVol) {
				return fmt.Errorf("can't delete volume %s: %w", key, err)
			}
			// Volume is gone already
			return nil
		}

		return err
	}
}
