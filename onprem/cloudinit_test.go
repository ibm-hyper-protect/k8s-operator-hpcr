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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdomanski/iso9660"
)

func TestCloudInit(t *testing.T) {
	path := "../build/test/TestCloudInit.iso"

	userDataContent := []byte("userdata")
	metaDataContent := []byte("metadata")

	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	require.NoError(t, err)

	os.Remove(path)

	isoData, err := CreateCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)

	err = os.WriteFile(path, isoData, os.ModePerm)
	require.NoError(t, err)

	isoFile, err := os.Open(path)
	require.NoError(t, err)

	isoImg, err := iso9660.OpenImage(isoFile)
	require.NoError(t, err)

	rootFile, err := isoImg.RootDir()
	require.NoError(t, err)

	children, err := rootFile.GetChildren()
	require.NoError(t, err)

	files := make(map[string][]byte)
	for _, child := range children {
		key := child.Name()
		data, err := io.ReadAll(child.Reader())
		require.NoError(t, err)

		files[key] = data
	}

	assert.Equal(t, userDataContent, files[userDataFilename])
	assert.Equal(t, metaDataContent, files[metaDataFilename])

	err = isoFile.Close()
	require.NoError(t, err)
}

func TestCloudInitUpload(t *testing.T) {
	config, err := defaultSSHConfig("../.env")
	if err != nil {
		t.SkipNow()
	}

	userDataContent := []byte("userdata1")
	metaDataContent := []byte("metadata2")

	isoData, err := CreateCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)

	client, err := CreateLivirtClient(config)
	require.NoError(t, err)

	uploader := UploadCloudInit(client)

	vol, err := uploader("libvirt", "TestCloudInitUpload.iso", isoData)
	require.NoError(t, err)

	// defer RemoveCloudInit(client)(vol.Key)

	assert.NotNil(t, vol)
}
