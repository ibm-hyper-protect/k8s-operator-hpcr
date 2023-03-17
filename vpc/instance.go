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
	"errors"
	"fmt"

	"github.com/IBM/vpc-go-sdk/vpcv1"
)

var InstanceNotFound = errors.New("instance was not found")

func FindInstance(service *vpcv1.VpcV1, name string) (*vpcv1.Instance, error) {
	pager, err := service.NewInstancesPager(&vpcv1.ListInstancesOptions{Name: &name})
	if err != nil {
		return nil, err
	}
	all, err := pager.GetAll()
	if err != nil {
		return nil, err
	}
	count := len(all)
	if count > 1 {
		return nil, fmt.Errorf("instance is not unique, total number is [%d]", count)
	}
	if count == 0 {
		return nil, InstanceNotFound
	}
	return &all[0], nil
}
