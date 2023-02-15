package driver

import (
	"context"
	"fmt"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"sigs.k8s.io/aws-file-cache-csi-driver/pkg/driver/mocks"
)

func TestNodePublishVolume(t *testing.T) {

	var (
		endpoint   = "endpoint"
		nodeID     = "nodeID"
		dnsname    = "fc-0a2d0632b5ff567e9.fsx.us-west-2.amazonaws.com"
		mountname  = "random"
		targetPath = "/target/path"
		stdVolCap  = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			},
		}
	)

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}
				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "success: missing mountname for static provisioning, default 'fsx' used",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}
				source := dnsname + "@tcp:/fsx"

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname": dnsname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "success: normal with read only mount",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
					Readonly:         true,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Eq([]string{"ro"})).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "success: normal with flock mount options",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				source := dnsname + "@tcp:/" + mountname

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: &csi.VolumeCapability{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{
								MountFlags: []string{"flock"},
							},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Eq([]string{"flock"})).Return(nil)
				_, err := driver.NodePublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodePublishVolume is failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "fail: missing dns name",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"mountname": mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "fail: missing target path",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: stdVolCap,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "fail: missing volume capability",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					TargetPath: targetPath,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "fail: unsupported volume capability",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: &csi.VolumeCapability{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
						},
					},
					TargetPath: targetPath,
				}

				_, err := driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "fail: mounter failed to MakeDir",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				err := fmt.Errorf("failed to MakeDir")
				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(err)

				_, err = driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
		{
			name: "fail: mounter failed to Mount",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodePublishVolumeRequest{
					VolumeId: "volumeId",
					VolumeContext: map[string]string{
						"dnsname":   dnsname,
						"mountname": mountname,
					},
					VolumeCapability: stdVolCap,
					TargetPath:       targetPath,
				}

				source := dnsname + "@tcp:/" + mountname
				err := fmt.Errorf("failed to Mount")
				mockMounter.EXPECT().MakeDir(gomock.Eq(targetPath)).Return(nil)
				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)
				mockMounter.EXPECT().Mount(gomock.Eq(source), gomock.Eq(targetPath), gomock.Eq("lustre"), gomock.Any()).Return(err)

				_, err = driver.NodePublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodePublishVolume is not failed: %v", err)
				}

				mockCtrl.Finish()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestNodeUnpublishVolume(t *testing.T) {

	var (
		endpoint   = "endpoint"
		nodeID     = "nodeID"
		targetPath = "/target/path"
	)

	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "success: normal",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).Return(nil)

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodeUnpublishVolume is failed: %v", err)
				}
			},
		},
		{
			name: "success: target already unmounted",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(true, nil)

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err != nil {
					t.Fatalf("NodeUnpublishVolume is failed: %v", err)
				}
			},
		},
		{
			name: "fail: targetPath is missing",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId: "volumeId",
				}

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodeUnpublishVolume is not failed: %v", err)
				}
			},
		},
		{
			name: "fail: mounter failed to umount",
			testFunc: func(t *testing.T) {
				mockCtrl := gomock.NewController(t)
				mockMounter := mocks.NewMockMounter(mockCtrl)
				driver := &Driver{
					endpoint: endpoint,
					nodeID:   nodeID,
					mounter:  mockMounter,
				}

				ctx := context.Background()
				req := &csi.NodeUnpublishVolumeRequest{
					VolumeId:   "volumeId",
					TargetPath: targetPath,
				}

				mockMounter.EXPECT().IsLikelyNotMountPoint(gomock.Eq(targetPath)).Return(false, nil)
				mountErr := fmt.Errorf("Unmount failed")
				mockMounter.EXPECT().Unmount(gomock.Eq(targetPath)).Return(mountErr)

				_, err := driver.NodeUnpublishVolume(ctx, req)
				if err == nil {
					t.Fatalf("NodeUnpublishVolume is not failed: %v", err)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}
