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
	"net"
	"strconv"
	"strings"
	"time"

	"os"
	"os/user"

	"github.com/digitalocean/go-libvirt/socket"
	"github.com/ibm-hyper-protect/hpcr-controller/env"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	v1 "k8s.io/api/core/v1"
)

const (
	defaultSSHPort  = 22
	dialTimeout     = 10 * time.Second
	defaultUnixSock = "/var/run/libvirt/libvirt-sock"

	// Environment variable names
	KeyHostname   = "HOSTNAME"
	KeyPrivateKey = "KEY"
	KeyPort       = "PORT"
	KeyKnownHosts = "KNOWN_HOSTS"
	KeyUser       = "USER"
)

type SSHConfig struct {
	Hostname   string   `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Port       int      `json:"port,omitempty" yaml:"port,omitempty"`
	User       string   `json:"user,omitempty" yaml:"user,omitempty"`
	KnownHosts []string `json:"knownHosts,omitempty" yaml:"knownHosts,omitempty"`
	Key        string   `json:"key,omitempty" yaml:"key,omitempty"`
}

type sshDialer struct {
	config *SSHConfig
}

func getHostname(config *SSHConfig) string {
	return config.Hostname
}

func getPort(config *SSHConfig) int {
	if config.Port <= 0 {
		return defaultSSHPort
	}
	return config.Port
}

func getHost(config *SSHConfig) string {
	return fmt.Sprintf("%s:%d", getHostname(config), getPort(config))
}

func getPrivateKey(config *SSHConfig) (ssh.Signer, error) {
	return ssh.ParsePrivateKey([]byte(config.Key))
}

func getUserName(config *SSHConfig) (string, error) {
	usr := config.User
	if len(usr) > 0 {
		return usr, nil
	}
	osUser, err := user.Current()
	if err != nil {
		return "", err
	}

	return osUser.Username, nil
}

func getHostKeyCallback(config *SSHConfig) (ssh.HostKeyCallback, error) {
	// if no keys are configured, fallback to ignore everything
	if len(config.KnownHosts) == 0 {
		return ssh.InsecureIgnoreHostKey(), nil // #nosec G106 ignoring host key on purpose
	}
	// make sure the temp dir exists
	tmpDir := os.TempDir()
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	// parse the hosts
	file, err := os.CreateTemp(os.TempDir(), "knownHosts")
	if err != nil {
		return nil, err
	}
	defer func() {
		file.Close() // #nosec: G104 - manually audited
		err := os.Remove(file.Name())
		if err != nil {
			log.Printf("Error during removal of [%s]: %v", file.Name(), err)
		}
	}()

	for _, host := range config.KnownHosts {
		_, err := fmt.Fprintln(file, host)
		if err != nil {
			return nil, err
		}
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	// construct the callback
	return knownhosts.New(file.Name())
}

func printBanner(msg string) error {
	log.Println(msg)
	return nil
}

func (dialer *sshDialer) Dial() (net.Conn, error) {
	// build the SSH config
	config := dialer.config

	origin := getHost(config)

	// detect the username
	username, err := getUserName(config)
	if err != nil {
		return nil, err
	}
	// host key
	hostKeyCallback, err := getHostKeyCallback(config)
	if err != nil {
		return nil, err
	}
	// private key
	signer, err := getPrivateKey(config)
	if err != nil {
		return nil, err
	}

	cfg := ssh.ClientConfig{
		User:            username,
		HostKeyCallback: hostKeyCallback,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		Timeout:         dialTimeout,
		BannerCallback:  printBanner,
	}

	sshClient, err := ssh.Dial("tcp", origin, &cfg)
	if err != nil {
		return nil, err
	}

	return sshClient.Dial("unix", defaultUnixSock)

}

// CreateSSHDialer produces a dialer that can connect to the given SSH config
func CreateSSHDialer(config *SSHConfig) socket.Dialer {
	return &sshDialer{config: config}
}

// GetSSHConfigFromEnvMap deserializes an SSH config from a set of (env) parameters
func GetSSHConfigFromEnvMap(envMap env.Environment) *SSHConfig {

	result := &SSHConfig{}

	result.Hostname = envMap[KeyHostname]
	result.Key = envMap[KeyPrivateKey]
	result.User = envMap[KeyUser]

	port, ok := envMap[KeyPort]
	if ok {
		result.Port, _ = strconv.Atoi(port)
	}

	hosts, ok := envMap[KeyKnownHosts]
	if ok {
		result.KnownHosts = strings.Split(hosts, "\n")
	}

	return result
}

// GetEnvMapFromSSHConfig serializes an SSH config into a string map
func GetEnvMapFromSSHConfig(config *SSHConfig) env.Environment {
	result := make(env.Environment)

	if len(config.Hostname) > 0 {
		result[KeyHostname] = config.Hostname
	}
	if len(config.Key) > 0 {
		result[KeyPrivateKey] = config.Key
	}
	if config.Port > 0 {
		result[KeyPort] = fmt.Sprintf("%d", config.Port)
	}
	if len(config.KnownHosts) > 0 {
		result[KeyKnownHosts] = strings.Join(config.KnownHosts, "\n")
	}
	if len(config.User) > 0 {
		result[KeyUser] = config.User
	}

	return result
}

// GetSSHConfigFromConfigMap deserializes an SSH config from a config map object
func GetSSHConfigFromConfigMap(configMap *v1.ConfigMap) *SSHConfig {
	return GetSSHConfigFromEnvMap(configMap.Data)
}
