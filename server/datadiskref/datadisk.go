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

package datadiskref

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
)

func CreatePingRoute(version, compileTime string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": version,
			"compile": compileTime,
		})
	}
}

// syncDataDisk is invoked to synchronize the state of our resource
func syncDataDisk(req map[string]any) common.Action {
	// assemble all information about the environment by merging the config maps
	env := common.EnvFromConfigMapsOrSecrets(req)

	client, err := onprem.CreateLivirtClientFromEnvMap(env)
	if err != nil {
		return common.CreateErrorAction(err)
	}
	defer client.Close()

	cfg, err := common.Transcode[*DataDiskRefConfigResource](req)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	opt, err := dataDiskRefOptionsFromConfigMap(cfg, env)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	return CreateSyncAction(client, opt)
}

func CreateControllerSyncRoute() gin.HandlerFunc {

	return func(c *gin.Context) {
		log.Printf("synchronizing data disk ref...")
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
		// constuct the action
		action := syncDataDisk(req)
		// execute and handle
		state, err := action()
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
		// done
		c.JSON(http.StatusOK, resp)
	}
}

// CreateControllerCustomizeRoute is invoked to
func CreateControllerCustomizeRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		cfg, err := common.Transcode[*DataDiskRefConfigResource](req)
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
			}),
		}
		// dump it
		data, err := json.Marshal(resp)
		if err == nil {
			log.Printf("customize response [%s]", string(data))
		}

		// done
		c.JSON(http.StatusOK, resp)
	}
}
