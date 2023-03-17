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
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/common"

	E "github.com/ibm-hyper-protect/hpcr-controller/env"
)

const (
	KeyIBMCloudGtApiEndpoint     = "IBMCLOUD_GT_API_ENDPOINT"
	DefaultIBMCloudGtApiEndpoint = "https://tags.global-search-tagging.cloud.ibm.com"
)

func GetIBMCloudGtApiEndpoint(env E.Environment) string {
	endpoint, ok := env[KeyIBMCloudGtApiEndpoint]
	if ok {
		return endpoint
	}
	return DefaultIBMCloudGtApiEndpoint
}

type GlobalTaggingOptions struct {
	URL           string
	Authenticator core.Authenticator

	// The API version, in format `YYYY-MM-DD`. For the API behavior documented here, specify any date between `2022-09-13`
	// and today's date (UTC).
	Version *string
}

type GlobalTagging struct {
	Service *core.BaseService

	// The API version, in format `YYYY-MM-DD`. For the API behavior documented here, specify any date between `2022-09-13`
	// and today's date (UTC).
	Version *string

	// The infrastructure generation. For the API behavior documented here, specify
	// `2`.
	generation *int64
}

// CreateGlobalTagging : constructs an Tag of GlobalTagging with passed in options.
func CreateGlobalTagging(options *GlobalTaggingOptions) (service *GlobalTagging, err error) {
	serviceOptions := &core.ServiceOptions{
		URL:           DefaultIBMCloudGtApiEndpoint,
		Authenticator: options.Authenticator,
	}

	err = core.ValidateStruct(options, "options")
	if err != nil {
		return
	}

	baseService, err := core.NewBaseService(serviceOptions)
	if err != nil {
		return
	}

	if options.URL != "" {
		err = baseService.SetServiceURL(options.URL)
		if err != nil {
			return
		}
	}

	if options.Version == nil {
		options.Version = core.StringPtr("2022-09-13")
	}

	service = &GlobalTagging{
		Service:    baseService,
		Version:    options.Version,
		generation: core.Int64Ptr(2),
	}

	return
}

// ListTagsOptions : The ListTags options.
type ListTagsOptions struct {
	// A server-provided token determining what resource to start the page on.
	Start *string `json:"start,omitempty"`

	// The number of resources to return on a page.
	Limit *int64 `json:"limit,omitempty"`

	// The type of the tag you want to list. Supported values are user, service and access.
	TagType *string `json:"tag_type,omitempty"`

	// If you want to return only the list of tags that are attached to a specified resource, pass the ID of the resource on this parameter. For resources that are onboarded to Global Search and Tagging, the resource ID is the CRN; for IMS resources, it is the IMS ID. When using this parameter, you must specify the appropriate provider (ims or ghost).
	AttachedTo *string `json:"attached_to,omitempty"`

	// Select a provider. Supported values are ghost and ims. To list both Global Search and Tagging tags and infrastructure tags, use ghost,ims. service and access tags can only be attached to resources that are onboarded to Global Search and Tagging, so you should not set this parameter to list them.
	Providers []string `json:"providers,omitempty"`

	// Allows users to set headers on API requests
	Headers map[string]string
}

// TagCollectionFirst : A link to the first page of resources.
type TagCollectionFirst struct {
	// The URL for a page of resources.
	Href *string `json:"href" validate:"required"`
}

// UnmarshalTagCollectionFirst unmarshals an instance of TagCollectionFirst from the specified map of raw messages.
func UnmarshalTagCollectionFirst(m map[string]json.RawMessage, result any) (err error) {
	obj := new(TagCollectionFirst)
	err = core.UnmarshalPrimitive(m, "href", &obj.Href)
	if err != nil {
		return
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(obj))
	return
}

// TagCollectionNext : A link to the next page of resources. This property is present for all pages except the last page.
type TagCollectionNext struct {
	// The URL for a page of resources.
	Href *string `json:"href" validate:"required"`
}

