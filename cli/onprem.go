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
	"os"

	"github.com/ibm-hyper-protect/hpcr-controller/common"
	"github.com/ibm-hyper-protect/hpcr-controller/env"
	"github.com/ibm-hyper-protect/hpcr-controller/onprem"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	J "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/json"
	"github.com/urfave/cli/v2"
)

const (
	DefaultStoragePool = "default"
	KeyTarget          = "target"
)

func CreateOnPremCommand() *cli.Command {
	return &cli.Command{
		Name:  "onprem",
		Usage: "generates a custom resource definition with a contract for onprem",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     KeyName,
				Aliases:  []string{"n"},
				Usage:    "Name the custom resource",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:     KeyLabel,
				Aliases:  []string{"l"},
				Usage:    "Label for the custom resource",
				Required: false,
			},
			&cli.StringFlag{
				Name:     KeyImageURL,
				Aliases:  []string{"i"},
				Usage:    "Download URL of the qcow2 file",
				Required: true,
			},
			&cli.StringFlag{
				Name:        KeyStoragePool,
				Aliases:     []string{"p"},
				Usage:       "Name of the storage pool",
				DefaultText: DefaultStoragePool,
				Required:    false,
			},
			&cli.PathFlag{
				Name:      KeyCertPath,
				Aliases:   []string{"c"},
				Usage:     "Path to the encryption certificate",
				TakesFile: true,
				Required:  true,
			},
			&cli.PathFlag{
				Name:      KeyComposeFolder,
				Aliases:   []string{"f"},
				Usage:     "Path to the compose folder",
				TakesFile: false,
				Required:  true,
			},
			&cli.StringSliceFlag{
				Name:     KeyTarget,
				Usage:    "Label used to select the associated config map(s)",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			// load env
			env := env.GetEnvAsMap(os.Environ())
			// prepare the inputs
			labels := labelsFromList(ctx.StringSlice(KeyLabel))
			targetLabels := labelsFromList(ctx.StringSlice(KeyTarget))
			name := ctx.String(KeyName)
			imageURL := ctx.String(KeyImageURL)
			storagePool := ctx.String(KeyStoragePool)
			compose := ctx.Path(KeyComposeFolder)
			cert, err := os.ReadFile(ctx.Path(KeyCertPath))
			if err != nil {
				return err
			}
			// the options
			opts := &onprem.OnPremCustomResourceEnvOptions{
				Name:           name,
				Labels:         labels,
				TargetLabels:   targetLabels,
				ImageURL:       imageURL,
				StoragePool:    storagePool,
				EncryptionCert: cert,
				ComposeFolder:  compose,
			}
			// construct the resource
			data, err := common.FromEither(F.Pipe2(
				opts,
				onprem.CreateCustomResourceFromEnv(env),
				E.Chain(J.Stringify[onprem.OnPremCustomResource]),
			))
			if err != nil {
				return err
			}
			// stream the result
			_, err = os.Stdout.Write(data)
			return err
		},
	}
}
