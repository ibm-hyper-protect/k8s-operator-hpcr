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
	"strings"

	"github.com/digitalocean/go-libvirt"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	A "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/array"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	"libvirt.org/go/libvirtxml"
)

var (
	emptyIPAddresses = A.Empty[string]()
)

func getIPAddress(lease libvirt.NetworkDhcpLease) string {
	return lease.Ipaddr
}

func createInstanceRunningAction(client *onprem.LivirtClient, inst *libvirtxml.Domain, opt *onprem.InstanceOptions) (*common.ResourceStatus, error) {

	getLoggingVolume := onprem.GetLoggingVolume(client)
	getLeases := onprem.GetDCHPLeases(client)

	// getIPAddresses determines the IP Addresses for the instance by checking for a all leases
	// for the configured network and then filtering down the list to the hostname
	getIPAddresses := func() []string {
		networks := onprem.GetNetworks(opt)
		var leases []libvirt.NetworkDhcpLease
		for _, network := range networks {
			lses, err := getLeases(network)
			if err != nil {
				log.Printf("Unable to get the leases for network [%s], cause: [%v]", network, err)
				return emptyIPAddresses
			}
			// append all
			leases = append(leases, lses...)
		}
		// filter down
		return F.Pipe2(
			leases,
			A.Filter(onprem.IsNetworkDhcpLeaseForHostname(opt.Name)),
			A.Map(getIPAddress),
		)
	}

	// fetch the logs
	log.Printf("Domain [%s] is running, fetching logs ...", opt.Name)
	// try to get the content of the logging volume
	logName := onprem.GetLoggingVolumeName(opt.Name)
	data, err := getLoggingVolume(opt.StoragePool, logName)
	if err != nil {
		// log this
		log.Printf("Unable to get the logging volumd [%s] from pool [%s], cause: [%v]", logName, opt.StoragePool, err)
		// returns some error status
		return &common.ResourceStatus{
			Status:      common.Waiting,
			Description: err.Error(),
			Error:       err,
		}, err
	}
	// marshal the instance
	instStrg, err := onprem.XMLMarshall(inst)
	// parse log into lines
	lines := F.Pipe1(
		strings.Split(data, "\n"),
		A.Map(strings.TrimSpace),
	)
	// partition the lines
	success, failure := onprem.PartitionLogs(lines)
	if onprem.VSIFailedToStart(failure) {
		// print some error details
		logs := strings.Join(failure, "\n")
		log.Printf("Domain [%s] failed to start, errors: [%s]", opt.Name, logs)
		// assemble some metadata
		metadata := C.RawMap{
			"logs": logs,
		}
		if err == nil {
			metadata["domainXML"] = instStrg
		}
		// VSI is ready but in an error state. It won't start at the next attempt
		return common.CreateAction(&common.ResourceStatus{
			Status:      common.Ready,
			Description: logs,
			Error:       nil,
			Metadata:    metadata,
		})
	}
	// check if we are still booting
	if onprem.VSIStartedSuccessfully(success) {
		// some logs
		logs := strings.Join(lines, "\n")
		// assemble some metadata
		metadata := C.RawMap{
			"logs":        logs,
			"ipaddresses": getIPAddresses(),
		}
		if err == nil {
			metadata["domainXML"] = instStrg
		}
		// juhuuu
		return common.CreateAction(&common.ResourceStatus{
			Status:      common.Ready,
			Description: logs,
			Error:       nil,
			Metadata:    metadata,
		})
	}
	// log this
	desc := strings.Join(lines, "\n")
	log.Printf("Domain [%s] is still booting, logs: [%s]", opt.Name, desc)
	// we need to wait
	return common.CreateAction(&common.ResourceStatus{
		Status:      common.Waiting,
		Description: desc,
		Error:       nil,
	})
}

// CreateSyncAction synchronizes the state of the resource and determines what to do next
func CreateSyncAction(client *onprem.LivirtClient, opt *onprem.InstanceOptions) (*common.ResourceStatus, error) {
	// checks for the validity of the instance
	isInstanceValid := onprem.IsInstanceValid(client)
	inst, ok := isInstanceValid(opt)
	if ok {
		// validate the instance
		return createInstanceRunningAction(client, inst, opt)
	}
	// start the instance
	instSync := onprem.CreateInstanceSync(client)
	result, err := instSync(opt)
	if err != nil {
		log.Printf("Unable to create the VSI [%s], cause: [%v]", opt.Name, err)
		return common.CreateErrorAction(err)
	}
	// log the result
	resultStrg, err := onprem.XMLMarshall(result)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	log.Printf("Instance: %s", resultStrg)
	// we need an additional sync to tell if the instance is ready
	return common.CreateWaitingAction()
}

func CreateFinalizeAction(client *onprem.LivirtClient, opt *onprem.InstanceOptions) (*common.ResourceStatus, error) {
	// TODO proper check for existence comes here
	// ...
	// destroy the instance
	deleteSync := onprem.DeleteInstanceSync(client)
	err := deleteSync(opt.StoragePool, opt.Name)
	if err != nil {
		log.Printf("Unable to delete the VSI [%s], cause: [%v]", opt.Name, err)
		return common.CreateErrorAction(err)
	}
	// done
	return common.CreateReadyAction()
}
