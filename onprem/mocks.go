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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"

	B "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/bytes"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/contract"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
)

const (
	// KeyOnPremConfig is the key into the environment to read the name of the ssh config
	KeyOnPremConfig = "ONPREM_CONFIG"
	// KeyStoragePool is the key into the environment to read the name of the target storage pool
	KeyStoragePool = "STORAGE_POOL"
)

// GetSSHConfigPath returns the path to the SSH config file
func GetSSHConfigPath() (string, error) {
	return homedir.Expand("~/.ssh/config")
}

// LoadSSHConfig loads the SSH config file
func LoadSSHConfig(configFile string) func(configName string) (*SSHConfig, error) {

	return func(configName string) (*SSHConfig, error) {

		cfgFile, err := os.Open(filepath.Clean(configFile))
		if err != nil {
			return nil, err
		}
		defer safeClose(cfgFile)

		cfg, err := ssh_config.Decode(cfgFile)
		if err != nil {
			return nil, err
		}

		// prepare the config
		sshConfig := &SSHConfig{}

		// populate the config
		port, err := cfg.Get(configName, "Port")
		if (err == nil) && len(port) > 0 {
			intPort, err := strconv.Atoi(port)
			if err != nil {
				return nil, err
			}
			sshConfig.Port = intPort
		}

		hostname, err := cfg.Get(configName, "Hostname")
		if (err == nil) && len(hostname) > 0 {
			sshConfig.Hostname = hostname
		} else {
			sshConfig.Hostname = configName
		}

		user, err := cfg.Get(configName, "User")
		if (err == nil) && len(user) > 0 {
			sshConfig.User = user
		}

		identityFile, err := cfg.Get(configName, "IdentityFile")
		if (err == nil) && len(identityFile) > 0 {
			// try to read the file
			resolved, err := homedir.Expand(identityFile)
			if err != nil {
				return nil, err
			}
			auth, err := os.ReadFile(filepath.Clean(resolved))
			if err != nil {
				return nil, err
			}
			// update the content
			sshConfig.Key = string(auth)
		}

		knownHosts, err := cfg.Get(configName, "UserKnownHostsFile")
		if err == nil {
			// fallback to the regular known hosts file
			if len(knownHosts) <= 0 {
				knownHosts = "~/.ssh/known_hosts"
			}
			// try to read the file
			resolved, err := homedir.Expand(knownHosts)
			if err != nil {
				return nil, err
			}
			auth, err := os.ReadFile(filepath.Clean(resolved))
			if err != nil {
				return nil, err
			}
			// split hosts by newline
			hosts := strings.Split(string(auth), "\n")
			sshConfig.KnownHosts = hosts
		}

		return sshConfig, nil
	}
}

// defaultSSHConfig loads the SSH config to connect to from an environment variable
// the config is read from '~/.ssh/config', the name of the config is read from the environment variable `envVarConfig`
func defaultSSHConfig(filenames ...string) (*SSHConfig, error) {
	// load env
	env, err := godotenv.Read(filenames...)
	if err != nil {
		return nil, err
	}
	return getSSHConfigFromEnv(env)
}

// defaultSSHConfig loads the SSH config to connect to from an environment variable
// the config is read from '~/.ssh/config', the name of the config is read from the environment variable `envVarConfig`
func getSSHConfigFromEnv(envMap env.Environment) (*SSHConfig, error) {
	// find config name
	name, ok := envMap[KeyOnPremConfig]
	if !ok {
		return nil, fmt.Errorf("unable to find the %s key", KeyOnPremConfig)
	}
	// find SSH path
	sshPath, err := GetSSHConfigPath()
	if err != nil {
		return nil, err
	}
	// load config
	return LoadSSHConfig(sshPath)(name)
}

func getEncryptedBusyboxContract(envMap env.Environment) E.Either[error, string] {
	// load the encryption certificate and create the encryption callback
	enc := F.Pipe2(
		envMap,
		contract.LoadPublicKeyFromEnv,
		E.Map[error](contract.EncryptContract),
	)

	ctr := F.Pipe2(
		envMap,
		contract.CreateBusyboxContract,
		E.Chain(contract.ValidateContract),
	)

	return F.Pipe5(
		enc,
		E.Ap[error, C.RawMap, E.Either[error, C.RawMap]](ctr),
		E.Flatten[error, C.RawMap],
		C.MapRefRawMapE,
		E.Chain(C.StringifyRawMapE),
		E.Map[error](B.ToString),
	)

}
