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
	"testing"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/server/common"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/vpc"
	"github.com/stretchr/testify/require"
)

func TestAction(t *testing.T) {
	env, err := envFromDotEnv()
	if err != nil {
		t.Skipf("No .env file")
	}
	auth, err := vpc.CreateAuthenticatorFromEnv(env)
	require.NoError(t, err)

	vpcSvc, err := vpc.CreateVpcServiceFromEnv(auth, env)
	require.NoError(t, err)

	taggingSvc, err := vpc.CreateTaggingServiceFromEnv(auth, env)
	require.NoError(t, err)

	// load some data
	data, err := loadMap("../../samples/create_resource_full.json")
	require.NoError(t, err)

	cfg, err := common.Transcode[*InstanceConfigResource](data)
	require.NoError(t, err)

	opt, err := InstanceOptionsFromConfigMap(vpcSvc, cfg, env)
	require.NoError(t, err)

	// execute and get status
	status, err := CreateSyncAction(vpcSvc, taggingSvc, opt)
	require.NoError(t, err)

	fmt.Println(status)
}
