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
	"io"
	"log"
	"net/http"

	A "github.com/IBM/fp-go/array"
	"github.com/gin-gonic/gin"
	CM "github.com/ibm-hyper-protect/k8s-operator-hpcr/common"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/datadisk"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/lock"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/networkref"
)

func CreatePingRoute(version, compileTime string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": version,
			"compile": compileTime,
		})
	}
}

// syncOnPrem is invoked to synchronize the state of our resource
func syncOnPrem(req map[string]any) (*common.ResourceStatus, error) {
	// just a poor man's solution for now
	if !lock.Lock.TryLock() {
		log.Println("Sync: waiting for lock ...")
		return common.CreateStatusAction(common.Waiting)
	}
	defer lock.Lock.Unlock()
	// assemble all information about the environment by merging the config maps
	env := common.EnvFromConfigMapsOrSecrets(req)

	attachedDataDisks, err := onprem.AttachedDataDisksFromRelated(req)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	// assemble information about the attached networkRefs
	networkRefs, err := onprem.NetworkRefsFromRelated(req)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	client, err := onprem.CreateLivirtClientFromEnvMap(env)
	if err != nil {
		log.Printf("Unable to create libvirt client, cause: [%v]", err)
		return common.CreateErrorAction(err)
	}
	defer client.Close()

	cfg, err := common.Transcode[*OnPremConfigResource](req)
	if err != nil {
		log.Printf("Unable to decode request, cause: [%v]", err)
		return common.CreateErrorAction(err)
	}

	opt, err := onpremInstanceOptionsFromConfigMap(cfg, env)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	// dump the attached network references
	if A.IsNonEmpty(networkRefs) {
		// extract names
		networkRefNames := A.MonadMap(networkRefs, func(disk *onprem.NetworkRefCustomResource) string {
			return disk.Name
		})
		// log the disks
		log.Printf("NetworkRefs: %v", networkRefNames)
	}

	// attach data disks
	opt.DataDisks = attachedDataDisks

	// attach networks
	opt.Networks = onprem.NetworkRefCustomResourceToNetworks(networkRefs)

	// make sure to construct the VSI
	return CreateSyncAction(client, opt)
}

// finalizeOnPrem deletes a VSI
func finalizeOnPrem(req map[string]any) (*common.ResourceStatus, error) {

	if !lock.Lock.TryLock() {
		log.Println("Finalize: waiting for lock ...")
		return common.CreateStatusAction(common.Waiting)
	}
	defer lock.Lock.Unlock()

	env := common.EnvFromConfigMapsOrSecrets(req)

	client, err := onprem.CreateLivirtClientFromEnvMap(env)
	if err != nil {
		log.Printf("Unable to create libvirt client, cause: [%v]", err)
		return common.CreateErrorAction(err)
	}
	defer client.Close()

	cfg, err := common.Transcode[*OnPremConfigResource](req)
	if err != nil {
		log.Printf("Unable to decode request, cause: [%v]", err)
		return common.CreateErrorAction(err)
	}

	opt, err := onpremInstanceOptionsFromConfigMap(cfg, env)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	return CreateFinalizeAction(client, opt)
}

func CreateControllerSyncRoute() gin.HandlerFunc {

	return func(c *gin.Context) {
		// log this config
		defer CM.EntryExit("OnPremCreateControllerSyncRoute")()

		jsonData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// decode the input
		var req map[string]any
		err = json.Unmarshal(jsonData, &req)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// log the request
		// log.Printf("JSON Input [%s]", string(jsonData))
		// execute and handle
		state, err := syncOnPrem(req)
		if err != nil {
			log.Printf("Error [%v]", err)
			// switch into error mode
			c.JSON(http.StatusOK, common.ResourceStatusToResponse(state))
			// bail out
			return
		}
		// done
		resp := common.ResourceStatusToResponse(state)
		// set a retry if we are not ready, yet
		if state.Status != common.Ready {
			resp["resyncAfterSeconds"] = 10
		}

		// data, err := json.Marshal(resp)
		// if err == nil {
		// 	log.Printf("Sync Response: [%s]", string(data))
		// }

		// done
		c.JSON(http.StatusOK, resp)
	}
}

func CreateControllerFinalizeRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		// log this config
		defer CM.EntryExit("OnPremCreateControllerFinalizeRoute")()

		jsonData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		var req map[string]any
		err = json.Unmarshal(jsonData, &req)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// execute and handle
		state, err := finalizeOnPrem(req)
		if err != nil {
			log.Printf("Error [%v]", err)
			// Handle error TODO really handle error
			c.JSON(http.StatusOK, gin.H{
				"finalized": true,
			})
			// bail out
			return
		}
		// done finalizing
		finalized := state.Status == common.Ready
		resp := gin.H{
			"finalized": finalized,
		}
		if !finalized {
			resp["resyncAfterSeconds"] = 10
		}
		// final response
		c.JSON(http.StatusOK, resp)
		log.Printf("Finalized: [%t]", finalized)
	}
}

// CreateControllerCustomizeRoute is invoked to
func CreateControllerCustomizeRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		// log this config
		defer CM.EntryExit("OnPremCreateControllerCustomizeRoute")()
		// parse body
		jsonData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// decode the input
		var req map[string]any
		err = json.Unmarshal(jsonData, &req)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// transcode to the expected format
		cfg, err := common.Transcode[*OnPremConfigResource](req)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// print namespace
		log.Printf("Getting related resources for [%s] in namespace [%s] ...", cfg.Parent.Name, cfg.Parent.Namespace)
		// produce a response
		resp := common.CustomizeHookResponse{
			RelatedResourceRules: common.CreateRelatedResourceRules([]common.RelatedResource{
				// config
				common.RefConfigMaps(cfg.Parent.Spec.TargetSelector),
				common.RefSecrets(cfg.Parent.Spec.TargetSelector),
				// disk
				datadisk.RefDataDisks(cfg.Parent.Spec.DiskSelector),
				datadisk.RefDataDiskRefs(cfg.Parent.Spec.DiskSelector),
				// networks
				networkref.RefNetworkRefs(cfg.Parent.Spec.NetworkSelector),
			}),
		}
		// dump it
		data, err := json.Marshal(resp)
		if err == nil {
			log.Printf("customize response for for [%s] in namespace [%s]: [%s]", cfg.Parent.Name, cfg.Parent.Namespace, string(data))
		}

		// done
		c.JSON(http.StatusOK, resp)
	}
}
