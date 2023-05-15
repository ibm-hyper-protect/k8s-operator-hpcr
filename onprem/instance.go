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
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"path"

	"crypto/sha256"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

type InstanceMetadata struct {
	XMLName xml.Name `xml:"https://github.com/ibm-hyper-protect/k8s-operator-hpcr instance"`
	Hash    string   `xml:"hash"`
}

type AttachedDataDisk struct {
	// name of the attached data disk
	Name string
	// name of the libvirt storage pool, the pool must exist
	StoragePool string
}

type InstanceOptions struct {
	// name of the instance, will also be the hostname
	Name string
	// the userdata field
	UserData string
	// URL to the HPCR qcow2
	ImageURL string
	// name of the libvirt storage pool, the pool must exist
	StoragePool string
	// attached data disks
	DataDisks []*AttachedDataDisk
	// attached networks
	Networks []string
}

type DataDiskOptions struct {
	// name of the data disk
	Name string
	// name of the libvirt storage pool, the pool must exist
	StoragePool string
	// size of the disk
	Size uint64
}

type DataDiskRefOptions struct {
	// name of the data disk
	Name string
	// name of the libvirt storage pool, the pool must exist
	StoragePool string
}

type NetworkRefOptions struct {
	// name of the network
	Name string
}

// GetNetwork returns the network attached to the instane
func GetNetworks(opt *InstanceOptions) []string {
	if len(opt.Networks) == 0 {
		return []string{DefaultNetwork}
	}
	return opt.Networks
}

func GetCIDataVolumeName(name string) string {
	return fmt.Sprintf("cidata-%s.iso", name)
}

func GetBootVolumeName(name string) string {
	return fmt.Sprintf("boot-%s.qcow2", name)
}

func GetLoggingVolumeName(name string) string {
	return fmt.Sprintf("console-%s.log", name)
}

// createInstanceHash computes a hash value for the instance options
func CreateInstanceHash(opt *InstanceOptions) string {
	h := sha256.New()
	h.Write([]byte(opt.Name))
	h.Write([]byte(opt.ImageURL))
	h.Write([]byte(opt.StoragePool))
	h.Write([]byte(opt.UserData))
	// add the data disks to the mix
	for _, disk := range opt.DataDisks {
		h.Write([]byte(disk.Name))
		h.Write([]byte(disk.StoragePool))
	}
	// add the networks
	for _, network := range opt.Networks {
		h.Write([]byte(network))
	}
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}

func createMetaData(name string) []byte {
	return []byte(fmt.Sprintf("local-hostname: %s", name))
}

// IsInstanceValid tests if an instance has a valid configuration
func IsInstanceValid(client *LivirtClient) func(opt *InstanceOptions) (*libvirtxml.Domain, bool) {
	// connection
	conn := client.LibVirt

	return func(opt *InstanceOptions) (*libvirtxml.Domain, bool) {
		// instance name
		name := opt.Name
		// check for domain
		existing, err := conn.DomainLookupByName(name)
		if err != nil {
			return nil, false
		}
		// check if the instance is running
		state, _, err := conn.DomainGetState(existing, 0)
		if err != nil {
			log.Printf("Unable to get state for domain [%s], cause: [%v]", name, err)
			return nil, false
		}
		if libvirt.DomainState(state) != libvirt.DomainRunning {
			log.Printf("Domain [%s] is not running, instead it is in state [%d].", name, state)
			return nil, false
		}
		// get some more info
		existingStrg, err := conn.DomainGetXMLDesc(existing, 0)
		if err != nil {
			log.Printf("Unable to get domain description for domain [%s], cause: [%v]", name, err)
			return nil, false
		}
		// try to access metadata
		existingXML, err := parseDomainXML(existingStrg)
		if err != nil {
			log.Printf("Unable to parse domain XML for domain [%s], cause: [%v]", name, err)
			return nil, false
		}
		if existingXML.Metadata == nil {
			log.Printf("Domain [%s] does not have metadata", name)
		}
		// check the metadata
		metadata := InstanceMetadata{}
		err = xml.Unmarshal([]byte(existingXML.Metadata.XML), &metadata)
		if err != nil {
			log.Printf("Unable to parse metadata XML for domain [%s], cause: [%v]", name, err)
			return existingXML, false
		}
		// test the hash
		newHash := CreateInstanceHash(opt)
		if metadata.Hash == newHash {
			// nothing to do
			log.Printf("Domain [%s] is already up to date, hashes match.", name)
			return existingXML, true
		}
		// needs update
		log.Printf("Domain [%s] needs an update, hashes differ!", name)
		return existingXML, false
	}
}

