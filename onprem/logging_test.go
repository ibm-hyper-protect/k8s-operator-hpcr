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
	_ "embed"
	"strings"
	"testing"

	A "github.com/IBM/fp-go/array"
	F "github.com/IBM/fp-go/function"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed samples/successful_log.txt
	successLog string
	//go:embed samples/failure_log.txt
	failureLog string
)

func TestFailedStartup(t *testing.T) {
	lines := F.Pipe1(
		strings.Split(failureLog, "\n"),
		A.Map(strings.TrimSpace),
	)

	success, failure := PartitionLogs(lines)

	assert.False(t, VSIStartedSuccessfully(success))
	assert.False(t, VSIStartedSuccessfully(lines))
	assert.True(t, VSIFailedToStart(failure))
	assert.True(t, VSIFailedToStart(lines))
}

func TestSuccessfulStartup(t *testing.T) {
	lines := F.Pipe1(
		strings.Split(successLog, "\n"),
		A.Map(strings.TrimSpace),
	)

	success, _ := PartitionLogs(lines)

	assert.True(t, VSIStartedSuccessfully(success))
	assert.True(t, VSIStartedSuccessfully(lines))
}

func TestPartitionSuccessfulStartup(t *testing.T) {

	lines := F.Pipe1(
		strings.Split(successLog, "\n"),
		A.Map(strings.TrimSpace),
	)

	success, failure := PartitionLogs(lines)

	// failures should be empty
	assert.Empty(t, failure)
	assert.NotEmpty(t, success)
}

func TestPartitionFailedStartup(t *testing.T) {

	lines := F.Pipe1(
		strings.Split(failureLog, "\n"),
		A.Map(strings.TrimSpace),
	)

	success, failure := PartitionLogs(lines)

	// failures should be empty
	assert.NotEmpty(t, failure)
	assert.NotEmpty(t, success)
}

// func TestDirectVolumeDownload(t *testing.T) {
// 	config, err := defaultSSHConfig("../.env")
// 	if err != nil {
// 		t.SkipNow()
// 	}

// 	getVolume := GetLoggingVolumeViaSSH(config)

// 	data, err := getVolume("/var/lib/libvirt/images/console-e140b66c-be72-4d49-9348-b0f4b658b073.log")
// 	require.NoError(t, err)

// 	fmt.Println(data)
// }

// func TestGetLoggingViaCommand(t *testing.T) {
// 	config, err := defaultSSHConfig("../.env")
// 	if err != nil {
// 		t.SkipNow()
// 	}

// 	data, err := getLoggingVolumeViaCommand(context.Background(), config, "../k8s-operator-hpcr.exe", "/var/lib/libvirt/images/console-e140b66c-be72-4d49-9348-b0f4b658b073.log")
// 	require.NoError(t, err)

// 	fmt.Println(data)
// }
