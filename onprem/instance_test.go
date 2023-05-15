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
	"testing"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/common"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstance(t *testing.T) {
	// load env
	env, err := godotenv.Read("../.env")
	if err != nil {
		t.SkipNow()
	}

	// build the contract
	busybox, err := common.FromEither(getEncryptedBusyboxContract(env))
	require.NoError(t, err)
	// log the contract for fun
	log.Printf("Contract:\n%s", busybox)
	// some options
	instOpt := &InstanceOptions{
		Name:        "TestCreateInstance",
		UserData:    busybox,
		ImageURL:    "http://localhost:8080/hpcr.qcow2",
		StoragePool: "images",
	}
	// ssh client
	config, err := getSSHConfigFromEnv(env)
	require.NoError(t, err)
	client, err := CreateLivirtClient(config)
	require.NoError(t, err)
	// creator
	instSync := CreateInstanceSync(client)

	result, err := instSync(instOpt)
	require.NoError(t, err)

	// print the result
	log.Printf("UUID: [%s]", result.UUID)
}

func TestCreateHash(t *testing.T) {
	// disk attachment
	var firstDisk = AttachedDataDisk{
		Name:        "first",
		StoragePool: "defaultPool",
	}
	var secondDisk = AttachedDataDisk{
		Name:        "second",
		StoragePool: "defaultPool",
	}
	// check if the hash creation is stable
	var opt1 InstanceOptions = InstanceOptions{
		Name:        "Carsten",
		UserData:    "user_data",
		ImageURL:    "http://example.com",
		StoragePool: "defaultPool",
		DataDisks: []*AttachedDataDisk{
			&firstDisk,
			&secondDisk,
		},
		Networks: []string{
			"second",
			"first",
		},
	}

	var opt2 InstanceOptions = InstanceOptions{
		Name:        "Carsten",
		UserData:    "user_data",
		ImageURL:    "http://example.com",
		StoragePool: "defaultPool",
		DataDisks: []*AttachedDataDisk{
			&secondDisk,
			&firstDisk,
		},
		Networks: []string{
			"first",
			"second",
		},
	}

	var hash1 = CreateInstanceHash(&opt1)
	var hash2 = CreateInstanceHash(&opt2)

	assert.Equal(t, hash1, hash2)
}
