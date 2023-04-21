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

const (
	DefaultStoragePool  = "default"
	DefaultDataDiskSize = uint64(100 * 1024 * 1024 * 1024)

	userDataFilename   = "user-data"
	metaDataFilename   = "meta-data"
	vendorDataFilename = "vendor-data"
	ciDataVolumeName   = "cidata"

	APIVersion      = "hpse.ibm.com/v1"
	KindVSI         = "HyperProtectContainerRuntimeOnPrem"
	KindDataDisk    = "HyperProtectContainerRuntimeOnPremDataDisk"
	KindDataDiskRef = "HyperProtectContainerRuntimeOnPremDataDiskRef"
	KindNetworkRef  = "HyperProtectContainerRuntimeOnPremNetworkRef"

	ResourceNameDataDisks    = "onprem-datadisks"
	ResourceNameDataDiskRefs = "onprem-datadiskrefs"
	ResourceNameNetworkRefs  = "onprem-networkrefs"
	ResourceNameVSIs         = "onprem-hpcrs"
)
