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
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeNoAttachedDataDisk(t *testing.T) {
	// read sample
	relJson, err := os.ReadFile("../samples/create_resource_full.json")
	require.NoError(t, err)

	var rel map[string]any
	err = json.Unmarshal(relJson, &rel)
	require.NoError(t, err)

	disks, err := AttachedDataDisksFromRelated(rel)
	require.NoError(t, err)

	assert.Empty(t, disks)
}

func TestDecodeOneAttachedDataDisk(t *testing.T) {
	// read sample
	relJson, err := os.ReadFile("./samples/create.req.json")
	require.NoError(t, err)

	var rel map[string]any
	err = json.Unmarshal(relJson, &rel)
	require.NoError(t, err)

	disks, err := AttachedDataDisksFromRelated(rel)
	require.NoError(t, err)

	assert.Len(t, disks, 1)
}

func TestCreateDataDisk(t *testing.T) {
	env, err := godotenv.Read("../.env")
	if err != nil {
		t.SkipNow()
	}

	config, err := getSSHConfigFromEnv(env)
	require.NoError(t, err)

	// get some config
	storagePool, ok := env[KeyStoragePool]
	require.True(t, ok)

	client, err := CreateLivirtClient(config)
	require.NoError(t, err)

	// expected size
	expSize := uint64(100 * 1024 * 1024 * 1024)

	// create the data disk
	dataDisk, err := CreateDataDisk(client)(storagePool, "TestCreateDataDisk", expSize)
	require.NoError(t, err)

	defer func() {
		err := RemoveDataDisk(client)(dataDisk.Key)
		if err != nil {
			log.Printf("Error removing the data disk, cause [%v]", err)
		}
	}()

	// get some metadata
	data, err := getStorageVolXMLDesc(client.LibVirt)(dataDisk)
	require.NoError(t, err)

	assert.Equal(t, data.Capacity.Value, expSize)
	assert.Equal(t, data.Name, dataDisk.Name)
}
