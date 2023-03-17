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
	"bytes"

	"libvirt.org/go/libvirtxml"
)

const (
	// some sensible value for the size of the logging volume
	maxLoggingVolumeSize = uint64(2 * 1024 * 1024)
)

// createLoggingVolumeDef creates the XML for the logging
func createLoggingVolumeDef(name string) *libvirtxml.StorageVolume {
	return &libvirtxml.StorageVolume{
		Name: name,
		Capacity: &libvirtxml.StorageVolumeSize{
			Value: maxLoggingVolumeSize,
		},
	}
}

// CreateLoggingVolume creates a logging volume for the console log
func CreateLoggingVolume(client *LivirtClient) func(storagePool, name string) (*libvirtxml.StorageVolume, error) {
	conn := client.LibVirt
	storageVolByNameXMLDesc := getStorageVolByNameXMLDesc(conn)
	storageVolXMLDesc := getStorageVolXMLDesc(conn)

	return func(storagePool, name string) (*libvirtxml.StorageVolume, error) {
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return nil, err
		}
		// check if we already know the volume
		_, err = storageVolByNameXMLDesc(pool, name)
		if err == nil {
			// we need to delete the volume
			_, err := deleteStorageVol(conn)(pool, name)
			if err != nil {
				return nil, err
			}
		}
		// Refresh the pool
		err = refreshPool(conn)(pool)
		if err != nil {
			return nil, err
		}
		// new volume
		volumeDef := createLoggingVolumeDef(name)
		volumeDefXML, err := XMLMarshall(volumeDef)
		if err != nil {
			return nil, err
		}

		// create the volume
		volume, err := conn.StorageVolCreateXML(pool, string(volumeDefXML), 0)
		if err != nil {
			return nil, err
		}

		// refresh the description
		return storageVolXMLDesc(&volume)

	}
}

// GetLoggingVolume retrieves the value of the logging volume
func GetLoggingVolume(client *LivirtClient) func(storagePool, name string) (string, error) {
	conn := client.LibVirt

	return func(storagePool, name string) (string, error) {
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return "", err
		}
		vol, err := conn.StorageVolLookupByName(pool, name)
		if err != nil {
			return "", err
		}
		//
		// load the value of the logging volume
		var buffer bytes.Buffer
		err = conn.StorageVolDownload(vol, &buffer, 0, maxLoggingVolumeSize, 0)
		if err != nil {
			return "", err
		}
		// returns the content of the logs
		return buffer.String(), nil
	}
}
