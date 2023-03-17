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
	"log"
	"testing"

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownHosts(t *testing.T) {
	config, err := defaultSSHConfig("../.env")
	if err != nil {
		t.SkipNow()
	}

	cb, err := getHostKeyCallback(config)
	require.NoError(t, err)
	assert.NotNil(t, cb)
}

func TestSSHConfig(t *testing.T) {
	config, err := defaultSSHConfig("../.env")
	if err != nil {
		t.SkipNow()
	}

	hostname := getHostname(config)
	assert.Equal(t, config.Hostname, hostname)

	user, err := getUserName(config)
	require.NoError(t, err)
	assert.Equal(t, config.User, user)

	host := getHost(config)
	assert.Equal(t, fmt.Sprintf("%s:22", config.Hostname), host)
}

func TestSSHConnection(t *testing.T) {
	config, err := defaultSSHConfig("../.env")
	if err != nil {
		t.SkipNow()
	}

	dialer := CreateSSHDialer(config)

	l := libvirt.NewWithDialer(dialer)
	err = l.ConnectToURI(libvirt.ConnectURI(""))
	require.NoError(t, err)

	v, err := l.ConnectGetLibVersion()
	require.NoError(t, err)
	log.Printf("[INFO] libvirt client libvirt version: %v\n", v)

	err = l.ConnectClose()
	require.NoError(t, err)
}

func TestSerializeToMap(t *testing.T) {
	config, err := defaultSSHConfig("../.env")
	if err != nil {
		t.SkipNow()
	}

	env := GetEnvMapFromSSHConfig(config)
	deser := GetSSHConfigFromEnvMap(env)

	assert.Equal(t, config, deser)

}
