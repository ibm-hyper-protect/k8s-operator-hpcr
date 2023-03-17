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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainXML(t *testing.T) {
	config, err := defaultSSHConfig("../.env")
	if err != nil {
		t.SkipNow()
	}

	client, err := CreateLivirtClient(config)
	require.NoError(t, err)

	uploader := UploadCloudInit(client)
	removeCloudInit := RemoveCloudInit(client)

	def, err := createDefaultDomainDef(client)
	require.NoError(t, err)

	userDataContent := []byte("userdata1")
	metaDataContent := []byte("metadata2")

	isoData, err := CreateCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)

	vol, err := uploader("libvirt", "TestDomainXML.iso", isoData)
	require.NoError(t, err)

	defer removeCloudInit(vol.Key)

	cloudInitDisk, err := CreateCloudInitDisk(client)(vol.Key)
	require.NoError(t, err)

	bootDisk, err := CreateBootDiskXML(client)("/var/lib/libvirt/hpcr.qcow2")
	require.NoError(t, err)

	def.Devices.Disks = append(def.Devices.Disks, *cloudInitDisk, *bootDisk)

	data, err := XMLMarshall(def)
	require.NoError(t, err)

	fmt.Println(data)
}
