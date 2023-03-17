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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/stretchr/testify/require"

	E "github.com/ibm-hyper-protect/hpcr-controller/env"
)

func envFromDotEnv() (E.Environment, error) {
	// read the file
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return EnvFromDotEnv(filepath.Join(dir, ".."))
}

func TestCreateService(t *testing.T) {
	env, err := envFromDotEnv()
	if err != nil {
		t.Skipf("No .env file")
	}

	service, err := CreateVpcServiceFromEnv(env)
	require.NoError(t, err)

	vpcsPager, err := service.NewVpcsPager(&vpcv1.ListVpcsOptions{})
	require.NoError(t, err)

	vpcs, err := vpcsPager.GetAll()
	require.NoError(t, err)

	for _, item := range vpcs {
		fmt.Println(*item.Name)
	}

	fmt.Println(service)
}

func TestSubnet(t *testing.T) {
	subnetid := "0726-b3c4aa3a-928a-4c8f-97b7-02ad4723c4e4"
	env, err := envFromDotEnv()
	if err != nil {
		t.Skipf("No .env file")
	}

	service, err := CreateVpcServiceFromEnv(env)
	require.NoError(t, err)

	subnet, _, err := service.GetSubnet(&vpcv1.GetSubnetOptions{ID: &subnetid})
	require.NoError(t, err)

	vpcid := subnet.VPC.ID
	zone := subnet.Zone.Name

	fmt.Printf("vpc: %s, zone: %s\n", *vpcid, *zone)
}
