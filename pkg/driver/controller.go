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

package driver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-file-cache-csi-driver/pkg/cloud"
	"sigs.k8s.io/aws-file-cache-csi-driver/pkg/util"
	"strconv"
	"strings"
)

var (
	// controllerCaps represents the capability of controller service
	controllerCaps = []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
	}
)

// TODO: make sure these values are consistent with the docs & templates
const (
	volumeContextDnsName                             = "dnsname"
	volumeContextMountName                           = "mountname"
	volumeParamsSubnetId                             = "subnetId"
	volumeParamsSecurityGroupIds                     = "securityGroupIds"
	volumeParamsDataRepositoryAssociations           = "dataRepositoryAssociations"
	volumeParamsFileCacheType                        = "fileCacheType"
	volumeParamsFileCacheTypeVersion                 = "fileCacheTypeVersion"
	volumeParamsKmsKeyId                             = "kmsKeyId"
	volumeParamsCopyTagsToDataRepositoryAssociations = "copyTagsToDataRepositoryAssociations"
	volumeParamsLustreConfiguration                  = "LustreConfiguration"
	volumeParamsWeeklyMaintenanceStartTime           = "weeklyMaintenanceStartTime"
	volumeParamsExtraTags                            = "extraTags"
)

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.V(4).Infof("CreateVolume: called with args %#v", req)
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume name not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if !d.isValidVolumeCapabilities(volCaps) {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not supported")
	}

	// create a new volume with idempotency, which is handled by `CreateFileSystem`
	volumeParams := req.GetParameters()
	subnetId := volumeParams[volumeParamsSubnetId]
	securityGroupIds := volumeParams[volumeParamsSecurityGroupIds]

	fcOptions := &cloud.FileCacheOptions{
		SubnetId:         subnetId,
		SecurityGroupIds: strings.Split(securityGroupIds, ","),
	}

	if val, ok := volumeParams[volumeParamsDataRepositoryAssociations]; ok {
		fcOptions.DataRepositoryAssociations = val
	}

	if val, ok := volumeParams[volumeParamsFileCacheType]; ok {
		fcOptions.FileCacheType = val
	}

	if val, ok := volumeParams[volumeParamsFileCacheTypeVersion]; ok {
		fcOptions.FileCacheTypeVersion = val
	}

	if val, ok := volumeParams[volumeParamsKmsKeyId]; ok {
		fcOptions.KmsKeyId = val
	}

	if val, ok := volumeParams[volumeParamsCopyTagsToDataRepositoryAssociations]; ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Invalid copyTagsToDataRepositoryAssociations value")
		}
		fcOptions.CopyTagsToDataRepositoryAssociations = boolVal
	}

	if val, ok := volumeParams[volumeParamsLustreConfiguration]; ok {
		lustreConfigs := strings.Split(val, ",")
		fcOptions.LustreConfiguration = lustreConfigs
	}

	if val, ok := volumeParams[volumeParamsWeeklyMaintenanceStartTime]; ok {
		fcOptions.WeeklyMaintenanceStartTime = val
	}

	capRange := req.GetCapacityRange()
	if capRange == nil {
		fcOptions.CapacityGiB = cloud.DefaultVolumeSize
	} else {
		fcOptions.CapacityGiB = util.RoundUpVolumeSize(capRange.GetRequiredBytes())
	}

	if val, ok := volumeParams[volumeParamsExtraTags]; ok {
		extraTags := strings.Split(val, ",")
		fcOptions.ExtraTags = extraTags
	}

	fc, err := d.cloud.CreateFileCache(ctx, volName, fcOptions)
	if err != nil {
		klog.V(4).Infof("CreateFileCache error: ", err.Error())
		switch err {
		case cloud.ErrFcExistsDiffSize:
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "Could not create volume %q: %v")
		}
	}

	err = d.cloud.WaitForFileCacheAvailable(ctx, fc.FileCacheId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "FileCache is not ready: %v", err)
	}

	return newCreateVolumeResponse(fc), nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(4).Infof("DeleteVolume: called with args: %#v", req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	if err := d.cloud.DeleteFileCache(ctx, volumeID); err != nil {
		if err == cloud.ErrNotFound {
			klog.V(4).Infof("DeleteVolume: volume not found, returning with success")
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "Could not delete volume ID %q: %v", volumeID, err)
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(4).Infof("ControllerGetCapabilities: called with args %#v", req)
	var caps []*csi.ControllerServiceCapability
	for _, cap := range controllerCaps {
		c := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.ControllerGetCapabilitiesResponse{Capabilities: caps}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	klog.V(4).Infof("GetCapacity: called with args %#v", req)
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	klog.V(4).Infof("ListVolumes: called with args %#v", req)
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	klog.V(4).Infof("ValidateVolumeCapabilities: called with args %#v", req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities not provided")
	}

	if _, err := d.cloud.DescribeFileCache(ctx, volumeID); err != nil {
		if err == cloud.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Volume not found")
		}
		return nil, status.Errorf(codes.Internal, "Could not get volume with ID %q: %v", volumeID, err)
	}

	confirmed := d.isValidVolumeCapabilities(volCaps)
	if confirmed {
		return &csi.ValidateVolumeCapabilitiesResponse{
			Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
				// TODO if volume context is provided, should validate it too
				//  VolumeContext:      req.GetVolumeContext(),
				VolumeCapabilities: volCaps,
				// TODO if parameters are provided, should validate them too
				//  Parameters:      req.GetParameters(),
			},
		}, nil
	} else {
		return &csi.ValidateVolumeCapabilitiesResponse{}, nil
	}
}

func (d *Driver) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, c := range volumeCaps {
			if c.GetMode() == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
		}
	}
	return foundAll
}

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func newCreateVolumeResponse(fc *cloud.FileCache) *csi.CreateVolumeResponse {
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      fc.FileCacheId,
			CapacityBytes: util.GiBToBytes(fc.CapacityGiB),
			VolumeContext: map[string]string{
				volumeContextDnsName:   fc.DnsName,
				volumeContextMountName: fc.MountName,
			},
		},
	}
}
