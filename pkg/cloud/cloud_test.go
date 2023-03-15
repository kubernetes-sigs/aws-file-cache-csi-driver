/*
Copyright 2022 The Kubernetes Authors

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
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/golang/mock/gomock"
	"sigs.k8s.io/aws-file-cache-csi-driver/pkg/cloud/mocks"
	"testing"
)

func TestCreateFileCache(t *testing.T) {
	var (
		volumeName                                 = "volumeName"
		mountname                                  = "mountName"
		fileCacheId                                = "fc-1234"
		volumeSizeGiB                        int64 = 1200
		subnetId                                   = "subnet-056da83524edbe641"
		securityGroupIds                           = []string{"sg-086f61ea73388fb6b", "sg-0145e55e976000c9e"}
		dnsname                                    = "test.us-east-1.fsx.amazonaws.com"
		dataRepositoryAssociations                 = "FileCachePath=/ns1/,DataRepositoryPath=nfs://10.0.92.69/,NFS={Version=NFS3},DataRepositorySubdirectories=[subdir1,subdir2,subdir3]"
		fileCacheType                              = "LUSTRE"
		fileCacheTypeVersion                       = "2.12"
		weeklyMaintenanceStartTime                 = "7:00:00"
		LustreConfiguration                        = []string{"DeploymentType=CACHE_1", "PerUnitStorageThroughput=1000", "MetadataConfiguration={StorageCapacity=2400}"}
		copyTagsToDataRepositoryAssociations       = true
		kmsKeyId                                   = "arn:aws:kms:us-east-1:215474938041:key/48313a27-7d88-4b51-98a4-fdf5bc80dbbe"
		extraTags                                  = []string{"key1=value1", "key2=value2"}
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			//TODO: Expand test checks after adding more values in FileCache object
			name: "success: normal create with working values",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				req := &FileCacheOptions{
					CapacityGiB:                          volumeSizeGiB,
					SubnetId:                             subnetId,
					SecurityGroupIds:                     securityGroupIds,
					DataRepositoryAssociations:           dataRepositoryAssociations,
					FileCacheType:                        fileCacheType,
					FileCacheTypeVersion:                 fileCacheTypeVersion,
					WeeklyMaintenanceStartTime:           weeklyMaintenanceStartTime,
					LustreConfiguration:                  LustreConfiguration,
					CopyTagsToDataRepositoryAssociations: copyTagsToDataRepositoryAssociations,
					KmsKeyId:                             kmsKeyId,
					ExtraTags:                            extraTags,
				}

				output := &fsx.CreateFileCacheOutput{
					FileCache: &fsx.FileCacheCreating{
						FileCacheId:          aws.String(fileCacheId),
						FileCacheType:        aws.String(fileCacheType),
						FileCacheTypeVersion: aws.String(fileCacheTypeVersion),
						StorageCapacity:      aws.Int64(volumeSizeGiB),
						DNSName:              aws.String(dnsname),
						LustreConfiguration: &fsx.FileCacheLustreConfiguration{
							MountName: aws.String(mountname),
						},
					},
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileCacheWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				resp, err := c.CreateFileCache(ctx, volumeName, req)

				if err != nil {
					t.Fatalf("CreateFileCache failed: %v", err)
				}

				if resp.FileCacheId != fileCacheId {
					t.Fatalf("FileCacheId mismatches. actual: %v expected: %v", resp.FileCacheId, fileCacheId)
				}

				if resp.CapacityGiB != volumeSizeGiB {
					t.Fatalf("CapacityGiB mismatches. actual: %v expected: %v", resp.CapacityGiB, volumeSizeGiB)
				}

				if resp.DnsName != dnsname {
					t.Fatalf("DnsName mismatches. actual: %v expected: %v", resp.DnsName, dnsname)
				}

				if resp.FileCacheType != fileCacheType {
					t.Fatalf("FileCacheType mismatches. actual: %v expected %v", resp.FileCacheType, fileCacheType)
				}

				if resp.FileCacheTypeVersion != fileCacheTypeVersion {
					t.Fatalf("FileCacheTypeVersion mismatches. actual %v expected %v", resp.FileCacheTypeVersion, fileCacheTypeVersion)
				}
				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing subnet ID",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				req := &FileCacheOptions{
					CapacityGiB:                volumeSizeGiB,
					SecurityGroupIds:           securityGroupIds,
					DataRepositoryAssociations: dataRepositoryAssociations,
				}

				ctx := context.Background()
				_, err := c.CreateFileCache(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileCache is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: missing DRA",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				req := &FileCacheOptions{
					CapacityGiB:      volumeSizeGiB,
					SubnetId:         subnetId,
					SecurityGroupIds: securityGroupIds,
				}

				ctx := context.Background()
				_, err := c.CreateFileCache(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileCache is not failed")
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: CreateFileCacheWithContext return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				req := &FileCacheOptions{
					CapacityGiB:                volumeSizeGiB,
					SubnetId:                   subnetId,
					SecurityGroupIds:           securityGroupIds,
					DataRepositoryAssociations: dataRepositoryAssociations,
				}

				ctx := context.Background()
				mockFSx.EXPECT().CreateFileCacheWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("CreateFileCacheWithContext failed"))
				_, err := c.CreateFileCache(ctx, volumeName, req)
				if err == nil {
					t.Fatal("CreateFileCache is not failed")
				}

				mockCtl.Finish()
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDeleteFileCache(t *testing.T) {
	var (
		fileCacheId = "fc-1234"
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				output := &fsx.DeleteFileCacheOutput{}
				ctx := context.Background()
				mockFSx.EXPECT().DeleteFileCacheWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				err := c.DeleteFileCache(ctx, fileCacheId)
				if err != nil {
					t.Fatalf("DeleteFileCache is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: DeleteFileCacheWithContext return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				mockFSx.EXPECT().DeleteFileCacheWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DeleteFileCacheWithContext failed"))
				err := c.DeleteFileCache(ctx, fileCacheId)
				if err == nil {
					t.Fatal("DeleteFileCache is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestDescribeFileCache(t *testing.T) {
	var (
		fileCacheId                = "fc-1234"
		FileCacheType              = "LUSTRE"
		FileCacheTypeVersion       = "2.12"
		volumeSizeGiB        int64 = 1200
		dnsname                    = "test.us-east-1.fsx.amazonaws.com"
		mountname                  = "mountName"
	)
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				output := &fsx.DescribeFileCachesOutput{
					FileCaches: []*fsx.FileCache{
						{
							FileCacheId:          aws.String(fileCacheId),
							FileCacheType:        aws.String(FileCacheType),
							FileCacheTypeVersion: aws.String(FileCacheTypeVersion),
							StorageCapacity:      aws.Int64(volumeSizeGiB),
							DNSName:              aws.String(dnsname),
							LustreConfiguration: &fsx.FileCacheLustreConfiguration{
								MountName: aws.String(mountname),
							},
						},
					},
				}
				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileCachesWithContext(gomock.Eq(ctx), gomock.Any()).Return(output, nil)
				_, err := c.DescribeFileCache(ctx, fileCacheId)
				if err != nil {
					t.Fatalf("DescribeFileCache is failed: %v", err)
				}

				mockCtl.Finish()
			},
		},
		{
			name: "fail: DescribeFileCacheWithContext return error",
			testFunc: func(t *testing.T) {
				mockCtl := gomock.NewController(t)
				mockFSx := mocks.NewMockFSx(mockCtl)
				c := &cloud{
					fsx: mockFSx,
				}

				ctx := context.Background()
				mockFSx.EXPECT().DescribeFileCachesWithContext(gomock.Eq(ctx), gomock.Any()).Return(nil, errors.New("DescribeFileCachesWithContext failed"))
				_, err := c.DescribeFileCache(ctx, fileCacheId)
				if err == nil {
					t.Fatal("DescribeFileCache is not failed")
				}

				mockCtl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
