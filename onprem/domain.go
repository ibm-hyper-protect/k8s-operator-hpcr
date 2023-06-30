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
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"time"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/google/uuid"
	CM "github.com/ibm-hyper-protect/k8s-operator-hpcr/common"
	"libvirt.org/go/libvirtxml"
)

type Domain struct {
	Name     string `json:"name" yaml:"name"`
	Pool     string `json:"pool" yaml:"pool"`
	UserData string `json:"user_data" yaml:"user_data"`
}

func uuidToString(id libvirt.UUID) string {
	return uuid.UUID(id).String()
}

func getGuestForArchType(caps *libvirtxml.Caps, arch string, virttype string) (*libvirtxml.CapsGuest, error) {
	for _, guest := range caps.Guests {
		if guest.Arch.Name == arch && guest.OSType == virttype {
			return &guest, nil
		}
	}
	return nil, fmt.Errorf("could not find any guests for architecture type %s/%s", virttype, arch)
}

func lookupMachine(machines []libvirtxml.CapsGuestMachine, targetmachine string) string {
	for _, machine := range machines {
		if machine.Name == targetmachine {
			if machine.Canonical != "" {
				return machine.Canonical
			}
			return machine.Name
		}
	}
	return ""
}

func getCanonicalMachineName(caps *libvirtxml.Caps, arch string, virttype string, targetmachine string) (string, error) {
	guest, err := getGuestForArchType(caps, arch, virttype)
	if err != nil {
		return "", err
	}

	name := lookupMachine(guest.Arch.Machines, targetmachine)
	if name != "" {
		return name, nil
	}

	for _, domain := range guest.Arch.Domains {
		name := lookupMachine(domain.Machines, targetmachine)
		if name != "" {
			return name, nil
		}
	}

	return "", fmt.Errorf("cannot find machine type %s for %s/%s in %v", targetmachine, virttype, arch, caps)
}

func getHostCapabilities(virConn *libvirt.Libvirt) (*libvirtxml.Caps, error) {
	// We should perhaps think of storing this on the connect object
	// on first call to avoid the back and forth
	capsXML, err := virConn.Capabilities()
	if err != nil {
		return nil, err
	}

	caps := &libvirtxml.Caps{}
	err = xml.Unmarshal([]byte(capsXML), caps)
	if err != nil {
		return nil, err
	}

	return caps, nil
}

func createDefaultDomainDef(client *LivirtClient) (*libvirtxml.Domain, error) {

	conn := client.LibVirt

	caps, err := getHostCapabilities(conn)
	if err != nil {
		return nil, err
	}

	guest, err := getGuestForArchType(caps, "s390x", "hvm")
	if err != nil {
		return nil, err
	}

	canonicalmachine, err := getCanonicalMachineName(caps, "s390x", "hvm", "s390-ccw-virtio")
	if err != nil {
		return nil, err
	}

	return &libvirtxml.Domain{
		Type: "kvm",
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Type:    "hvm",
				Arch:    "s390x",
				Machine: canonicalmachine,
			},
		},
		Metadata: &libvirtxml.DomainMetadata{},
		Memory: &libvirtxml.DomainMemory{
			Value: 4 * 1024 * 1024,
		},
		CurrentMemory: &libvirtxml.DomainCurrentMemory{
			Value: 4 * 1024 * 1024,
		},
		VCPU: &libvirtxml.DomainVCPU{
			Value: 2,
		},
		Clock: &libvirtxml.DomainClock{
			Offset: "utc",
		},
		Devices: &libvirtxml.DomainDeviceList{
			Disks:    []libvirtxml.DomainDisk{},
			Emulator: guest.Arch.Emulator,
			MemBalloon: &libvirtxml.DomainMemBalloon{
				Model: "none",
			},
			RNGs: []libvirtxml.DomainRNG{
				{
					Model: "virtio",
					Backend: &libvirtxml.DomainRNGBackend{
						Random: &libvirtxml.DomainRNGBackendRandom{Device: "/dev/urandom"},
					},
				},
			},
			Consoles: []libvirtxml.DomainConsole{
				{
					Source: &libvirtxml.DomainChardevSource{
						Pty: &libvirtxml.DomainChardevSourcePty{},
					},
					Target: &libvirtxml.DomainConsoleTarget{
						Type: "sclp",
					},
				},
			},
			Interfaces: []libvirtxml.DomainInterface{
				{
					Model: &libvirtxml.DomainInterfaceModel{
						Type: "virtio",
					},
					Source: &libvirtxml.DomainInterfaceSource{
						Network: &libvirtxml.DomainInterfaceSourceNetwork{
							Network: DefaultNetwork,
						},
					},
					Driver: &libvirtxml.DomainInterfaceDriver{
						IOMMU: "on",
					},
				},
			},
		},
	}, nil
}

func parseDomainXML(s string) (*libvirtxml.Domain, error) {
	var domainDef libvirtxml.Domain
	err := xml.Unmarshal([]byte(s), &domainDef)
	if err != nil {
		return nil, err
	}
	return &domainDef, nil
}