// UnmarshalTagCollectionNext unmarshals an instance of TagCollectionNext from the specified map of raw messages.
func UnmarshalTagCollectionNext(m map[string]json.RawMessage, result any) (err error) {
	obj := new(TagCollectionNext)
	err = core.UnmarshalPrimitive(m, "href", &obj.Href)
	if err != nil {
		return
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(obj))
	return
}

// TagCollection : TagCollection struct
type TagCollection struct {
	// A link to the first page of resources.
	First *TagCollectionFirst `json:"first" validate:"required"`

	// Collection of virtual server Tags.
	Tags []Tag `json:"Tags" validate:"required"`

	// The maximum number of resources that can be returned by the request.
	Limit *int64 `json:"limit" validate:"required"`

	// A link to the next page of resources. This property is present for all pages
	// except the last page.
	Next *TagCollectionNext `json:"next,omitempty"`

	// The total number of resources across all pages.
	TotalCount *int64 `json:"total_count" validate:"required"`
}

// UnmarshalTagCollection unmarshals an instance of TagCollection from the specified map of raw messages.
func UnmarshalTagCollection(m map[string]json.RawMessage, result any) (err error) {
	obj := new(TagCollection)
	err = core.UnmarshalModel(m, "first", &obj.First, UnmarshalTagCollectionFirst)
	if err != nil {
		return
	}
	err = core.UnmarshalModel(m, "items", &obj.Tags, UnmarshalTag)
	if err != nil {
		return
	}
	err = core.UnmarshalPrimitive(m, "limit", &obj.Limit)
	if err != nil {
		return
	}
	err = core.UnmarshalModel(m, "next", &obj.Next, UnmarshalTagCollectionNext)
	if err != nil {
		return
	}
	err = core.UnmarshalPrimitive(m, "total_count", &obj.TotalCount)
	if err != nil {
		return
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(obj))
	return
}

// TagsPager can be used to simplify the use of the "ListTags" method.
type TagsPager struct {
	hasNext     bool
	options     *ListTagsOptions
	client      *GlobalTagging
	pageContext struct {
		next *string
	}
}

type Tag struct {
	Name string `json:"name"`
}

// UnmarshalTag unmarshals an instance of Instance from the specified map of raw messages.
func UnmarshalTag(m map[string]json.RawMessage, result any) (err error) {
	obj := new(Tag)
	err = core.UnmarshalPrimitive(m, "name", &obj.Name)
	if err != nil {
		return
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(obj))
	return
}

// GetEnableGzipCompression returns the service's EnableGzipCompression field
func (service *GlobalTagging) GetEnableGzipCompression() bool {
	return service.Service.GetEnableGzipCompression()
}

// NewTagsPager returns a new TagsPager Tag.
func (service *GlobalTagging) NewTagsPager(options *ListTagsOptions) (pager *TagsPager, err error) {
	if options.Start != nil && *options.Start != "" {
		err = fmt.Errorf("the 'options.Start' field should not be set")
		return
	}

	var optionsCopy ListTagsOptions = *options
	pager = &TagsPager{
		hasNext: true,
		options: &optionsCopy,
		client:  service,
	}
	return
}

// HasNext returns true if there are potentially more results to be retrieved.
func (pager *TagsPager) HasNext() bool {
	return pager.hasNext
}

// GetNextWithContext returns the next page of results using the specified Context.
func (pager *TagsPager) GetNextWithContext(ctx context.Context) (page []Tag, err error) {
	if !pager.HasNext() {
		return nil, fmt.Errorf("no more results available")
	}

	pager.options.Start = pager.pageContext.next

	result, _, err := pager.client.ListTagsWithContext(ctx, pager.options)
	if err != nil {
		return
	}

	var next *string
	if result.Next != nil {
		var start *string
		start, err = core.GetQueryParam(result.Next.Href, "start")
		if err != nil {
			err = fmt.Errorf("error retrieving 'start' query parameter from URL '%s': %s", *result.Next.Href, err.Error())
			return
		}
		next = start
	}
	pager.pageContext.next = next
	pager.hasNext = (pager.pageContext.next != nil)
	page = result.Tags

	return
}

