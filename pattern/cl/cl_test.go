package cl_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	cl "github.com/xypwn/gocl/cl-3.1"
	"github.com/xypwn/hd2-hash-cracker/hash"
	"github.com/xypwn/hd2-hash-cracker/pattern"
	pcl "github.com/xypwn/hd2-hash-cracker/pattern/cl"
)

func TestCl(t *testing.T) {
	require := require.New(t)
	prog, err := pattern.Compile("", []byte("<a|b|c|d|e|f|g|h|i|j>{1,6}"), pattern.CompileOptions{})
	require.NoError(err)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	platforms, err := cl.GetPlatformIDs()
	require.NoError(err)
	if len(platforms) == 0 {
		require.Fail("no OpenCL platforms")
	}
	platform := platforms[0]
	devices, err := cl.GetDeviceIDs(platform, cl.DEVICE_TYPE_GPU)
	require.NoError(err)
	if len(devices) == 0 {
		require.Fail("no OpenCL devices")
	}
	device := devices[0]

	targetHashes := []uint64{
		hash.Murmur64aSum("aba"),
		hash.Murmur64aSum("aaaa"),
		hash.Murmur64aSum("aaaaa"),
		hash.Murmur64aSum("aaaaaa"),
		hash.Murmur64aSum("abcdef"),
	}

	var allMatches []string
	cr, err := pcl.NewCracker(device, prog, targetHashes, pcl.Options{})
	require.NoError(err)
	defer cr.Delete()
	for {
		//fmt.Println(cr.TotalIdx(), "/", prog.Comp)
		matches, err := cr.Dispatch()
		if err == pcl.Done {
			break
		}
		require.NoError(err)
		allMatches = append(allMatches, matches...)
	}
	require.Equal([]string{"aba", "aaaa", "aaaaa", "aaaaaa", "abcdef"}, allMatches)
	//fmt.Println(allMatches)
}
