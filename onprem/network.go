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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	A "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/array"
	"libvirt.org/go/libvirtxml"
)

const (
	// default name for the network to attach to
	DefaultNetwork = "default"
)

var (
	// full identifier of the disk ref config entry
	KeyNetworkRefConfig = fmt.Sprintf("%s.%s", KindNetworkRef, APIVersion)
)

// IsNetworkDhcpLeaseForHostname checks if this lease is for the hostname
func IsNetworkDhcpLeaseForHostname(hostname string) func(libvirt.NetworkDhcpLease) bool {
	return func(lease libvirt.NetworkDhcpLease) bool {
		for _, name := range lease.Hostname {
			if name == hostname {
				return true
			}
		}
		return false
	}
}

// GetDCHPLeases returns the DCHP leases for a given network
func GetDCHPLeases(client *LivirtClient) func(networkName string) ([]libvirt.NetworkDhcpLease, error) {

	conn := client.LibVirt

	return func(networkName string) ([]libvirt.NetworkDhcpLease, error) {

		network, err := conn.NetworkLookupByName(networkName)
		if err != nil {
			log.Printf("Unable to lookup the network [%s], cause: [%v]", networkName, err)
			return nil, err
		}

		leases, ret, err := conn.NetworkGetDhcpLeases(network, nil, math.MaxInt32, 0)
		if err != nil {
			log.Printf("Unable to get leases for the network [%s], ret: [%d], cause: [%v]", networkName, ret, err)
			return nil, err
		}

		return leases, err
	}
}

func parseNetworkXML(s string) (*libvirtxml.Network, error) {
	var networkDef libvirtxml.Network
	err := xml.Unmarshal([]byte(s), &networkDef)
	if err != nil {
		return nil, err
	}
	return &networkDef, nil
}

func getNetworkXMLDesc(conn *libvirt.Libvirt) func(net *libvirt.Network) (*libvirtxml.Network, error) {
	return func(net *libvirt.Network) (*libvirtxml.Network, error) {
		// try to get more info
		xmlDef, err := conn.NetworkGetXMLDesc(*net, 0)
		if err != nil {
			return nil, err
		}
		// parse
		return parseNetworkXML(xmlDef)
	}
}

// GetNetworkRef tries to return a network ref
func GetNetworkRef(client *LivirtClient) func(opt *NetworkRefOptions) (*libvirtxml.Network, error) {
	// connection
	conn := client.LibVirt
	networkXMLDesc := getNetworkXMLDesc(conn)

	return func(opt *NetworkRefOptions) (*libvirtxml.Network, error) {
		// check for the network
		net, err := conn.NetworkLookupByName(opt.Name)
		if err != nil {
			log.Printf("Unable to lookup network [%s], cause: [%v]", opt.Name, err)
			return nil, err
		}
		// get some metadata
		netXML, err := networkXMLDesc(&net)
		if err != nil {
			log.Printf("Unable to get information for network [%s], cause: [%v]", opt.Name, err)
			return nil, err
		}
		// nothing to do
		return netXML, nil
	}
}

// NetworkRefsFromRelated decodes the set of configured networks from the related data structure
func NetworkRefsFromRelated(data map[string]any) ([]*NetworkRefCustomResource, error) {
	var result []*NetworkRefCustomResource
	if related, ok := data["related"].(map[string]any); ok {
		// all config maps
		if networkRefs, ok := related[KeyNetworkRefConfig].(map[string]any); ok {
			// decode each network ref
			for _, networkRef := range networkRefs {
				// transcode to the expected format
				netRef, err := common.Transcode[*NetworkRefCustomResource](networkRef)
				if err != nil {
					return nil, err
				}
				// validate the status of the network ref
				if common.Status(netRef.Status.Status) == common.Ready {
					result = append(result, netRef)
				} else {
					// print the invalid network config
					res, err := json.Marshal(netRef)
					if err == nil {
						log.Printf("Network reference not ready is [%s]", string(res))
					}
					// disk is not in a valid status
					log.Printf("Network Reference [%s] is not in ready state, ignoring, cause: [%s]", netRef.Name, netRef.Status.Description)
				}
			}
		}
	}
	// ok
	return result, nil
}

func networkRefCustomResourceToNetworks(res *NetworkRefCustomResource) string {
	return res.Spec.NetworkName
}

// NetworkRefCustomResourceToNetworks converts from an array of NetworkRefCustomResource to an array of attached disks
var NetworkRefCustomResourceToNetworks = A.Map(networkRefCustomResourceToNetworks)

func createNetworkXML(networkName string) libvirtxml.DomainInterface {
	log.Printf("Defining domain interface on network [%s]", networkName)
	return libvirtxml.DomainInterface{
		Model: &libvirtxml.DomainInterfaceModel{
			Type: "virtio",
		},
		Source: &libvirtxml.DomainInterfaceSource{
			Network: &libvirtxml.DomainInterfaceSourceNetwork{
				Network: networkName,
			},
		},
		Driver: &libvirtxml.DomainInterfaceDriver{
			IOMMU: "on",
		},
	}
}

// CreateNetworksXML creates the XML for the networks
func CreateNetworksXML() func(networkNames []string) ([]libvirtxml.DomainInterface, error) {

	mapNames := A.Map(createNetworkXML)

	return func(networkNames []string) ([]libvirtxml.DomainInterface, error) {
		return mapNames(networkNames), nil
	}
}
