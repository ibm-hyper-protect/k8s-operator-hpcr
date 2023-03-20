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

package common

import (
	"github.com/gin-gonic/gin"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
)

type Status int

const (
	Waiting Status = iota
	Ready
	Error
)

type ResourceStatus struct {
	Status      Status
	Description string
	Error       error
	Metadata    C.RawMap
}

type Action func() (*ResourceStatus, error)

func CreateReadyAction() Action {
	return CreateStatusAction(Ready)
}

func CreateWaitingAction() Action {
	return CreateStatusAction(Waiting)
}

func CreateStatusAction(status Status) Action {
	return func() (*ResourceStatus, error) {
		return &ResourceStatus{
			Status:      status,
			Description: "Ready",
			Error:       nil,
		}, nil
	}
}

func CreateErrorAction(err error) Action {
	return func() (*ResourceStatus, error) {
		return &ResourceStatus{
			Status:      Error,
			Description: err.Error(),
			Error:       err,
		}, err
	}
}

func ResourceStatusToResponse(state *ResourceStatus) gin.H {
	return gin.H{
		"status": gin.H{
			"status":      state.Status,
			"description": state.Description,
			"metadata":    state.Metadata,
		},
	}
}
