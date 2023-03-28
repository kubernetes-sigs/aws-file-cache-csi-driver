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

package util

import (
	"reflect"
	"testing"
)

func TestRoundUpVolumeSize(t *testing.T) {
	testCases := []struct {
		name        string
		sizeInBytes int64
		expected    int64
	}{
		{
			name:        "Roundup 1 byte",
			sizeInBytes: 1,
			expected:    1200,
		},
		{
			name:        "Roundup 1 Gib",
			sizeInBytes: 1 * GiB,
			expected:    1200,
		},
		{
			name:        "Roundup 1000 Gib",
			sizeInBytes: 1000 * GiB,
			expected:    1200,
		},
		{
			name:        "Roundup 2000 Gib",
			sizeInBytes: 2000 * GiB,
			expected:    2400,
		},
		{
			name:        "Roundup 2400 Gib",
			sizeInBytes: 2400 * GiB,
			expected:    2400,
		},
		{
			name:        "Roundup 2400 Gib + 1 Byte",
			sizeInBytes: 2400*GiB + 1,
			expected:    4800,
		},
		{
			name:        "Roundup 3600 Gib",
			sizeInBytes: 3600 * GiB,
			expected:    4800,
		},
		{
			name:        "Roundup 4800 Gib",
			sizeInBytes: 4800 * GiB,
			expected:    4800,
		},
		{
			name:        "Roundup 4800 Gib + 1 Byte",
			sizeInBytes: 4800*GiB + 1,
			expected:    7200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := RoundUpVolumeSize(tc.sizeInBytes)
			if actual != tc.expected {
				t.Fatalf("RoundUpVolumeSize got wrong result. actual: %d, expected: %d", actual, tc.expected)
			}
		})
	}
}

func TestGiBToBytes(t *testing.T) {
	var sizeInGiB int64 = 3

	actual := GiBToBytes(sizeInGiB)
	if actual != 3*GiB {
		t.Fatalf("Wrong result for GiBToBytes. Got: %d", actual)
	}
}

func TestSplitUnnestedCommas(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "dataRepositoryAssociations",
			input:    "FileCachePath=/ns1/,DataRepositoryPath=nfs://filler.domain.com/,NFS={Version=NFS3,DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]},DataRepositorySubdirectories=[subdir1,subdir2,subdir3]",
			expected: []string{"FileCachePath=/ns1/", "DataRepositoryPath=nfs://filler.domain.com/", "NFS={Version=NFS3,DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]}", "DataRepositorySubdirectories=[subdir1,subdir2,subdir3]"},
		},
		{
			name:     "NFS",
			input:    "Version=NFS3,DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]",
			expected: []string{"Version=NFS3", "DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := SplitUnnestedCommas(tc.input)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("SplitUnnestedCommas got wrong result. actual: %+v, expected: %+v", actual, tc.expected)
			}
		})
	}
}

func TestMapValues(t *testing.T) {

	testCases := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "LustreConfiguration",
			input:    []string{"DeploymentType=CACHE_1", "PerUnitStorageThroughput=1000", "MetadataConfiguration={StorageCapacity=2400}"},
			expected: map[string]string{"DeploymentType": "CACHE_1", "PerUnitStorageThroughput": "1000", "MetadataConfiguration": "{StorageCapacity=2400}"},
		},
		{
			name:     "dataRepositoryAssociations",
			input:    []string{"FileCachePath=/ns1/", "DataRepositoryPath=nfs://filler.domain.com/", "NFS={Version=NFS3,DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]}", "DataRepositorySubdirectories=[subdir1,subdir2,subdir3]"},
			expected: map[string]string{"FileCachePath": "/ns1/", "DataRepositoryPath": "nfs://filler.domain.com/", "NFS": "{Version=NFS3,DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]}", "DataRepositorySubdirectories": "[subdir1,subdir2,subdir3]"},
		},
		{
			name:     "NFS",
			input:    []string{"Version=NFS3", "DnsIps=[10.0.0.1,10.0.0.2,10.0.0.3]"},
			expected: map[string]string{"Version": "NFS3", "DnsIps": "[10.0.0.1,10.0.0.2,10.0.0.3]"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := MapValues(tc.input)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Fatalf("MapValues got wrong result. actual: %+v, expected: %+v", actual, tc.expected)
			}
		})
	}
}
