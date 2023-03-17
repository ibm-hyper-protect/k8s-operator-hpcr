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

	libvirt "github.com/digitalocean/go-libvirt"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
)

type LivirtClient struct {
	io.Closer
	LibVirt *libvirt.Libvirt
	Hash    string
}

func (client *LivirtClient) Close() error {
	return client.LibVirt.ConnectClose()
}

// CreateLivirtClient creates a libvirt connection based on an SSH config
func CreateLivirtClient(sshConfig *SSHConfig) (*LivirtClient, error) {

	dialer := CreateSSHDialer(sshConfig)

	// construct the client
	l := libvirt.NewWithDialer(dialer)
	// TODO do we need to be able to pass a sub identifier of the libvirt instance
	err := l.ConnectToURI(libvirt.ConnectURI(""))
	if err != nil {
		return nil, err
	}

	// build the hash
	hash := getHost(sshConfig)

	return &LivirtClient{
		LibVirt: l,
		Hash:    hash,
	}, nil
}

// CreateLivirtClientFromEnvMap constructs the libvirt client from an env map
func CreateLivirtClientFromEnvMap(envMap env.Environment) (*LivirtClient, error) {
	// just dispatch
	return CreateLivirtClient(GetSSHConfigFromEnvMap(envMap))
}
