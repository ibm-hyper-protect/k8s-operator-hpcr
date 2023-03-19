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

package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/datadisk"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/onprem"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/vpc"
)

// CreateServer creates the server that implements the actual controller
func CreateServer(version, compileTime string) func(port int) error {
	r := gin.Default()
	// register the VPC routes
	r.GET("/vpc/ping", vpc.CreatePingRoute(version, compileTime))
	r.POST("/vpc/sync", vpc.CreateControllerSyncRoute())
	r.POST("/vpc/finalize", vpc.CreateControllerFinalizeRoute())
	r.POST("/vpc/customize", vpc.CreateControllerCustomizeRoute())
	// register the onprem routes
	r.GET("/onprem/ping", onprem.CreatePingRoute(version, compileTime))
	r.POST("/onprem/sync", onprem.CreateControllerSyncRoute())
	r.POST("/onprem/finalize", onprem.CreateControllerFinalizeRoute())
	r.POST("/onprem/customize", onprem.CreateControllerCustomizeRoute())
	// register the data disk routes
	r.GET("/datadisk/ping", datadisk.CreatePingRoute(version, compileTime))
	r.POST("/datadisk/sync", datadisk.CreateControllerSyncRoute())
	r.POST("/datadisk/finalize", datadisk.CreateControllerFinalizeRoute())
	r.POST("/datadisk/customize", datadisk.CreateControllerCustomizeRoute())

	return func(port int) error {
		return r.Run(fmt.Sprintf(":%d", port))
	}
}
