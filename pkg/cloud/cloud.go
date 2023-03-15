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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/fsx"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"sigs.k8s.io/aws-file-cache-csi-driver/pkg/util"
)

// DRA configuration parameters
const (
	DRAOptionsDataRepositoryPath           = "DataRepositoryPath"
	DRAOptionsDataRepositorySubdirectories = "DataRepositorySubdirectories"
	DRAOptionsFileCachePath                = "FileCachePath"
	DRAOptionsNFSConfiguration             = "NFS"
	NFSConfigurationOptionsDnsIps          = "DnsIps"
	NFSConfigurationOptionsVersion         = "Version"
)

// Lustre configuration parameters
const (
	LustreConfigOptionsDeploymentType           = "DeploymentType"
	LustreConfigOptionsMetaDataConfiguration    = "MetadataConfiguration"
	LustreConfigOptionsPerUnitStorageThroughput = "PerUnitStorageThroughput"
)

// Default values
const (
	// DefaultVolumeSize represents the default size used
	// this is the minimum Amazon File Cache FC size
	DefaultVolumeSize = 1200

	DefaultFileSystemType          = "LUSTRE"
	DefaultFileSystemTypeVersion   = "2.12"
	DefaultFileCacheDeploymentType = "CACHE_1"

	DefaultMetadataStorageCapacity  = 2400
	DefaultPerUnitStorageThroughput = 1000

	// PollCheckInterval specifies the interval to check if file cache is ready;
	// needs to be shorter than the provisioner timeout
	PollCheckInterval = 30 * time.Second
	// PollCheckTimeout specifies the time limit for polling DescribeFileCaches
	// for a completed create/update operation.
	PollCheckTimeout = 15 * time.Minute
)

// Tags
const (
	// VolumeNameTagKey is the key value that refers to the volume's name.
	VolumeNameTagKey = "CSIVolumeName"
)

var (
	// ErrMultiFileCaches is an error that is returned when multiple
	// file caches are found with the same volume name.
	ErrMultiFileCaches = errors.New("Multiple filecaches with same ID")

	// ErrFcExistsDiffSize is an error that is returned if a file cache
	// exists with a given ID, but a different capacity is requested.
	ErrFcExistsDiffSize = errors.New("There is already a disk with same ID and different size")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("Resource was not found")
)

// FileCache this is mainly for ValidateVolumeCapabilities
type FileCache struct {
	FileCacheId              string
	CapacityGiB              int64
	DnsName                  string
	MountName                string
	FileCacheType            string
	FileCacheTypeVersion     string
	PerUnitStorageThroughput int64
}