// GetAllWithContext returns all results by invoking GetNextWithContext() repeatedly
// until all pages of results have been retrieved.
func (pager *TagsPager) GetAllWithContext(ctx context.Context) (allItems []Tag, err error) {
	for pager.HasNext() {
		var nextPage []Tag
		nextPage, err = pager.GetNextWithContext(ctx)
		if err != nil {
			return
		}
		allItems = append(allItems, nextPage...)
	}
	return
}

// GetNext invokes GetNextWithContext() using context.Background() as the Context parameter.
func (pager *TagsPager) GetNext() (page []Tag, err error) {
	return pager.GetNextWithContext(context.Background())
}

// GetAll invokes GetAllWithContext() using context.Background() as the Context parameter.
func (pager *TagsPager) GetAll() (allItems []Tag, err error) {
	return pager.GetAllWithContext(context.Background())
}

// ListTags : List all Tags
// This request lists all Tags in the region.
func (service *GlobalTagging) ListTags(listTagsOptions *ListTagsOptions) (result *TagCollection, response *core.DetailedResponse, err error) {
	return service.ListTagsWithContext(context.Background(), listTagsOptions)
}

// ListTagsWithContext is an alternate form of the ListTags method which supports a Context parameter
func (service *GlobalTagging) ListTagsWithContext(ctx context.Context, listTagsOptions *ListTagsOptions) (result *TagCollection, response *core.DetailedResponse, err error) {
	err = core.ValidateStruct(listTagsOptions, "ListTagsOptions")
	if err != nil {
		return
	}

	builder := core.NewRequestBuilder(core.GET)
	builder = builder.WithContext(ctx)
	builder.EnableGzipCompression = service.GetEnableGzipCompression()
	_, err = builder.ResolveRequestURL(service.Service.Options.URL, `/tags`, nil)
	if err != nil {
		return
	}

	for headerName, headerValue := range listTagsOptions.Headers {
		builder.AddHeader(headerName, headerValue)
	}

	sdkHeaders := common.GetSdkHeaders("vpc", "V1", "ListTags")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	builder.AddHeader("Accept", "application/json")

	builder.AddQuery("version", fmt.Sprint(*service.Version))
	builder.AddQuery("generation", fmt.Sprint(*service.generation))
	if listTagsOptions.Start != nil {
		builder.AddQuery("start", fmt.Sprint(*listTagsOptions.Start))
	}
	if listTagsOptions.Limit != nil {
		builder.AddQuery("limit", fmt.Sprint(*listTagsOptions.Limit))
	}
	if listTagsOptions.TagType != nil {
		builder.AddQuery("tag_type", fmt.Sprint(*listTagsOptions.TagType))
	}
	if listTagsOptions.AttachedTo != nil {
		builder.AddQuery("attached_to", fmt.Sprint(*listTagsOptions.AttachedTo))
	}
	if listTagsOptions.Providers != nil && len(listTagsOptions.Providers) > 0 {
		builder.AddQuery("providers", strings.Join(listTagsOptions.Providers, ","))
	}

	request, err := builder.Build()
	if err != nil {
		return
	}

	var rawResponse map[string]json.RawMessage
	response, err = service.Service.Request(request, &rawResponse)
	if err != nil {
		return
	}
	if rawResponse != nil {
		err = core.UnmarshalModel(rawResponse, "", &result, UnmarshalTagCollection)
		if err != nil {
			return
		}
		response.Result = result
	}

	return
}

type Resource struct {
	// The CRN or IMS ID of the resource
	ResourceID string `json:"resource_id"`
	// The IMS resource type of the resource
	ResourceType *string `json:"resource_type,omitempty"`
}

