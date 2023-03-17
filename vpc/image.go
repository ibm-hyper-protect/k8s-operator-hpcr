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

package vpc

import (
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/Masterminds/semver"
)

type Image struct {
	ID      string
	Version *semver.Version
}

var reImgName = regexp.MustCompile(`^ibm-hyper-protect-container-runtime-(\d+)-(\d+)-s390x-(\d+)$`)

var ErrorStockImageNotFound = errors.New("stock image was not found")

type byVersion []Image

func (vs byVersion) Len() int      { return len(vs) }
func (vs byVersion) Swap(i, j int) { vs[i], vs[j] = vs[j], vs[i] }
func (vs byVersion) Less(j, i int) bool {
	return vs[i].Version.LessThan(vs[j].Version)
}

func FindStockImages(service *vpcv1.VpcV1) ([]Image, error) {
	vis := vpcv1.ListImagesOptionsVisibilityPublicConst
	pager, err := service.NewImagesPager(&vpcv1.ListImagesOptions{Visibility: &vis})
	if err != nil {
		return nil, err
	}
	all, err := pager.GetAll()
	if err != nil {
		return nil, err
	}
	// result
	var res []Image
	for _, img := range all {
		// check for a match
		sub := reImgName.FindStringSubmatch(*img.Name)
		if sub != nil && *img.Status == vpcv1.ImageStatusAvailableConst {
			res = append(res, Image{ID: *img.ID, Version: semver.MustParse(fmt.Sprintf("v%s.%s.%s", sub[1], sub[2], sub[3]))})
		}
	}
	// sort
	sort.Sort(byVersion(res))
	return res, err
}

func FindLatestStockImage(service *vpcv1.VpcV1) (string, error) {
	images, err := FindStockImages(service)
	if err != nil {
		return "", err
	}
	if len(images) == 0 {
		return "", ErrorStockImageNotFound
	}
	// return the first image
	return images[0].ID, nil
}

func Findimage(service *vpcv1.VpcV1, name string) (string, error) {
	pager, err := service.NewImagesPager(&vpcv1.ListImagesOptions{Name: &name})
	if err != nil {
		return "", err
	}
	all, err := pager.GetAll()
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", fmt.Errorf("Image with name [%s] could not be found", name)
	}
	return *all[0].ID, nil
}
