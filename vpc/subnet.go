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
	"fmt"
	"log"
	"regexp"

	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

const (
	fieldRegion = "region"
)

var (
	// parses the region from a region or zone identifier
	reRegion = regexp.MustCompile(`^([a-zA-Z]+(?:-[a-zA-Z]+)+)(?:-\d+)?$`)
)

// FindRegionFromSubnet locates the region from a subnet
func FindRegionFromSubnet(search *globalsearchv2.GlobalSearchV2) func(subnetID string) (string, error) {
	searchAny := globalsearchv2.SearchOptionsIsPublicAnyConst
	limit := int64(1)

	return func(subnetID string) (string, error) {

		query := fmt.Sprintf("type:%s AND resource_id:%s AND service_name:is", vpcv1.SubnetResourceTypeSubnetConst, subnetID)

		res, _, err := search.Search(&globalsearchv2.SearchOptions{
			IsPublic: &searchAny,
			Limit:    &limit,
			Fields:   []string{fieldRegion},
			Query:    &query,
		})
		if err != nil {
			log.Printf("Error trying to search for subnet [%s], cause: [%v]", subnetID, err)
		}
		if len(res.Items) == 0 {
			log.Printf("Unable to locate subnet [%s]", subnetID)
			return "", fmt.Errorf("unable to locate subnet [%s]", subnetID)
		}
		// some debugging
		item := res.Items[0]
		log.Printf("Found subnet [%s]", *item.CRN)
		// read region
		region, ok := item.GetProperty(fieldRegion).(string)
		if !ok {
			return "", fmt.Errorf("unable to read region from subnet [%s]", subnetID)
		}
		// check
		m := reRegion.FindStringSubmatch(region)
		if m == nil {
			return "", fmt.Errorf("region identifier [%s] does not seem to be a region", region)
		}
		// done
		return m[1], nil
	}
}
