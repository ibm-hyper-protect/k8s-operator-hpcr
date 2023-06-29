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

package cli

import (
	"encoding/json"
	"os"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	c "github.com/urfave/cli/v2"
)

const (
	pathFlagName = "path"
)

// DownloadCommand downloads a volume via ssh from a remote server
func DownloadCommand(version, compiled, commit string) *c.Command {
	return &c.Command{
		Name:        "download",
		Usage:       "Downloads the content of a volume from a remote server via SSH",
		Description: "Downloads the content of a volume from a remote server via SSH",
		Flags: []c.Flag{
			&c.StringFlag{
				Name:     pathFlagName,
				Usage:    "Path of the file to fetch",
				Required: true,
				Aliases:  []string{"p"},
			},
		},
		Action: func(ctx *c.Context) error {
			// path to download
			path := ctx.String(pathFlagName)
			// unmarshal the ssh config from stdin
			dec := json.NewDecoder(os.Stdin)
			var sshConfig onprem.SSHConfig
			if err := dec.Decode(&sshConfig); err != nil {
				return err
			}
			// download the volume
			getVolume := onprem.GetLoggingVolumeViaSSH(&sshConfig)
			content, err := getVolume(path)
			if err != nil {
				return err
			}
			// write content to stdout and exit normally
			if _, err := os.Stdout.WriteString(content); err != nil {
				return err
			}
			// success
			return nil
		},
	}
}
