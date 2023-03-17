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
	"strings"

	"github.com/ibm-hyper-protect/k8s-operator-hpcr/onprem"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KeyConfig = "config"
)

func CreateSSHConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "ssh-config",
		Usage: "generates an SSH config map based on the environment",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     KeyConfig,
				Aliases:  []string{"c"},
				Usage:    "Name of the SSH config entry",
				Required: true,
			},
			&cli.StringFlag{
				Name:     KeyName,
				Aliases:  []string{"n"},
				Usage:    "Name the SSH config map",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:     KeyLabel,
				Aliases:  []string{"l"},
				Usage:    "Label for the config map",
				Required: false,
			},
		},
		Action: func(ctx *cli.Context) error {
			// find SSH path
			sshPath, err := onprem.GetSSHConfigPath()
			if err != nil {
				return err
			}
			// load config
			sshConfig, err := onprem.LoadSSHConfig(sshPath)(ctx.String("config"))
			if err != nil {
				return err
			}
			// convert from slice to map
			labels := make(map[string]string)
			for _, label := range ctx.StringSlice(KeyLabel) {
				split := strings.SplitN(label, ":", 2)
				if len(split) == 2 {
					labels[split[0]] = split[1]
				}
			}
			// produce the config map
			configMap := &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   ctx.String(KeyName),
					Labels: labels,
				},
				Data: onprem.GetEnvMapFromSSHConfig(sshConfig),
			}
			// serialize
			return json.NewEncoder(os.Stdout).Encode(configMap)
		},
	}
}
