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
	"log"
	"net/http"
	"time"

	"libvirt.org/go/libvirtxml"
)

// checks if the file needs update
func needsUpdateFromURL(url string, vol *libvirtxml.StorageVolume) bool {
	// access some typical metadata
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return true
	}
	if vol.Target.Timestamps != nil && len(vol.Target.Timestamps.Mtime) > 0 {
		// add modified header
		req.Header.Set("If-Modified-Since", timeFromEpoch(vol.Target.Timestamps.Mtime).UTC().Format(http.TimeFormat))
	}
	// send
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return true
	}
	if resp.StatusCode == http.StatusNotModified {
		return false
	}
	// get size
	volSize, ok := getVolumeSize(vol)
	// check the size
	remoteSize := resp.ContentLength
	if remoteSize > 0 && ok && remoteSize == int64(volSize) {
		return false
	}
	return true
}

// CloneBootDisk will clone an existing (boot) disk, so the clone may safely be modified
func CloneBootDisk(client *LivirtClient) func(storagePool string, existingVolumeXML *libvirtxml.StorageVolume, newName string) (*libvirtxml.StorageVolume, error) {
	conn := client.LibVirt
	storageVolByNameXMLDesc := getStorageVolByNameXMLDesc(conn)
	storageVolXMLDesc := getStorageVolXMLDesc(conn)

	return func(storagePool string, existingVolumeXML *libvirtxml.StorageVolume, newName string) (*libvirtxml.StorageVolume, error) {
		// some logging
		log.Printf("Cloning boot disk [%s] available on pool [%s] into [%s] ...", existingVolumeXML.Name, storagePool, newName)
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return nil, err
		}
		// lookup volume by key
		existingVol, err := conn.StorageVolLookupByKey(existingVolumeXML.Key)
		if err != nil {
			return nil, err
		}
		// check if we already know the new volume
		_, err = storageVolByNameXMLDesc(pool, newName)
		if err == nil {
			// we need to delete the volume
			_, err := deleteStorageVol(conn)(pool, newName)
			if err != nil {
				return nil, err
			}
		}
		// Refresh the pool
		err = refreshPool(conn)(pool)
		if err != nil {
			return nil, err
		}

		// update the volume identifier
		volumeDef := createDefaultVolume()
		volumeDef.Name = newName
		volumeDef.Allocation = existingVolumeXML.Allocation
		volumeDef.Capacity = existingVolumeXML.Capacity
		volumeDef.Physical = existingVolumeXML.Physical
		volumeDef.Target = existingVolumeXML.Target

		volumeDefXML, err := XMLMarshall(volumeDef)
		if err != nil {
			return nil, err
		}

		t0 := time.Now()
		log.Printf("Starting clone of [%s] to pool [%s], size=[%d bytes]...", existingVolumeXML.Name, pool.Name, volumeDef.Capacity.Value)

		// create the volume
		clonedVolume, err := conn.StorageVolCreateXMLFrom(pool, string(volumeDefXML), existingVol, 0)
		if err != nil {
			return nil, err
		}
		t1 := time.Now()
		log.Printf("Clone of [%s] to pool [%s] done in [%f s].", existingVolumeXML.Name, pool.Name, t1.Sub(t0).Seconds())

		// Refresh the pool
		err = refreshPool(conn)(pool)
		if err != nil {
			return nil, err
		}

		// refresh the description
		return storageVolXMLDesc(&clonedVolume)
	}
}

// UploadBootDisk uploads the iso file to the remote storage pool
func UploadBootDisk(client *LivirtClient) func(storagePool, name, url string) (*libvirtxml.StorageVolume, error) {
	conn := client.LibVirt
	// hooks
	storageVolXMLDesc := getStorageVolByNameXMLDesc(conn)
	return func(storagePool, name, url string) (*libvirtxml.StorageVolume, error) {
		// some logging
		log.Printf("Make boot disk [%s] available on pool [%s] ...", name, storagePool)
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return nil, err
		}
		// check if we already know the volume
		existing, err := storageVolXMLDesc(pool, name)
		if err == nil {
			// maybe there is no need for an update
			if !needsUpdateFromURL(url, existing) {
				log.Println("Skipping upload, image is already available.")
				return existing, nil
			}
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
		// get some metadata
		resp, err := http.Get(url) // #nosec G107 - we do want the URL to come from config
		if err != nil {
			return nil, err
		}
		defer safeClose(resp.Body)
		size := uint64(resp.ContentLength)
		// update the volume identifier
		volumeDef := createDefaultVolume()
		volumeDef.Name = name
		volumeDef.Capacity.Unit = "B"
		volumeDef.Capacity.Value = size
		volumeDef.Target.Format.Type = "qcow2"

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
		log.Printf("Starting upload of [%s] to pool [%s], size=[%d bytes]...", url, pool.Name, size)

		err = conn.StorageVolUpload(volume, createReaderWithLog(resp.Body, size), 0, size, 0)
		if err != nil {
			return nil, err
		}
		t1 := time.Now()
		log.Printf("Upload of [%s] to pool [%s] done in [%f s].", url, pool.Name, t1.Sub(t0).Seconds())

		// Refresh the pool
		err = refreshPool(conn)(pool)
		if err != nil {
			return nil, err
		}

		// refresh the description
		return storageVolXMLDesc(pool, name)
	}
}

// CreateBootDiskXML creates the XML for the boot disk
func CreateBootDiskXML(client *LivirtClient) func(key string) (*libvirtxml.DomainDisk, error) {
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
				Dev: "vda",
				Bus: "virtio",
			},
			Driver: &libvirtxml.DomainDiskDriver{
				Name:  "qemu",
				Type:  "qcow2",
				IOMMU: "on",
			},
			Source: &libvirtxml.DomainDiskSource{
				File: &libvirtxml.DomainDiskSourceFile{
					File: diskVolumeFile,
				},
			},
			Boot: &libvirtxml.DomainDeviceBoot{
				Order: 1,
			},
		}, nil
	}
}
