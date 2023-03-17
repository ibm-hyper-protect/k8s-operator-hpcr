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
	"log"
	"time"

	"github.com/kdomanski/iso9660"
	"libvirt.org/go/libvirtxml"
)

type CloudInit struct {
	UserData string `json:"user_data" yaml:"user_data"`
	MetaData string `json:"meta_data" yaml:"meta_data"`
}

// CreateCloudInitDisk creates the XML for the cloud init disk
func CreateCloudInitDisk(client *LivirtClient) func(key string) (*libvirtxml.DomainDisk, error) {
	conn := client.LibVirt

	return func(key string) (*libvirtxml.DomainDisk, error) {

		diskVolume, err := conn.StorageVolLookupByKey(key)
		if err != nil {
			return nil, err
		}
		diskVolumeFile, err := conn.StorageVolGetPath(diskVolume)
		if err != nil {
			return nil, err
		}

		return &libvirtxml.DomainDisk{
			Device: "disk",
			Target: &libvirtxml.DomainDiskTarget{
				Dev: "vdb",
				Bus: "virtio",
			},
			Driver: &libvirtxml.DomainDiskDriver{
				Name:  "qemu",
				Type:  "raw",
				IOMMU: "on",
			},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{
					File: diskVolumeFile,
				},
			},
		}, nil
	}
}

// CreateCloudInit produces a cloud init ISO file as a data blob with a userdata and a metadata section
func CreateCloudInit(userData, metaData []byte) ([]byte, error) {
	writer, err := iso9660.NewWriter()
	if err != nil {
		return nil, err
	}
	defer writer.Cleanup()

	err = writer.AddFile(bytes.NewReader(userData), userDataFilename)
	if err != nil {
		return nil, err
	}

	err = writer.AddFile(bytes.NewReader(metaData), metaDataFilename)
	if err != nil {
		return nil, err
	}

	err = writer.AddFile(bytes.NewReader([]byte{}), vendorDataFilename)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	err = writer.WriteTo(&buf, ciDataVolumeName)
	if err != nil {
		return nil, err
	}

	// done
	return buf.Bytes(), nil
}

// UploadCloudInit uploads the iso file to the remote storage pool
func UploadCloudInit(client *LivirtClient) func(storagePool, name string, isoData []byte) (*libvirtxml.StorageVolume, error) {
	conn := client.LibVirt
	// hooks
	storageVolXMLDesc := getStorageVolByNameXMLDesc(conn)
	// target path
	return func(storagePool, name string, isoData []byte) (*libvirtxml.StorageVolume, error) {
		// some logging
		log.Printf("Make cloud init file [%s] available on pool [%s] ...", name, storagePool)
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return nil, err
		}
		// check if we already know the volume
		_, err = storageVolXMLDesc(pool, name)
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
		// some metadata
		size := uint64(len(isoData))
		// update the volume identifier
		volumeDef := createDefaultVolume()
		volumeDef.Name = name
		volumeDef.Capacity.Unit = "B"
		volumeDef.Capacity.Value = size
		volumeDef.Target.Format.Type = "raw"

		volumeDefXML, err := XMLMarshall(volumeDef)
		if err != nil {
			return nil, err
		}

		// create the volume
		volume, err := conn.StorageVolCreateXML(pool, string(volumeDefXML), 0)
		if err != nil {
			return nil, err
		}

		t0 := time.Now()
		log.Printf("Starting upload of [%s] to pool [%s], size=[%d bytes]...", name, pool.Name, size)

		err = conn.StorageVolUpload(volume, bytes.NewReader(isoData), 0, size, 0)
		if err != nil {
			return nil, err
		}
		t1 := time.Now()
		log.Printf("Upload of [%s] to pool [%s] done in [%f s].", name, pool.Name, t1.Sub(t0).Seconds())

		// Refresh the pool
		err = refreshPool(conn)(pool)
		if err != nil {
			return nil, err
		}

		// refresh the description
		return storageVolXMLDesc(pool, name)
	}

}

// RemoveCloudInit removes the cloud init data from the storage pool
func RemoveCloudInit(client *LivirtClient) func(key string) error {
	return deleteVolumeByKey(client)
}