type AttachTagsBody struct {
	// List of resources on which the tag or tags are attached.
	Resources []Resource `json:"resources"`
	// An array of tag names to attach
	TagNames []string `json:"tag_names"`
}

type TagResultsItem struct {
	// The CRN or IMS ID of the resource
	ResourceID string `json:"resource_id"`
	// It is true if the operation exits with an error.
	IsError bool `json:"is_error"`
}

type AttachTagsResponse struct {
	Results []TagResultsItem `json:"results"`
}

const (
	AttachTagsOptionsTagTypeService = "service"
	AttachTagsOptionsTagTypeUser    = "user"
)

// AttachTagsOptions : The AttachTags options.
type AttachTagsOptions struct {

	// The type of the tag. Supported values are user, service and access. service and access are not supported for IMS resources.
	TagType *string `json:"tag_type,omitempty"`

	// Allows users to set headers on API requests
	Headers map[string]string
}

// UnmarshalTagResultsItem unmarshals an instance of VPC from the specified map of raw messages.
func UnmarshalTagResultsItem(m map[string]json.RawMessage, result any) (err error) {
	obj := new(TagResultsItem)
	err = core.UnmarshalPrimitive(m, "resource_id", &obj.ResourceID)
	if err != nil {
		return
	}
	err = core.UnmarshalPrimitive(m, "is_error", &obj.IsError)
	if err != nil {
		return
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(obj))
	return
}

// UnmarshalAttachTagsResponse unmarshals an instance of VPC from the specified map of raw messages.
func UnmarshalAttachTagsResponse(m map[string]json.RawMessage, result any) (err error) {
	obj := new(AttachTagsResponse)
	err = core.UnmarshalModel(m, "results", &obj.Results, UnmarshalTagResultsItem)
	if err != nil {
		return
	}
	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(obj))
	return
}

// ListTags : List all Tags
// This request lists all Tags in the region.
func (service *GlobalTagging) AttachTags(attachTagsOptions *AttachTagsOptions, payload *AttachTagsBody) (result *AttachTagsResponse, response *core.DetailedResponse, err error) {
	return service.AttachTagsContext(context.Background(), attachTagsOptions, payload)
}

// AttachTagsContext is an alternate form of the AttachTags method which supports a Context parameter
func (service *GlobalTagging) AttachTagsContext(ctx context.Context, attachTagsOptions *AttachTagsOptions, payload *AttachTagsBody) (result *AttachTagsResponse, response *core.DetailedResponse, err error) {
	err = core.ValidateStruct(attachTagsOptions, "AttachTags")
	if err != nil {
		return
	}

	builder := core.NewRequestBuilder(core.POST)
	builder = builder.WithContext(ctx)
	builder.EnableGzipCompression = service.GetEnableGzipCompression()
	_, err = builder.ResolveRequestURL(service.Service.Options.URL, `/tags/attach/`, nil)
	if err != nil {
		return
	}

	for headerName, headerValue := range attachTagsOptions.Headers {
		builder.AddHeader(headerName, headerValue)
	}

	sdkHeaders := common.GetSdkHeaders("vpc", "V1", "AttachTags")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	builder.AddHeader("Accept", "application/json")
	builder.AddHeader("Content-Type", "application/json")

	builder.AddQuery("version", fmt.Sprint(*service.Version))
	builder.AddQuery("generation", fmt.Sprint(*service.generation))
	if attachTagsOptions.TagType != nil {
		builder.AddQuery("tag_type", fmt.Sprint(*attachTagsOptions.TagType))
	}

	_, err = builder.SetBodyContentJSON(payload)
	if err != nil {
		return
	}

	request, err := builder.Build()
	if err != nil {
		return
	}

	var rawResponse map[string]json.RawMessage
	response, err = service.Service.Request(request, &rawResponse)
	if err != nil {
		return
	}
	if rawResponse != nil {
		err = core.UnmarshalModel(rawResponse, "", &result, UnmarshalAttachTagsResponse)
		if err != nil {
			return
		}
		response.Result = result
	}

	return
}
