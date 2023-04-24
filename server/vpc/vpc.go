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

package vpc

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/gin-gonic/gin"
	E "github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/vpc"
)

type RuntimeConfig struct {
	Authenticator core.Authenticator
	Service       *vpcv1.VpcV1
	Options       *InstanceOptions
	Env           E.Environment
}

func CreatePingRoute(version, compileTime string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": version,
			"compile": compileTime,
		})
	}
}

func createRuntimeConfig(req map[string]any) (*RuntimeConfig, error) {
	env := common.EnvFromConfigMapsOrSecrets(req)

	cfg, err := common.Transcode[*InstanceConfigResource](req)
	if err != nil {
		log.Printf("Unable to convert input to InstanceConfigResource, cause: [%v]", err)
		return nil, err
	}

	auth, err := vpc.CreateAuthenticatorFromEnv(env)
	if err != nil {
		log.Printf("Unable to create authenticator, cause: [%v]", err)
		return nil, err
	}

	searchSvc, err := vpc.CreateGlobalSearchServiceFromEnv(auth, env)
	if err != nil {
		log.Printf("Unable to create global search service, cause: [%v]", err)
		return nil, err
	}

	subnetID, err := getSubnetID(cfg, env)
	if err != nil {
		log.Printf("Unable to find subnet, cause: [%v]", err)
		return nil, err
	}

	region, err := vpc.FindRegionFromSubnet(searchSvc)(subnetID)
	if err != nil {
		log.Printf("Unable to find region, cause: [%v]", err)
		return nil, err
	}

	vpcSvc, err := vpc.CreateVpcServiceFromEnvAndRegion(auth, region, env)
	if err != nil {
		log.Printf("Unable to create VPC service, cause: [%v]", err)
		return nil, err
	}

	opt, err := InstanceOptionsFromConfigMap(vpcSvc, cfg, env)
	if err != nil {
		log.Printf("Unable to create options, cause: [%v]", err)
		return nil, err
	}

	return &RuntimeConfig{
		Authenticator: auth,
		Service:       vpcSvc,
		Options:       opt,
		Env:           env,
	}, nil
}

func syncVPC(req map[string]any) common.Action {

	cfg, err := createRuntimeConfig(req)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	taggingSvc, err := vpc.CreateTaggingServiceFromEnv(cfg.Authenticator, cfg.Env)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	return CreateSyncAction(cfg.Service, taggingSvc, cfg.Options)
}

func finalizeVPC(req map[string]any) common.Action {

	cfg, err := createRuntimeConfig(req)
	if err != nil {
		return common.CreateErrorAction(err)
	}

	return CreateFinalizeAction(cfg.Service, cfg.Options)
}

func CreateControllerSyncRoute() gin.HandlerFunc {

	return func(c *gin.Context) {
		log.Printf("synchronizing cloud VSI ...")
		jsonData, err := io.ReadAll(c.Request.Body)
		if err != nil {
			// print stome log
			log.Printf("Error accessing the request body, cause: [%v]", err)
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		var req map[string]any
		err = json.Unmarshal(jsonData, &req)
		if err != nil {
			// print stome log
			log.Printf("Error during unmarshaling, cause: [%v]", err)
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// constuct the action
		action := syncVPC(req)
		// execute and handle
		state, err := action()
		if err != nil {
			// print some log
			log.Printf("Error executing the sync, cause: [%v]", err)
			// Handle error
			c.JSON(http.StatusBadRequest, common.ResourceStatusToResponse(state))
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

func CreateControllerFinalizeRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("finalizing ...")

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
		// constuct the action
		action := finalizeVPC(req)
		// execute and handle
		state, err := action()
		if err != nil {
			// Handle error
			c.JSON(http.StatusOK, common.ResourceStatusToResponse(state))
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
		c.JSON(http.StatusOK, resp)
	}
}

func CreateControllerCustomizeRoute() gin.HandlerFunc {
	return func(c *gin.Context) {

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
		cfg, err := common.Transcode[*InstanceConfigResource](req)
		if err != nil {
			// Handle error
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		// print namespace
		log.Printf("Getting related resources for [%s] in namespace [%s] ...", cfg.Parent.Name, cfg.Parent.Namespace)

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
			log.Printf("customize response for for [%s] in namespace [%s]: [%s]", cfg.Parent.Name, cfg.Parent.Namespace, string(data))
		}

		// done
		c.JSON(http.StatusOK, resp)
	}

}
