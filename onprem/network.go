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
	"log"
	"math"

	libvirt "github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

const (
	// default name for the network to attach to
	DefaultNetwork = "default"
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