type FileCacheOptions struct {
	CapacityGiB                          int64
	SubnetId                             string
	SecurityGroupIds                     []string
	DataRepositoryAssociations           string
	FileCacheType                        string
	FileCacheTypeVersion                 string
	KmsKeyId                             string
	CopyTagsToDataRepositoryAssociations bool
	LustreConfiguration                  []string
	WeeklyMaintenanceStartTime           string
	ExtraTags                            []string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/fsx/
type FSx interface {
	CreateFileCacheWithContext(aws.Context, *fsx.CreateFileCacheInput, ...request.Option) (*fsx.CreateFileCacheOutput, error)
	DeleteFileCacheWithContext(aws.Context, *fsx.DeleteFileCacheInput, ...request.Option) (*fsx.DeleteFileCacheOutput, error)
	DescribeFileCachesWithContext(aws.Context, *fsx.DescribeFileCachesInput, ...request.Option) (*fsx.DescribeFileCachesOutput, error)
}

type Cloud interface {
	CreateFileCache(ctx context.Context, volumeName string, FileCacheOptions *FileCacheOptions) (fs *FileCache, err error)
	DeleteFileCache(ctx context.Context, FileCacheId string) (err error)
	DescribeFileCache(ctx context.Context, FileCacheId string) (fs *FileCache, err error)
	WaitForFileCacheAvailable(ctx context.Context, FileCacheId string) error
}

type cloud struct {
	fsx FSx
}

// NewCloud returns a new instance of AWS cloud
// It panics if session is invalid
func NewCloud(region string) Cloud {
	awsConfig := &aws.Config{
		Region:                        aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	return &cloud{
		fsx: fsx.New(session.Must(session.NewSession(awsConfig))),
	}
}

func (c *cloud) CreateFileCache(ctx context.Context, volumeName string, fileCacheOptions *FileCacheOptions) (cache *FileCache, err error) {
	if len(fileCacheOptions.SubnetId) == 0 {
		return nil, fmt.Errorf("SubnetId is required")
	}

	if fileCacheOptions.DataRepositoryAssociations == "" {
		return nil, fmt.Errorf("At least one DRA is required")
	}

	var dataRepositoryAssociations []*fsx.FileCacheDataRepositoryAssociation

	dras := strings.Fields(fileCacheOptions.DataRepositoryAssociations)
	for _, slice := range dras {
		draConfiguration := &fsx.FileCacheDataRepositoryAssociation{}

		configSlices := util.SplitUnnestedCommas(slice)
		configMap := util.MapValues(configSlices)

		if dataRepositoryPath, ok := configMap[DRAOptionsDataRepositoryPath]; ok {
			draConfiguration.SetDataRepositoryPath(dataRepositoryPath)
		}

		if dataRepositorySubdirectories, ok := configMap[DRAOptionsDataRepositorySubdirectories]; ok {
			subdirectories := strings.Split(dataRepositorySubdirectories[1:len(dataRepositorySubdirectories)-1], ",")
			draConfiguration.SetDataRepositorySubdirectories(aws.StringSlice(subdirectories))
		}

		if fileCachePath, ok := configMap[DRAOptionsFileCachePath]; ok {
			draConfiguration.SetFileCachePath(fileCachePath)
		}

		if nfsConfig, ok := configMap[DRAOptionsNFSConfiguration]; ok {
			nfsSlices := util.SplitUnnestedCommas(nfsConfig[1 : len(nfsConfig)-1])
			nfsMap := util.MapValues(nfsSlices)

			nfsConfiguration := &fsx.FileCacheNFSConfiguration{}

			if nfsDnsIps, ok := nfsMap[NFSConfigurationOptionsDnsIps]; ok {
				DnsIps := strings.Split(nfsDnsIps[1:len(nfsDnsIps)-1], ",")
				nfsConfiguration.SetDnsIps(aws.StringSlice(DnsIps))
			}

			if nfsVersion, ok := nfsMap[NFSConfigurationOptionsVersion]; ok {
				nfsConfiguration.SetVersion(nfsVersion)
			}
			draConfiguration.SetNFS(nfsConfiguration)
		}
		dataRepositoryAssociations = append(dataRepositoryAssociations, draConfiguration)
	}

	var tags = []*fsx.Tag{
		{
			Key:   aws.String(VolumeNameTagKey),
			Value: aws.String(volumeName),
		},
	}

	for _, extraTag := range fileCacheOptions.ExtraTags {
		extraTagSplit := strings.Split(extraTag, "=")
		tagKey := extraTagSplit[0]
		tagValue := extraTagSplit[1]

		tags = append(tags, &fsx.Tag{
			Key:   aws.String(tagKey),
			Value: aws.String(tagValue),
		})
	}

	fcType := DefaultFileSystemType
	if fileCacheOptions.FileCacheType != "" {
		fcType = fileCacheOptions.FileCacheType
	}
	fcTypeVersion := DefaultFileSystemTypeVersion
	if fileCacheOptions.FileCacheType != "" {
		fcTypeVersion = fileCacheOptions.FileCacheTypeVersion
	}

	input := &fsx.CreateFileCacheInput{
		ClientRequestToken:         aws.String(volumeName),
		DataRepositoryAssociations: dataRepositoryAssociations,
		FileCacheType:              aws.String(fcType),
		FileCacheTypeVersion:       aws.String(fcTypeVersion),
		StorageCapacity:            aws.Int64(fileCacheOptions.CapacityGiB),
		SubnetIds:                  []*string{aws.String(fileCacheOptions.SubnetId)},
		SecurityGroupIds:           aws.StringSlice(fileCacheOptions.SecurityGroupIds),
		Tags:                       tags,
	}

	lustreConfiguration := &fsx.CreateFileCacheLustreConfiguration{}
	//map for lustre configuration values
	configMap := util.MapValues(fileCacheOptions.LustreConfiguration)

	if deploymentType, ok := configMap[LustreConfigOptionsDeploymentType]; ok {
		lustreConfiguration.SetDeploymentType(deploymentType)
	} else {
		lustreConfiguration.SetDeploymentType(DefaultFileCacheDeploymentType)
	}

	if metadataConfiguration, ok := configMap[LustreConfigOptionsMetaDataConfiguration]; ok {
		storageCapacityPair := strings.Split(metadataConfiguration[1:len(metadataConfiguration)-1], "=")
		metadataStorageCapacity, err := strconv.ParseInt(storageCapacityPair[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid input for MetadataConfiguration StorageCapacity: %s", err)
		}
		metadataConfig := &fsx.FileCacheLustreMetadataConfiguration{StorageCapacity: aws.Int64(metadataStorageCapacity)}
		lustreConfiguration.SetMetadataConfiguration(metadataConfig)
	} else {
		metadataConfig := &fsx.FileCacheLustreMetadataConfiguration{StorageCapacity: aws.Int64(DefaultMetadataStorageCapacity)}
		lustreConfiguration.SetMetadataConfiguration(metadataConfig)
	}

	if perUnitStorageThroughput, ok := configMap[LustreConfigOptionsPerUnitStorageThroughput]; ok {
		throughput, err := strconv.ParseInt(perUnitStorageThroughput, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid input for MetadataConfiguration PerUnitStorageThroughput: %s", err)
		}
		lustreConfiguration.SetPerUnitStorageThroughput(throughput)
	} else {
		lustreConfiguration.SetPerUnitStorageThroughput(DefaultPerUnitStorageThroughput)
	}

	if fileCacheOptions.WeeklyMaintenanceStartTime != "" {
		lustreConfiguration.SetWeeklyMaintenanceStartTime(fileCacheOptions.WeeklyMaintenanceStartTime)
	}

	input.SetLustreConfiguration(lustreConfiguration)

	if fileCacheOptions.KmsKeyId != "" {
		input.SetKmsKeyId(fileCacheOptions.KmsKeyId)
	}

	if fileCacheOptions.CopyTagsToDataRepositoryAssociations {
		input.SetCopyTagsToDataRepositoryAssociations(true)
	}

	klog.V(4).Infof("CreateFileCacheInput: ", input.GoString())
	output, err := c.fsx.CreateFileCacheWithContext(ctx, input)
	if err != nil {
		if isIncompatibleParameter(err) {
			return nil, ErrFcExistsDiffSize
		}
		return nil, fmt.Errorf("CreateFileCache failed: %v", err)
	}

	if output.FileCache.LustreConfiguration.MountName == nil {
		return nil, fmt.Errorf("mountName returned in CreateFileCache is nil")
	}
	mountName := *output.FileCache.LustreConfiguration.MountName

	perUnitStorageThroughput := int64(0)
	if output.FileCache.LustreConfiguration.PerUnitStorageThroughput != nil {
		perUnitStorageThroughput = *output.FileCache.LustreConfiguration.PerUnitStorageThroughput
	}

	return &FileCache{
		FileCacheId:              *output.FileCache.FileCacheId,
		CapacityGiB:              *output.FileCache.StorageCapacity,
		DnsName:                  *output.FileCache.DNSName,
		MountName:                mountName,
		FileCacheType:            *output.FileCache.FileCacheType,
		FileCacheTypeVersion:     *output.FileCache.FileCacheTypeVersion,
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}, nil
}

func (c *cloud) DeleteFileCache(ctx context.Context, fileCacheId string) (err error) {
	input := &fsx.DeleteFileCacheInput{
		FileCacheId: aws.String(fileCacheId),
	}
	if _, err = c.fsx.DeleteFileCacheWithContext(ctx, input); err != nil {
		if isFileCacheNotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("DeleteFileCache failed: %v", err)
	}
	return nil
}

func (c *cloud) DescribeFileCache(ctx context.Context, fileCacheId string) (*FileCache, error) {
	fc, err := c.getFileCache(ctx, fileCacheId)
	if err != nil {
		return nil, err
	}

	if fc.LustreConfiguration.MountName == nil {
		return nil, fmt.Errorf("mountName returned in DescribeFileCache is nil")
	}
	mountName := *fc.LustreConfiguration.MountName

	perUnitStorageThroughput := int64(0)
	if fc.LustreConfiguration.PerUnitStorageThroughput != nil {
		perUnitStorageThroughput = *fc.LustreConfiguration.PerUnitStorageThroughput
	}

	return &FileCache{
		FileCacheId:              *fc.FileCacheId,
		CapacityGiB:              *fc.StorageCapacity,
		DnsName:                  *fc.DNSName,
		MountName:                mountName,
		FileCacheType:            *fc.FileCacheType,
		FileCacheTypeVersion:     *fc.FileCacheTypeVersion,
		PerUnitStorageThroughput: perUnitStorageThroughput,
	}, nil
}

func (c *cloud) WaitForFileCacheAvailable(ctx context.Context, fileCacheId string) error {
	err := wait.Poll(PollCheckInterval, PollCheckTimeout, func() (done bool, err error) {
		fc, err := c.getFileCache(ctx, fileCacheId)
		if err != nil {
			return true, err
		}
		klog.V(2).Infof("WaitForFileCacheAvailable file cache %q status is: %q", fileCacheId, *fc.Lifecycle)
		switch *fc.Lifecycle {
		case "AVAILABLE":
			return true, nil
		case "CREATING":
			return false, nil
		default:
			return true, fmt.Errorf("Unexpected state for file cache %s: %q", fileCacheId, *fc.Lifecycle)
		}
	})

	return err
}

func (c *cloud) getFileCache(ctx context.Context, fileCacheId string) (*fsx.FileCache, error) {
	input := &fsx.DescribeFileCachesInput{
		FileCacheIds: []*string{aws.String(fileCacheId)},
	}
	output, err := c.fsx.DescribeFileCachesWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(output.FileCaches) == 0 {
		return nil, ErrNotFound
	}

	if len(output.FileCaches) > 1 {
		return nil, ErrMultiFileCaches
	}

	return output.FileCaches[0], nil
}

func isFileCacheNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeFileCacheNotFound {
			return true
		}
	}
	return false
}

func isIncompatibleParameter(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == fsx.ErrCodeIncompatibleParameterError {
			return true
		}
	}
	return false
}
