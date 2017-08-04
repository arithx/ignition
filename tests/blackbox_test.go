// Copyright 2017 CoreOS, Inc.
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

package blackbox

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"

	// Register the tests
	_ "github.com/coreos/ignition/tests/registry"
)

func (server *HTTPServer) Config(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{
	"ignition": { "version": "2.0.0" },
	"storage": {
		"files": [{
		  "filesystem": "root",
		  "path": "/foo/bar",
		  "contents": { "source": "data:,example%20file%0A" }
		}]
	}
}`))
}

func (server *HTTPServer) Contents(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`asdf
fdsa`))
}

type HTTPServer struct{}

func (server *HTTPServer) Start() {
	http.HandleFunc("/contents", server.Contents)
	http.HandleFunc("/config", server.Config)

	s := &http.Server{Addr: ":8080"}
	go s.ListenAndServe()
}

func TestMain(m *testing.M) {
	server := &HTTPServer{}
	server.Start()
	os.Exit(m.Run())
}

func TestIgnitionBlackBox(t *testing.T) {
	for _, test := range register.Tests[register.PositiveTest] {
		t.Run(test.Name, func(t *testing.T) {
			outer(t, test, false)
		})
	}
}

func TestIgnitionBlackBoxNegative(t *testing.T) {
	for _, test := range register.Tests[register.NegativeTest] {
		t.Run(test.Name, func(t *testing.T) {
			outer(t, test, true)
		})
	}
}

func PreCleanup(t *testing.T) {
	mountpoints, _ := exec.Command(
		"findmnt", "-l", "-o", "target").CombinedOutput()
	points := strings.Split(string(mountpoints), "\n")
	for i := len(points) - 1; i >= 0; i-- {
		tmpDir := os.Getenv("TMPDIR")
		if tmpDir == "" {
			tmpDir = "/tmp"
		}
		match, err := filepath.Match(fmt.Sprintf("%s/*", tmpDir), points[i])
		if err != nil {
			t.Log(err)
		}
		if match {
			_, _ = exec.Command("umount", points[i]).CombinedOutput()
		}
	}
}

func outer(t *testing.T, test types.Test, negativeTests bool) {
	PreCleanup(t)
	t.Log(test.Name)

	tmpDirectory, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create a temp dir: %v", err)
	}
	_ = os.Setenv("TMPDIR", tmpDirectory)

	path := os.Getenv("PATH")
	cwd, _ := os.Getwd()
	_ = os.Setenv("PATH", fmt.Sprintf(
		"%s:%s", filepath.Join(cwd, "bin/amd64"), path))

	var rootLocation string

	// Setup
	for i, disk := range test.In {
		// There may be more partitions created by Ignition, so look at the
		// expected output instead of the input to determine image size
		imageSize := calculateImageSize(test.Out[i].Partitions)

		// Finish data setup
		for _, part := range disk.Partitions {
			if part.GUID == "" {
				part.GUID = generateUUID(t)
			}
			updateTypeGUID(t, part)
		}
		setOffsets(disk.Partitions)
		for _, part := range test.Out[i].Partitions {
			updateTypeGUID(t, part)
		}
		setOffsets(test.Out[i].Partitions)

		// NOTE: the image file is written into cwd because sgdisk fails when
		// the file is located in /tmp/

		// Creation
		createVolume(t, disk.ImageFile, imageSize, 20, 16, 63, disk.Partitions)
		loopDevice := setDevices(t, disk.ImageFile, disk.Partitions)
		rootMounted := mountRootPartition(t, disk.Partitions)
		if rootMounted && strings.Contains(test.Config, "passwd") {
			prepareRootPartitionForPasswd(t, disk.Partitions)
		}
		mountPartitions(t, disk.Partitions)
		createFiles(t, disk.Partitions)
		unmountPartitions(t, disk.Partitions)

		// Mount device name substitution
		for _, d := range test.MntDevices {
			device := pickDevice(t, disk.Partitions, disk.ImageFile, d.Label)
			// The device may not be on this disk, if it's not found here let's
			// assume we'll find it on another one and keep going
			if device != "" {
				test.Config = strings.Replace(test.Config, d.Substitution, device, -1)
			}
		}

		// Replace any instance of $<image-file> with the actual loop device
		// that got assigned to it
		test.Config = strings.Replace(test.Config, "$"+disk.ImageFile, loopDevice, -1)

		if rootLocation == "" {
			rootLocation = getRootLocation(disk.Partitions)
		}
	}

	if rootLocation == "" {
		t.Fatal("ROOT filesystem not found! A partition labeled ROOT is requred")
	}

	// Let's make sure that all of the devices we needed to substitute names in
	// for were found
	for _, d := range test.MntDevices {
		if strings.Contains(test.Config, d.Substitution) {
			t.Fatalf("Didn't find a drive with label: %s", d.Substitution)
		}
	}

	// Ignition
	configDir := writeIgnitionConfig(t, test.Config)
	disks := runIgnition(t, "disks", rootLocation, configDir, negativeTests)
	files := runIgnition(t, "files", rootLocation, configDir, negativeTests)
	if negativeTests && disks && files {
		t.Fatal("Expected failure and ignition succeeded")
	}

	// Validation and cleanup
	for i, disk := range test.Out {
		// Update out structure with mount points & devices
		setExpectedPartitionsDrive(test.In[i].Partitions, disk.Partitions)

		if !negativeTests {
			// Validation
			mountPartitions(t, disk.Partitions)
			validatePartitions(t, disk.Partitions, test.In[i].ImageFile)
			validateFilesystems(t, disk.Partitions, test.In[i].ImageFile)
			validateFilesDirectoriesAndLinks(t, disk.Partitions)
			unmountPartitions(t, disk.Partitions)
		}

		// Cleanup
		unmountRootPartition(t, disk.Partitions)
		destroyDevices(t, disk.ImageFile)
		removeMountFolders(t, disk.Partitions)
		removeFile(t, disk.ImageFile)
	}
	_ = os.Setenv("PATH", path)
	_ = os.Setenv("TMPDIR", "")
	removeFile(t, filepath.Join(configDir, "config.ign"))
}
