// Copyright 2023 IBM Corp.
//
// Licensed under the Ache License, Version 2.0 (the "License");
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"os/exec"

	CM "github.com/ibm-hyper-protect/k8s-operator-hpcr/common"
	"golang.org/x/crypto/ssh"
	"libvirt.org/go/libvirtxml"
)

const (
	// some sensible value for the size of the logging volume
	maxLoggingVolumeSize = uint64(2 * 1024 * 1024)
	maxDownloadTimeout   = 5 * time.Second
)

var (
	// expression to match tokens
	reSuccessToken        = regexp.MustCompile(`HPL\d+I`)
	reErrorToken          = regexp.MustCompile(`HPL\d+E`)
	reStartedSuccessfully = regexp.MustCompile(`(HPL10001I)|(VSI has started successfully)`)
)

// createLoggingVolumeDef creates the XML for the logging
func createLoggingVolumeDef(name string) *libvirtxml.StorageVolume {
	return &libvirtxml.StorageVolume{
		Name: name,
		Capacity: &libvirtxml.StorageVolumeSize{
			Value: maxLoggingVolumeSize,
		},
	}
}

// CreateLoggingVolume creates a logging volume for the console log
func CreateLoggingVolume(client *LivirtClient) func(storagePool, name string) (*libvirtxml.StorageVolume, error) {
	conn := client.LibVirt
	storageVolByNameXMLDesc := getStorageVolByNameXMLDesc(conn)
	storageVolXMLDesc := getStorageVolXMLDesc(conn)

	return func(storagePool, name string) (*libvirtxml.StorageVolume, error) {
		// access the pool
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			return nil, err
		}
		// check if we already know the volume
		_, err = storageVolByNameXMLDesc(pool, name)
		if err == nil {
			// we need to delete the volume
			_, err := deleteStorageVol(conn)(pool, name)
			if err != nil {
				return nil, err
			}
		}
		// Refresh the pool
		err = refreshPool(conn)(pool)
		if err != nil {
			return nil, err
		}
		// new volume
		volumeDef := createLoggingVolumeDef(name)
		volumeDefXML, err := XMLMarshall(volumeDef)
		if err != nil {
			return nil, err
		}

		// create the volume
		volume, err := conn.StorageVolCreateXML(pool, string(volumeDefXML), 0)
		if err != nil {
			return nil, err
		}

		// refresh the description
		return storageVolXMLDesc(&volume)

	}
}

// GetLoggingVolume retrieves the value of the logging volume
// the HPCR console log is very small by design, so passing it as a string does make sense
func GetLoggingVolume(client *LivirtClient) func(storagePool, name string) (string, error) {
	conn := client.LibVirt

	return func(storagePool, name string) (string, error) {
		msg := fmt.Sprintf("GetLoggingVolume(%s, %s)", storagePool, name)

		defer CM.PanicAfterTimeout(msg, maxDownloadTimeout)()
		defer CM.EntryExit(msg)()
		// access the pool
		log.Printf("Looking up storage pool [%s] by name ...", name)
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			log.Printf("Error looking up storage pool [%s] by name, cause: [%v]", storagePool, err)
			return "", err
		}
		log.Printf("Lookup up of storage pool [%s] was successful.", pool.Name)

		// go for the volume
		log.Printf("Looking up volume [%s] by name in pool [%s] ...", name, pool.Name)
		vol, err := conn.StorageVolLookupByName(pool, name)
		if err != nil {
			log.Printf("Error looking up volume [%s] by name in pool [%s], cause: [%v]", name, pool.Name, err)
			return "", err
		}
		log.Printf("Lookup up of volume [%s] by name in pool [%s] was successful.", vol.Name, pool.Name)

		// load the value of the logging volume
		var buffer bytes.Buffer
		log.Printf("Downloading volume [%s] ...", vol.Key)
		err = conn.StorageVolDownload(vol, &buffer, 0, maxLoggingVolumeSize, 0)
		if err != nil {
			log.Printf("Error downloading volume [%s], cause: [%v]", vol.Key, err)
			return "", err
		}
		log.Printf("Download of volume [%s] was successful", vol.Key)
		// returns the content of the logs
		return buffer.String(), nil
	}
}

// PartitionLogs partitions the original logs into success and error logs
func PartitionLogs(logs []string) ([]string, []string) {
	var success, failure []string
	for _, line := range logs {
		if reSuccessToken.MatchString(line) {
			success = append(success, line)
		}
		if reErrorToken.MatchString(line) {
			failure = append(failure, line)
		}
	}
	return success, failure
}

// VSIStartedSuccessfully tests if the VSI started successfully
func VSIStartedSuccessfully(logs []string) bool {
	// check if the logs contain an indication for successful startup
	for _, line := range logs {
		if reStartedSuccessfully.MatchString(line) {
			return true
		}
	}
	return false
}

