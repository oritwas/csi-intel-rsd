// Copyright 2019 Intel Corporation. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rsd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Transport is an interface to communicate with RSD server
type Transport interface {
	Get(entrypoint string, result interface{}) error
	Post(entrypoint string, data map[string]string, result interface{}) (*http.Header, error)
	Delete(entrypoint string, data map[string]string, result interface{}) (*http.Header, error)
}

// Client is a struct that interfaces with the RSD Redfish API
type Client struct {
	baseurl    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClient creates new RSD Client
func NewClient(baseurl, username, password string, timeout time.Duration) (*Client, error) {
	return &Client{
		baseurl:    baseurl,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// request queries sends HTTP request to the RSD endpoint and decodes HTTP response
func (rsd *Client) request(entrypoint, method string, body io.Reader, result interface{}) (*http.Header, error) {
	url := rsd.baseurl + entrypoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't make request from %s", url)
	}

	if rsd.username != "" {
		req.SetBasicAuth(rsd.username, rsd.password)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := rsd.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't get http response from %s", url)
	}

	defer resp.Body.Close()

	// Decode response if needed
	if result != nil {
		err = json.NewDecoder(resp.Body).Decode(result)
		if err != nil {
			return &resp.Header, errors.Wrapf(err, "Can't decode http response from %s", url)
		}
	}

	return &resp.Header, nil
}

// Get sends GET RSD endpoint and returns decoded http response
func (rsd *Client) Get(entrypoint string, result interface{}) error {
	_, err := rsd.request(entrypoint, "GET", nil, result)
	return err
}

// Post sends POST request to RSD endpoint and returns decoded http response
func (rsd *Client) Post(entrypoint string, data map[string]string, result interface{}) (*http.Header, error) {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't marshal data: %v", data)
	}
	return rsd.request(entrypoint, "POST", bytes.NewReader(marshalled), result)
}

// Delete sends DELETE request to RSD endpoint
func (rsd *Client) Delete(entrypoint string, data map[string]string, result interface{}) (*http.Header, error) {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "Can't marshal data: %v", data)
	}
	return rsd.request(entrypoint, "DELETE", bytes.NewReader(marshalled), result)
}

// GetStorageServiceCollection returns StorageServiceCollection
func GetStorageServiceCollection(rsd Transport) (*StorageServiceCollection, error) {
	var result StorageServiceCollection
	err := rsd.Get(StorageServiceCollectionEntryPoint, &result)
	if err != nil {
		return nil, errors.Wrap(err, "Can't query StorageServiceCollection")
	}

	return &result, err
}

// GetVolumeCollection returns VolumeCollection for the storage service <ssNum>
func GetVolumeCollection(rsd Transport, ssNum int) (*VolumeCollection, error) {
	ssCollection, err := GetStorageServiceCollection(rsd)
	if err != nil {
		return nil, errors.Wrap(err, "Can't get storage service collection")
	}

	services, err := ssCollection.GetMembers(rsd)
	if err != nil {
		return nil, errors.Wrap(err, "Can't get storage service collection members")
	}

	if len(services) == 0 {
		return nil, errors.New("No storage services found in a collection")
	}

	return services[ssNum].GetVolumeCollection(rsd)
}