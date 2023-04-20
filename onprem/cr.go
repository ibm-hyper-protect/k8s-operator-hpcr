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
	CTR "github.com/ibm-hyper-protect/k8s-operator-hpcr/contract"
	"github.com/ibm-hyper-protect/k8s-operator-hpcr/env"
	C "github.com/ibm-hyper-protect/terraform-provider-hpcr/contract"
	B "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/bytes"
	E "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/either"
	F "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/function"
	I "github.com/ibm-hyper-protect/terraform-provider-hpcr/fp/identity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OnPremCustomResourceSpec struct {
	// the encrypted contract document
	Contract string `json:"contract"`
	// URL to the service that serves the base qcow2 image
	ImageURL string `json:"imageURL"`
	// name of the storage pool, must exist and must be large enough
	StoragePool string `json:"storagePool"`
	// specification of the associated config maps
	TargetSelector *metav1.LabelSelector `json:"targetSelector"`
	// specification of the associated data disks
	DiskSelector *metav1.LabelSelector `json:"diskSelector"`
}

type DataDiskCustomResourceSpec struct {
	// size of the data disk, defaults to 100GiB
	Size uint64 `json:"size"`
	// name of the storage pool, must exist and must be large enough
	StoragePool string `json:"storagePool"`
	// specification of the associated config maps
	TargetSelector *metav1.LabelSelector `json:"targetSelector"`
}

type DataDiskStatus struct {
	// description of the data disk status
	Description string `json:"description"`
	// the status flag
	Status int `json:"status"`
}

type OnPremCustomResource struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec OnPremCustomResourceSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type DataDiskCustomResource struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec DataDiskCustomResourceSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// status of this custom resource
	Status DataDiskStatus `json:"status,omitempty"`
}

type OnPremCustomResourceOptions struct {
	// name of the instance, will also be the hostname
	Name string
	// labels
	Labels map[string]string
	// references to the configs (for SSH)
	TargetLabels map[string]string
	// URL to the HPCR qcow2
	ImageURL string
	// name of the libvirt storage pool, the pool must exist
	StoragePool string
	// encryption Certificate
	EncryptionCert []byte
	// clear text contract
	Contract C.RawMap
}

type OnPremCustomResourceEnvOptions struct {
	// name of the instance, will also be the hostname
	Name string
	// labels
	Labels map[string]string
	// references to the configs (for SSH)
	TargetLabels map[string]string
	// URL to the HPCR qcow2
	ImageURL string
	// name of the libvirt storage pool, the pool must exist
	StoragePool string
	// encryption Certificate
	EncryptionCert []byte
	// folder containing the compose file
	ComposeFolder string
}

// CreateCustomResource creates a custom resource from a contract
func CreateCustomResource(opt *OnPremCustomResourceOptions) E.Either[error, *OnPremCustomResource] {
	// load the encryption certificate and create the encryption callback
	userData := F.Pipe5(
		opt.EncryptionCert,
		CTR.EncryptContract,
		I.Ap[C.RawMap, E.Either[error, C.RawMap]](opt.Contract),
		C.MapRefRawMapE,
		E.Chain(C.StringifyRawMapE),
		E.Map[error](B.ToString),
	)
	// produce the resource
	return F.Pipe1(
		userData,
		E.Map[error](func(contract string) *OnPremCustomResource {
			return &OnPremCustomResource{
				TypeMeta: metav1.TypeMeta{
					Kind:       KindVSI,
					APIVersion: APIVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   opt.Name,
					Labels: opt.Labels,
				},
				Spec: OnPremCustomResourceSpec{
					Contract:    contract,
					ImageURL:    opt.ImageURL,
					StoragePool: opt.StoragePool,
					TargetSelector: &metav1.LabelSelector{
						MatchLabels: opt.TargetLabels,
					},
				},
			}
		}),
	)

}

// CreateCustomResourceFromEnv creates a custom resource from some environment
func CreateCustomResourceFromEnv(envMap env.Environment) func(opt *OnPremCustomResourceEnvOptions) E.Either[error, *OnPremCustomResource] {
	// contract callback
	createContract := CTR.CreateContract(envMap)

	return func(opt *OnPremCustomResourceEnvOptions) E.Either[error, *OnPremCustomResource] {
		return F.Pipe3(
			opt.ComposeFolder,
			createContract,
			E.Map[error](func(contract C.RawMap) *OnPremCustomResourceOptions {
				return &OnPremCustomResourceOptions{
					Name:           opt.Name,
					ImageURL:       opt.ImageURL,
					Labels:         opt.Labels,
					TargetLabels:   opt.TargetLabels,
					StoragePool:    opt.StoragePool,
					EncryptionCert: opt.EncryptionCert,
					Contract:       contract,
				}
			}),
			E.Chain(CreateCustomResource),
		)
	}
}