// VSIFailedToStart tests if the VSI failed to start
func VSIFailedToStart(logs []string) bool {
	// check if the logs contain an for failure
	for _, line := range logs {
		if reErrorToken.MatchString(line) {
			return true
		}
	}
	return false
}

// GetLoggingVolumeViaSSH retrieves the value of the logging volume via a new and direct SSH connection
// the HPCR console log is very small by design, so passing it as a string does make sense
func GetLoggingVolumeViaSSH(config *SSHConfig) func(path string) (string, error) {

	return func(path string) (string, error) {
		msg := fmt.Sprintf("GetLoggingVolumeViaSSH(%s)", path)
		defer CM.PanicAfterTimeout(msg, maxDownloadTimeout)()
		defer CM.EntryExit(msg)()

		origin := getHost(config)

		// detect the username
		username, err := getUserName(config)
		if err != nil {
			return "", err
		}
		// host key
		hostKeyCallback, err := getHostKeyCallback(config)
		if err != nil {
			return "", err
		}
		// private key
		signer, err := getPrivateKey(config)
		if err != nil {
			log.Printf("Unable to get private key, cause: [%v]", err)
			return "", err
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
			log.Printf("Unable to create SSH client, cause: [%v]", err)
			return "", err
		}
		defer sshClient.Close()

		session, err := sshClient.NewSession()
		if err != nil {
			log.Printf("Unable to create SSH session, cause: [%v]", err)
			return "", err
		}
		defer session.Close()

		// capture the output of the session
		var buffer bytes.Buffer
		session.Stdout = &buffer

		log.Printf("Downloading volume [%s] ...", path)
		if err := session.Run(fmt.Sprintf("/usr/bin/cat \"%s\"", path)); err != nil {
			log.Printf("Unable to cat [%s], cause: [%v]", path, err)
			return "", err
		}
		log.Printf("Download of volume [%s] was successful", path)

		return buffer.String(), nil
	}
}

// getLoggingVolumeViaSSH retrieves the value of the logging volume by spawning a separate command. The advantage of this approach is
// that that command can be canceled if it times out
// the HPCR console log is very small by design, so passing it as a string does make sense
func getLoggingVolumeViaCommand(ctx context.Context, config *SSHConfig, command string, path string) (string, error) {
	msg := fmt.Sprintf("getLoggingVolumeViaCommand(%s, %s)", command, path)
	defer CM.EntryExit(msg)()

	// marshal the ssh config
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Printf("Unable to marshal SSH config, cause: [%v]", err)
		return "", err
	}

	// start the command with a proper timeout
	withTimeout, cancelTimeout := context.WithTimeout(ctx, maxDownloadTimeout)
	defer cancelTimeout()

	var buffer bytes.Buffer

	cmd := exec.CommandContext(withTimeout, command, "download", "--path", path)
	cmd.Stdin = bytes.NewReader(configBytes)
	cmd.Stdout = &buffer
	cmd.Stderr = os.Stderr

	log.Printf("Executing command [%s] ...", cmd.Path)
	err = cmd.Run()
	if err != nil {
		log.Printf("Error running command [%s], cause: [%v]", cmd.Path, err)
		return "", err
	}
	log.Printf("Execution of command [%s] was successful", cmd.Path)

	return buffer.String(), nil
}

// GetLoggingVolumeViaSSH retrieves the value of the logging volume by spawning a separate command. The advantage of this approach is
// that that command can be canceled if it times out
// the HPCR console log is very small by design, so passing it as a string does make sense
func GetLoggingVolumeViaCommand(client *LivirtClient) func(storagePool, name string) (string, error) {
	// config needed for further processing
	sshConfig := client.SSHConfig
	conn := client.LibVirt

	return func(storagePool, name string) (string, error) {
		msg := fmt.Sprintf("GetLoggingVolumeViaCommand(%s, %s)", storagePool, name)
		defer CM.EntryExit(msg)()

		executable, err := os.Executable()
		if err != nil {
			log.Printf("Unable to locate the current executable, cause: [%v]", err)
			return "", err
		}

		// access the pool
		log.Printf("Looking up storage pool [%s] by name ...", name)
		pool, err := conn.StoragePoolLookupByName(storagePool)
		if err != nil {
			log.Printf("Error looking up storage pool [%s] by name, cause: [%v]", storagePool, err)
			return "", err
		}
		log.Printf("Lookup up of storage pool [%s] was successful.", pool.Name)

		// go for the volume
		log.Printf("Looking up volume [%s] by name in pool [%s] ...", name, pool.Name)
		vol, err := conn.StorageVolLookupByName(pool, name)
		if err != nil {
			log.Printf("Error looking up volume [%s] by name in pool [%s], cause: [%v]", name, pool.Name, err)
			return "", err
		}
		log.Printf("Lookup up of volume [%s] by name in pool [%s] was successful.", vol.Name, pool.Name)

		return getLoggingVolumeViaCommand(context.Background(), sshConfig, executable, vol.Key)
	}
}
