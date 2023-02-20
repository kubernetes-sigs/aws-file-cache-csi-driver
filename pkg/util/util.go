package util

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	GiB = 1024 * 1024 * 1024
)

func RoundUpVolumeSize(volumeSizeBytes int64) int64 {
	if volumeSizeBytes < 2400*GiB {
		return roundUpSize(volumeSizeBytes, 1200*GiB) * 1200
	} else {
		return roundUpSize(volumeSizeBytes, 2400*GiB) * 2400
	}
}
func roundUpSize(volumeSizeBytes int64, allocationUnitBytes int64) int64 {
	return (volumeSizeBytes + allocationUnitBytes - 1) / allocationUnitBytes
}

func ParseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("could not parse endpoint: %v", err)
	}

	addr := path.Join(u.Host, filepath.FromSlash(u.Path))

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "tcp":
	case "unix":
		addr = path.Join("/", addr)
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return "", "", fmt.Errorf("could not remove unix domain socket %q: %v", addr, err)
		}
	default:
		return "", "", fmt.Errorf("unsupported protocol: %s", scheme)
	}

	return scheme, addr, nil
}

// GiBToBytes converts GiB to Bytes
func GiBToBytes(volumeSizeGiB int64) int64 {
	return volumeSizeGiB * GiB
}

func SplitUnnestedCommas(input string) []string {
	leftCurly := []int{}
	leftSquare := []int{}

	slices := []string{}
	lastComma := 0

	for i, c := range input {
		if "{" == string(c) {
			leftCurly = append(leftCurly, i)
		} else if "[" == string(c) {
			leftSquare = append(leftSquare, i)
		} else if "}" == string(c) {
			if len(leftCurly) > 0 {
				leftCurly = leftCurly[:len(leftCurly)-1]
			}
		} else if "]" == string(c) {
			if len(leftSquare) > 0 {
				leftSquare = leftSquare[:len(leftSquare)-1]
			}
		} else if "," == string(c) {
			if len(leftCurly) == 0 && len(leftSquare) == 0 {
				slices = append(slices, input[lastComma:i])
				lastComma = i + 1
			}
		}
	}

	slices = append(slices, input[lastComma:])
	return slices
}