func shutDownDomain(client *LivirtClient) func(domain *libvirt.Domain) error {
	conn := client.LibVirt
	return func(domain *libvirt.Domain) error {
		// check if the domain is running
		state, _, err := conn.DomainGetState(*domain, 0)
		if err != nil {
			// if we cannot get the domain state, assume it's gone
			log.Printf("Unable to get the domain state for domain [%s], err: [%v]", domain.Name, err)
			return nil
		}
		if libvirt.DomainState(state) != libvirt.DomainRunning {
			return nil
		}
		// try to shutdown the domain
		log.Printf("Shutting down domain [%s] ...", domain.Name)
		err = conn.DomainShutdown(*domain)
		if err != nil {
			return err
		}
		// wait for the domain state
		for i := 0; i < 50; i++ {
			// get the domain state
			state, reason, err := conn.DomainGetState(*domain, 0)
			log.Printf("Domain State [%d], reason [%d]", state, reason)
			if err != nil {
				// if we cannot get the domain state, assume it's gone
				log.Printf("Unable to get the domain state for domain [%s], err: [%v]", domain.Name, err)
				return nil
			}
			// check for states that depict a shutdown system
			switch libvirt.DomainState(state) {
			case libvirt.DomainShutdown, libvirt.DomainShutoff, libvirt.DomainCrashed, libvirt.DomainNostate, libvirt.DomainPaused, libvirt.DomainPmsuspended:
				return nil
			case libvirt.DomainRunning:
				// keep trying
			case libvirt.DomainBlocked:
				log.Printf("Domain [%s] is blocked, not sure what to do ...", domain.Name)
			default:
				log.Printf("Domain [%s] is in unknown state [%d]", domain.Name, state)
			}
			// wait a bit
			time.Sleep(2 * time.Second)
		}
		return fmt.Errorf("timeout waiting for domain [%s] to complete", domain.Name)
	}
}

// deleteDomain tries to gracefully shutdown the domain
func deleteDomain(client *LivirtClient) func(domain *libvirt.Domain) error {
	conn := client.LibVirt
	shutdown := shutDownDomain(client)

	return func(domain *libvirt.Domain) error {
		// shutdown
		err := shutdown(domain)
		if err != nil {
			return err
		}
		// final cleanup
		log.Printf("Destroying domain [%s] ...", domain.Name)
		err = conn.DomainDestroy(*domain)

		if err != nil {
			var libvirtErr libvirt.Error
			if !errors.As(err, &libvirtErr) || libvirtErr.Code != 55 {
				return err
			}
		}

		log.Printf("Undefining domain [%s] ...", domain.Name)
		err = conn.DomainUndefine(*domain)

		return err
	}
}

func DeleteDomainByName(client *LivirtClient) func(name string) error {

	conn := client.LibVirt

	delDomain := deleteDomain(client)

	return func(name string) error {
		// log this config
		defer CM.EntryExit(fmt.Sprintf("DeleteDomainByName(%s)", name))()
		// log this
		log.Printf("Deleting domain by name [%s] ...", name)
		// locate the domain
		domain, err := conn.DomainLookupByName(name)
		// TODO check for domain does not exist
		if err != nil {
			// log this fact
			log.Printf("Domain [%s] cannot be located, assuming it's been deleted, cause: [%v]", name, err)
			return nil
		}
		// delete
		return delDomain(&domain)
	}

}

func GetDomains(client *LivirtClient) func() ([]libvirt.Domain, error) {
	conn := client.LibVirt

	return func() ([]libvirt.Domain, error) {
		res, _, err := conn.ConnectListAllDomains(1000, 0)
		return res, err
	}
}

func StartDomain(client *LivirtClient) func(*libvirtxml.Domain) (*libvirtxml.Domain, error) {

	conn := client.LibVirt

	return func(domainXML *libvirtxml.Domain) (*libvirtxml.Domain, error) {
		// marshal
		domainString, err := XMLMarshall(domainXML)
		if err != nil {
			return nil, err
		}
		// dump the input
		log.Printf("Domain definition of domain [%s] is [%s]", domainXML.Name, domainString)
		// define the domain
		log.Printf("Defining domain [%s] ...", domainXML.Name)
		domain, err := conn.DomainDefineXML(domainString)
		if err != nil {
			return nil, err
		}
		err = conn.DomainSetAutostart(domain, 1)
		if err != nil {
			return nil, err
		}
		// get some identifier
		domainId := uuidToString(domain.UUID)
		// create the beast
		log.Printf("Creating domain [%s] with ID [%s]...", domain.Name, domainId)
		err = conn.DomainCreate(domain)
		if err != nil {
			return nil, err
		}
		// read back the domain info
		log.Printf("Reading domain info for [%s] with ID [%s] ...", domain.Name, domainId)
		xmlDesc, err := conn.DomainGetXMLDesc(domain, 0)
		if err != nil {
			return nil, err
		}
		// parse the XML
		updatedDomainXML, err := parseDomainXML(xmlDesc)
		if err != nil {
			return nil, err
		}
		// ok
		return updatedDomainXML, err
	}
}