// CreateInstanceSync (synchronously) creates an instance
func CreateInstanceSync(client *LivirtClient) func(opt *InstanceOptions) (*libvirtxml.Domain, error) {
	// some shortcuts
	uploadBootDisk := UploadBootDisk(client)
	cloneBootDisk := CloneBootDisk(client)
	uploadCloudInit := UploadCloudInit(client)

	createBootDisk := CreateBootDiskXML(client)
	createCloudInit := CreateCloudInitDisk(client)
	startDomain := StartDomain(client)
	deleteDomain := DeleteDomainByName(client)

	createLoggingVolume := CreateLoggingVolume(client)
	isInstanceValid := IsInstanceValid(client)
	createDataDiskXML := CreateDataDiskXML(client)

	createNetworksXML := CreateNetworksXML()

	return func(opt *InstanceOptions) (*libvirtxml.Domain, error) {
		// prepare some names
		name := opt.Name
		cidataName := GetCIDataVolumeName(name)
		bootName := GetBootVolumeName(name)
		logName := GetLoggingVolumeName(name)
		// compute some identifier of the input
		metadata := InstanceMetadata{
			Hash: CreateInstanceHash(opt),
		}
		metadataXML, err := XMLMarshall(metadata)
		if err != nil {
			return nil, err
		}
		// check for domain
		existingDomain, valid := isInstanceValid(opt)
		if valid {
			return existingDomain, nil
		}
		// cidata
		cidataIso, err := CreateCloudInit([]byte(opt.UserData), createMetaData(name))
		if err != nil {
			return nil, err
		}
		// delete a previous domain
		log.Println("Deleting domain ...")
		err = deleteDomain(name)
		if err != nil {
			return nil, err
		}
		// make sure to upload the image
		log.Println("Uploading boot disk ...")
		bootVolume, err := uploadBootDisk(opt.StoragePool, path.Base(opt.ImageURL), opt.ImageURL)
		if err != nil {
			return nil, err
		}
		// make sure to clone the image
		log.Println("Cloning boot disk ...")
		clonedBootVolume, err := cloneBootDisk(opt.StoragePool, bootVolume, bootName)
		if err != nil {
			return nil, err
		}
		// make sure to upload cidata
		log.Println("Uploading cidata disk ...")
		cidataVolume, err := uploadCloudInit(opt.StoragePool, cidataName, cidataIso)
		if err != nil {
			return nil, err
		}
		// reserve space for the logs
		log.Println("Initializing console logging ...")
		logVolume, err := createLoggingVolume(opt.StoragePool, logName)
		if err != nil {
			return nil, err
		}
		// construct the libvirt XML
		bootXML, err := createBootDisk(clonedBootVolume.Key)
		if err != nil {
			return nil, err
		}
		cidataXML, err := createCloudInit(cidataVolume.Key)
		if err != nil {
			return nil, err
		}
		domainXML, err := createDefaultDomainDef(client)
		if err != nil {
			return nil, err
		}
		// update some fields
		domainXML.Name = name
		domainXML.Metadata.XML = metadataXML
		domainXML.Devices.Disks = append(domainXML.Devices.Disks, *bootXML, *cidataXML) // order of disks is important
		// add data disks
		for idx, dataDisk := range opt.DataDisks {
			diskXML, err := createDataDiskXML(dataDisk.StoragePool, dataDisk.Name, idx)
			if err != nil {
				return nil, err
			}
			domainXML.Devices.Disks = append(domainXML.Devices.Disks, *diskXML)
		}
		// write console log to a file
		domainXML.Devices.Consoles[0].Log = &libvirtxml.DomainChardevLog{
			File:   logVolume.Key,
			Append: "off",
		}
		// add networks
		if len(opt.Networks) > 0 {
			networks, err := createNetworksXML(opt.Networks)
			if err != nil {
				return nil, err
			}
			domainXML.Devices.Interfaces = networks
		}
		// start the domain
		return startDomain(domainXML)
	}
}

// DeleteInstanceSync (synchronously) deletes an instance
func DeleteInstanceSync(client *LivirtClient) func(storagePool, name string) error {

	conn := client.LibVirt
	deleteDomain := DeleteDomainByName(client)
	delDisk := deleteStorageVol(conn)

	// delete the disks, but failure will only be logged
	delDisks := func(storagePool, name string) {
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			log.Printf("Unable to locate storage pool [%s], cause: [%v]", storagePool, err)
			return
		}
		// print some status
		log.Printf("Deleting disks attached to domain [%s] ...", name)
		// check the names
		volumes := []string{
			GetCIDataVolumeName(name),
			GetBootVolumeName(name),
			GetLoggingVolumeName(name),
		}
		// delete the volumes
		for _, vol := range volumes {
			_, err = delDisk(pool, vol)
			if err != nil {
				log.Printf("Unable to delete disk [%s], cause: [%v]", vol, err)
			}
		}
	}

	return func(storagePool, name string) error {
		// delete the domain
		err := deleteDomain(name)
		// delete the disks
		delDisks(storagePool, name)
		// done
		return err
	}
}
