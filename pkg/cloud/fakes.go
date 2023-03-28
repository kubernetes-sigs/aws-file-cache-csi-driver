/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloud

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type FakeCloudProvider struct {
	m          *metadata
	fileCaches map[string]*FileCache
}

func NewFakeCloudProvider() *FakeCloudProvider {
	return &FakeCloudProvider{
		m:          &metadata{"instanceID", "region", "az"},
		fileCaches: make(map[string]*FileCache),
	}
}

func (c *FakeCloudProvider) GetMetadata() MetadataService {
	return c.m
}

func (c *FakeCloudProvider) CreateFileCache(ctx context.Context, volumeName string, fileCacheOptions *FileCacheOptions) (fc *FileCache, err error) {
	fc, exists := c.fileCaches[volumeName]
	if exists {
		if fc.CapacityGiB == fileCacheOptions.CapacityGiB {
			return fc, nil
		} else {
			return nil, ErrFcExistsDiffSize
		}
	}

	// Value for Testing Purposes
	perUnitStorageThroughput := int64(1000)

	fc = &FileCache{
		FileCacheId:              fmt.Sprintf("fc-%d", random.Uint64()),
		CapacityGiB:              fileCacheOptions.CapacityGiB,
		DnsName:                  "test.us-east-1.fsx.amazonaws.com",
		MountName:                "random",
		FileCacheType:            fileCacheOptions.FileCacheType,
		FileCacheTypeVersion:     fileCacheOptions.FileCacheTypeVersion,
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}

	c.fileCaches[volumeName] = fc
	return fc, nil
}

func (c *FakeCloudProvider) DeleteFileCache(ctx context.Context, volumeID string) (err error) {
	delete(c.fileCaches, volumeID)
	for name, fc := range c.fileCaches {
		if fc.FileCacheId == volumeID {
			delete(c.fileCaches, name)
		}
	}
	return nil
}

func (c *FakeCloudProvider) DescribeFileCache(ctx context.Context, volumeID string) (fc *FileCache, err error) {
	for _, fc := range c.fileCaches {
		if fc.FileCacheId == volumeID {
			return fc, nil
		}
	}
	return nil, ErrNotFound
}

func (c *FakeCloudProvider) WaitForFileCacheAvailable(ctx context.Context, fileCacheId string) error {
	return nil
}
