// Copyright 2015 CoreOS, Inc.
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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type File struct {
	Name     string
	Path     string
	Contents []string
	Mode     string
}

type Partition struct {
	Number         int
	Label          string
	TypeCode       string
	TypeGUID       string
	GUID           string
	Device         string
	Offset         int
	Length         int
	FilesystemType string
	MountPath      string
	Hybrid         bool
	Files          []File
}

type MntDevice struct {
	label string
	code  string
}

type Test struct {
	name       string
	in         []*Partition
	out        []*Partition
	mntDevices []MntDevice
	config     string
}

func getBaseDisk() []*Partition {
	return []*Partition{
		{
			Number:         1,
			Label:          "EFI-SYSTEM",
			TypeCode:       "efi",
			Length:         262144,
			FilesystemType: "ext4",
			Hybrid:         true,
			Files: []File{
				{
					Name:     "multiLine",
					Path:     "path/example",
					Contents: []string{"line 1", "line 2"},
				}, {
					Name:     "singleLine",
					Path:     "another/path/example",
					Contents: []string{"single line"},
				}, {
					Name: "emptyFile",
					Path: "empty",
				}, {
					Name: "noPath",
					Path: "",
				},
			},
		}, {
			Number:   2,
			Label:    "BIOS-BOOT",
			TypeCode: "bios",
			Length:   4096,
		}, {
			Number:         3,
			Label:          "USR-A",
			GUID:           "7130c94a-213a-4e5a-8e26-6cce9662f132",
			TypeCode:       "coreos-rootfs",
			Length:         2097152,
			FilesystemType: "ext2",
		}, {
			Number:   4,
			Label:    "USR-B",
			GUID:     "e03dd35c-7c2d-4a47-b3fe-27f15780a57c",
			TypeCode: "coreos-rootfs",
			Length:   2097152,
		}, {
			Number:   5,
			Label:    "ROOT-C",
			GUID:     "d82521b4-07ac-4f1c-8840-ddefedc332f3",
			TypeCode: "blank",
			Length:   0,
		}, {
			Number:         6,
			Label:          "OEM",
			TypeCode:       "data",
			Length:         262144,
			FilesystemType: "ext4",
		}, {
			Number:   7,
			Label:    "OEM-CONFIG",
			TypeCode: "coreos-reserved",
			Length:   131072,
		}, {
			Number:   8,
			Label:    "coreos-reserved",
			TypeCode: "blank",
			Length:   0,
		}, {
			Number:         9,
			Label:          "ROOT",
			TypeCode:       "coreos-resize",
			Length:         12943360,
			FilesystemType: "ext4",
		},
	}
}

func newTest(name string, in []*Partition, out []*Partition, mntDevices []MntDevice, config string) Test {
	return Test{
		name:       name,
		in:         in,
		out:        out,
		mntDevices: mntDevices,
		config:     config,
	}
}

func createTests() []Test {
	tests := []Test{}

	name := "Reformat rootfs to ext4 & drop file in /ignition/test"
	in := getBaseDisk()
	out := getBaseDisk()
	mntDevices := []MntDevice{
		{
			label: "EFI-SYSTEM",
			code:  "$DEVICE",
		},
	}
	config := `{
		"ignition": {"version": "2.0.0"},
		"storage": {
			"filesystems": [{
				"mount": {
					"device": "$DEVICE",
					"format": "ext4",
					"create": {
						"force": true
					}},
				 "name": "test"}],
			"files": [{
				"filesystem": "test",
				"path": "/ignition/test",
				"contents": {"source": "data:,asdf"}
			}]}
	}`

	in[0].FilesystemType = "ext2"
	out[0].Files = []File{
		{
			Name:     "test",
			Path:     "ignition",
			Contents: []string{"asdf"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Create a systemd service"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
		"ignition": { "version": "2.0.0" },
		"systemd": {
			"units": [{
				"name": "example.service",
				"enable": true,
				"contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
			}]
		}
	}`
	out[8].Files = []File{
		{
			Name:     "example.service",
			Path:     "etc/systemd/system",
			Contents: []string{"[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"},
		},
		{
			Name:     "20-ignition.preset",
			Path:     "etc/systemd/system-preset",
			Contents: []string{"enable example.service", ""},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Modify Services"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
	  "ignition": { "version": "2.0.0" },
	  "systemd": {
	    "units": [{
	      "name": "systemd-networkd.service",
	      "dropins": [{
	        "name": "debug.conf",
	        "contents": "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"
	      }]
	    }]
	  }
	}`
	out[8].Files = []File{
		{
			Name:     "debug.conf",
			Path:     "etc/systemd/system/systemd-networkd.service.d",
			Contents: []string{"[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Reformat a Filesystem to Btrfs"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = []MntDevice{
		{
			label: "OEM",
			code:  "$DEVICE",
		},
	}
	config = `{
	  "ignition": { "version": "2.0.0" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "btrfs",
	        "create": {
	          "force": true,
	          "options": [ "--label=OEM" ]
	        }
	      }
	    }]
	  }
	}`
	out[5].FilesystemType = "btrfs"

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Reformat a Filesystem to XFS"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = []MntDevice{
		{
			label: "OEM",
			code:  "$DEVICE",
		},
	}
	config = `{
	  "ignition": { "version": "2.0.0" },
	  "storage": {
	    "filesystems": [{
	      "mount": {
	        "device": "$DEVICE",
	        "format": "xfs",
	        "create": {
	          "force": true,
	          "options": [ "-L", "OEM" ]
	        }
	      }
	    }]
	  }
	}`
	out[5].FilesystemType = "xfs"

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Setting the hostname"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
	  "ignition": { "version": "2.0.0" },
	  "storage": {
	    "files": [{
	      "filesystem": "root",
	      "path": "/etc/hostname",
	      "mode": 420,
	      "contents": { "source": "data:,core1" }
	    }]
	  }
	}`
	out[8].Files = []File{
		{
			Name:     "hostname",
			Path:     "etc",
			Contents: []string{"core1"},
			Mode:     "420",
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Create Files on the Root Filesystem"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
	  "ignition": { "version": "2.0.0" },
	  "storage": {
	    "files": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
	      "contents": { "source": "data:,example%20file%0A" }
	    }]
	  }
	}`
	out[8].Files = []File{
		{
			Name:     "bar",
			Path:     "foo",
			Contents: []string{"example file\n"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Create Files from Remote Contents"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
	  "ignition": { "version": "2.0.0" },
	  "storage": {
	    "files": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
	      "contents": {
	        "source": "https://gist.githubusercontent.com/arithx/af92c8a97c6ade777ea741047b18e815/raw/02fd7082ee25036c519d1caab9cf74ebc782b189/gistfile1.txt",
	        "verification": { "hash": "sha512-1a04c76c17079cd99e688ba4f1ba095b927d3fecf2b1e027af361dfeafb548f7f5f6fdd675aaa2563950db441d893ca77b0c3e965cdcb891784af96e330267d7" }
	      }
	    }]
	  }
	}`
	out[8].Files = []File{
		{
			Name:     "bar",
			Path:     "foo",
			Contents: []string{"asdf\nfdsa"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Replacing the Config with a Remote Config"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
	  "ignition": {
	    "version": "2.0.0",
	    "config": {
	      "replace": {
	        "source": "https://gist.githubusercontent.com/arithx/fd9fad926b01a35a57ac5c0cc77d2bf7/raw/56bacc7d24fc08aa7303d5eca57c3e3ad9074c32/gistfile1.txt",
	        "verification": { "hash": "sha512-632b67707297c47e309548466ea44d5eceb205e1595198045295c7b14a804fcc6a42b44166631d636bf17ea2c0e00a51a00190511c9333147115ed0be695df19" }
	      }
	    }
	  }
	}`
	out[8].Files = []File{
		{
			Name:     "bar",
			Path:     "foo",
			Contents: []string{"example file\n"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	name = "Adding users"
	in = getBaseDisk()
	out = getBaseDisk()
	mntDevices = nil
	config = `{
		"ignition": {
			"version": "2.0.0"
		},
		"passwd": {
			"users": [{
					"name": "test",
					"create": {},
					"passwordHash": "zJW/EKqqIk44o",
					"sshAuthorizedKeys": [
						"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDBRZPFJNOvQRfokigTtl0IBi71LHZrFOk4EJ3Zowtk/bX5uIVai0Cd4+hqlocYL10idgtFBH28skeKfsmHwgS9XwOvP+g+kqAl7yCz8JEzIUzl1fxNZDToi0jA3B5MwXkpt+IWfnabwi2cRZhlzrz9rO+eExu5s3NfaRmmmCYrjCJIRPKSCrW8U0n9fVSbX4PDdMXVmH7r+t8MtR8523vCbakFR/Y0YIqkPVdfuUXHh9rDCdH4B7mt7nYX2LWQXGUvmI13mgQoy04ifkaR3ImuOMp3Y1J1gm6clO74IMCq/sK9+XJhbxMPPHUoUJ2EwbaG7Dbh3iqz47e9oVki4gIH stephenlowrie@localhost.localdomain"
					]
				},
				{
					"name": "jenkins",
					"create": {
						"uid": 1000
					}
				}
			]
		}
	}`
	out[8].Files = []File{
		{
			Name:     "passwd",
			Path:     "etc",
			Contents: []string{"TODO"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	return tests
}

func TestIgnitionBlackBox(t *testing.T) {
	t.Log("Entered TestIgnitionBlackBox")
	tests := createTests()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outer(t, test)
		})
		//outer(t, test)
	}
}

func PreCleanup(t *testing.T) {
	mountpoints, _ := exec.Command("findmnt", "-l", "-o", "target").CombinedOutput()
	points := strings.Split(string(mountpoints), "\n")
	for i := len(points) - 1; i >= 0; i-- {
		for _, pat := range []string{"/tmp/hd1p*", "/tmp/hd1p*/*"} {
			match, err := filepath.Match(pat, points[i])
			if err != nil {
				t.Log(err)
			}
			if match {
				_, _ = exec.Command("umount", points[i]).CombinedOutput()
			}
		}
	}
	removeFile(t, "config.ign")
	removeFile(t, "test.img")
}

func outer(t *testing.T, test Test) {
	PreCleanup(t)
	t.Log(test.name)

	imgName := "test.img"
	imageSize := calculateImageSize(test.in)

	// Finish data setup
	for _, part := range test.in {
		if part.GUID == "" {
			part.GUID = generateUUID(t)
		}
		updateTypeGUID(t, part)
	}
	setOffsets(test.in)
	for _, part := range test.out {
		updateTypeGUID(t, part)
	}
	setOffsets(test.out)

	// Creation
	createVolume(t, imgName, imageSize, 20, 16, 63, test.in)
	setDevices(t, imgName, test.in)
	mountRootPartition(t, test.in)
	copyIdToRootPartition(t, test.in)
	mountPartitions(t, test.in)
	createFiles(t, test.in)
	//dumpDiskInfo(t, imgName, test.in)
	unmountPartitions(t, test.in)

	// Ignition
	config := test.config
	for _, d := range test.mntDevices {
		device := pickDevice(t, test.in, imgName, d.label)
		config = strings.Replace(config, d.code, device, -1)
		//t.Log(config, device, d.code)
	}
	writeIgnitionConfig(t, config)
	root := getRootLocation(t, test.in)
	runIgnition(t, "disks", root)
	runIgnition(t, "files", root)

	// Update out structure with mount points & devices
	setExpectedPartitionsDrive(test.in, test.out)

	// Validation
	mountPartitions(t, test.out)
	//dumpDiskInfo(t, imgName, test.out)
	validatePartitions(t, test.out, imgName)
	validateFiles(t, test.out)

	// Cleanup
	unmountPartitions(t, test.out)
	unmountRootPartition(t, test.out)
	removeMountFolders(t, test.out)
	removeFile(t, "config.ign")
	removeFile(t, imgName)
}

func copyIdToRootPartition(t *testing.T, partitions []*Partition) {
	for _, p := range partitions {
		if p.Label == "ROOT" {
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "home"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "lib64"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "var/log"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "sbin"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "bin"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "etc/default"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "proc/self"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "proc/sys/kernel"}, "/"), 0755)
			_ = os.MkdirAll(strings.Join([]string{p.MountPath, "usr/share/baselayout"}, "/"), 0755)
			_, _ = exec.Command("cp", "/lib64/libselinux.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libc.so.6", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libdl.so.2", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libpcre.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/ld-linux-x86-64.so.2", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libpthread.so.0", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libaudit-vdso.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libselinux.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libsemanage.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libacl.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libattr.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libcap-ng.so.0", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libsepol.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libbz2.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libustr-1.0.so.1", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libpthread.so.0", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libnss_files.so.2", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/lib64/libtinfo.so.6", strings.Join([]string{p.MountPath, "lib64"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/etc/ld.so.cache", strings.Join([]string{p.MountPath, "etc"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/etc/login.defs", strings.Join([]string{p.MountPath, "etc"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/etc/nsswitch.conf", strings.Join([]string{p.MountPath, "etc"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/etc/passwd", strings.Join([]string{p.MountPath, "etc"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/etc/group", strings.Join([]string{p.MountPath, "etc"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/etc/default/useradd", strings.Join([]string{p.MountPath, "etc/default"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/sbin/useradd", strings.Join([]string{p.MountPath, "sbin"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/bin/id", strings.Join([]string{p.MountPath, "bin"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/bin/bash", strings.Join([]string{p.MountPath, "bin"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/usr/share/baselayout/group", strings.Join([]string{p.MountPath, "usr/share/baselayout"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/usr/share/baselayout/passwd", strings.Join([]string{p.MountPath, "usr/share/baselayout"}, "/")).CombinedOutput()
			_, _ = exec.Command("cp", "/usr/share/baselayout/nsswitch.conf", strings.Join([]string{p.MountPath, "usr/share/baselayout"}, "/")).CombinedOutput()
		}
	}
}

func getRootLocation(t *testing.T, partitions []*Partition) string {
	for _, p := range partitions {
		if p.Label == "ROOT" {
			return p.MountPath
		}
	}
	t.Fatal("ROOT filesystem not found! A partition labeled ROOT is requred")
	return ""
}

func removeFile(t *testing.T, imgName string) {
	err := os.Remove(imgName)
	if err != nil {
		//t.Log(err)
	}
}

func removeMountFolders(t *testing.T, partitions []*Partition) {
	for _, p := range partitions {
		err := os.RemoveAll(p.MountPath)
		if err != nil {
			t.Log(err)
		}
	}
}

func runIgnition(t *testing.T, stage string, root string) {
	out, err := exec.Command(
		"ignition", "-clear-cache", "-oem",
		"file", "-stage", stage, "-root", root).CombinedOutput()
	debugInfo, derr := ioutil.ReadFile("/var/log/syslog")
	if derr == nil {
		debugOut := []string{}
		lines := strings.Split(string(debugInfo), "\n")
		for _, line := range lines {
			if strings.Contains(line, "ignition") {
				debugOut = append(debugOut, line)
			}
		}
		//t.Log(derr, debugOut)
	}
	if err != nil {
		t.Fatal("ignition", err, string(out))
	}

}

func pickDevice(t *testing.T, partitions []*Partition, fileName string, label string) string {
	number := -1
	for _, p := range partitions {
		if p.Label == label {
			number = p.Number
		}
	}
	if number == -1 {
		t.Fatal("Didn't find a drive with label:", label)
		return ""
	}

	kpartxOut, err := exec.Command("kpartx", "-l", fileName).CombinedOutput()
	if err != nil {
		t.Fatal("kpartx -l", err, string(kpartxOut))
	}
	//t.Log(string(kpartxOut))
	return fmt.Sprintf("/dev/mapper/%sp%d",
		strings.Trim(strings.Split(string(kpartxOut), " ")[4], "/dev/"), number)
}

func writeIgnitionConfig(t *testing.T, config string) {
	err := ioutil.WriteFile("config.ign", []byte(config), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func calculateImageSize(partitions []*Partition) int64 {
	size := int64(63 * 512)
	for _, p := range partitions {
		size += int64(align(p.Length, 512) * 512)
	}
	size = size + int64(4096*512) // extra room to allow for alignments
	return size
}

func dumpDiskInfo(t *testing.T, fileName string, partitions []*Partition) {
	ptTable, err := exec.Command(
		"/sbin/sgdisk", "-p", fileName).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(ptTable))

	mounts, err := exec.Command("/bin/cat", "/proc/mounts").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(mounts))

	for _, p := range partitions {
		if p.TypeCode == "blank" {
			continue
		}
		sgdisk, err := exec.Command(
			"/sbin/sgdisk", "-i", strconv.Itoa(p.Number),
			fileName).CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(sgdisk))
	}
}

func createVolume(
	t *testing.T, fileName string, size int64, cylinders int, heads int,
	sectorsPerTrack int, partitions []*Partition) {
	// attempt to create the file, will leave already existing files alone.
	// os.Truncate requires the file to already exist
	out, err := os.Create(fileName)
	if err != nil {
		t.Fatal("create", err, out)
	}
	out.Close()

	// Truncate the file to the given size
	err = os.Truncate(fileName, size)
	if err != nil {
		t.Fatal("truncate", err)
	}

	createPartitionTable(t, fileName, partitions)

	for counter, partition := range partitions {
		if partition.TypeCode == "blank" || partition.FilesystemType == "" {
			continue
		}

		mntPath, err := ioutil.TempDir("", fmt.Sprintf("%s%d", "hd1p", counter))
		if err != nil {
			t.Fatal(err)
		}
		partition.MountPath = mntPath
	}
}

func setDevices(t *testing.T, fileName string, partitions []*Partition) {
	loopDevice := kpartxAdd(t, fileName)

	for _, partition := range partitions {
		if partition.TypeCode == "blank" || partition.FilesystemType == "" {
			continue
		}

		partition.Device = fmt.Sprintf(
			"/dev/mapper/%sp%d", loopDevice, partition.Number)
		formatPartition(t, partition)
	}
}

func formatPartition(t *testing.T, partition *Partition) {
	switch partition.FilesystemType {
	case "vfat":
		formatVFAT(t, partition)
	case "ext2", "ext4":
		formatEXT(t, partition)
	case "btrfs":
		formatBTRFS(t, partition)
	default:
		if partition.FilesystemType == "blank" || partition.FilesystemType == "" {
			return
		}
		t.Fatal("Unknown partition", partition.FilesystemType)
	}
}

func formatVFAT(t *testing.T, partition *Partition) {
	opts := []string{}
	if partition.Label != "" {
		opts = append(opts, "-n", partition.Label)
	}
	opts = append(
		opts, partition.Device)
	out, err := exec.Command("/sbin/mkfs.vfat", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("mkfs.vfat", err, string(out))
	}
}

func formatEXT(t *testing.T, partition *Partition) {
	out, err := exec.Command(
		"/sbin/mke2fs", "-q", "-t", partition.FilesystemType, "-b", "4096",
		"-i", "4096", "-I", "128", partition.Device).CombinedOutput()
	if err != nil {
		t.Fatal("mke2fs", err, string(out))
	}

	opts := []string{"-e", "remount-ro"}
	if partition.Label != "" {
		opts = append(opts, "-L", partition.Label)
	}

	if partition.TypeCode == "coreos-usr" {
		opts = append(
			opts, "-U", "clear", "-T", "20091119110000", "-c", "0", "-i", "0",
			"-m", "0", "-r", "0")
	}
	opts = append(opts, partition.Device)
	tuneOut, err := exec.Command("/sbin/tune2fs", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("tune2fs", err, string(tuneOut))
	}
}

func formatBTRFS(t *testing.T, partition *Partition) {
	opts := []string{}
	if partition.Label != "" {
		opts = append(opts, "--label", partition.Label)
	}
	opts = append(opts, partition.Device)
	out, err := exec.Command("/sbin/mkfs.btrfs", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("mkfs.btrfs", err, string(out))
	}

	// todo: subvolumes?
}

func align(count int, alignment int) int {
	offset := count % alignment
	if offset != 0 {
		count += alignment - offset
	}
	return count
}

func setOffsets(partitions []*Partition) {
	offset := 34
	for _, p := range partitions {
		if p.Length == 0 || p.TypeCode == "blank" {
			continue
		}
		offset = align(offset, 4096)
		p.Offset = offset
		offset += p.Length
	}
}

func createPartitionTable(
	t *testing.T, fileName string, partitions []*Partition) {
	opts := []string{fileName}
	hybrids := []int{}
	for _, p := range partitions {
		if p.TypeCode == "blank" || p.Length == 0 {
			continue
		}
		opts = append(opts, fmt.Sprintf(
			"--new=%d:%d:+%d", p.Number, p.Offset, p.Length))
		opts = append(opts, fmt.Sprintf(
			"--change-name=%d:%s", p.Number, p.Label))
		if p.TypeGUID != "" {
			opts = append(opts, fmt.Sprintf(
				"--typecode=%d:%s", p.Number, p.TypeGUID))
		}
		if p.GUID != "" {
			opts = append(opts, fmt.Sprintf(
				"--partition-guid=%d:%s", p.Number, p.GUID))
		}
		if p.Hybrid {
			hybrids = append(hybrids, p.Number)
		}
	}
	if len(hybrids) > 0 {
		if len(hybrids) > 3 {
			t.Fatal("Can't have more than three hybrids")
		} else {
			opts = append(opts, fmt.Sprintf("-h=%s", intJoin(hybrids, ":")))
		}
	}
	//t.Log("/sbin/sgdisk", strings.Join(opts, " "))
	sgdiskOut, err := exec.Command(
		"/sbin/sgdisk", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("sgdisk", err, string(sgdiskOut))
	}
}

func kpartxAdd(t *testing.T, fileName string) string {
	kpartxOut, err := exec.Command(
		"/sbin/kpartx", "-av", fileName).CombinedOutput()
	if err != nil {
		t.Fatal("kpartx", err, string(kpartxOut))
	}
	kpartxOut, err = exec.Command(
		"/sbin/kpartx", "-l", fileName).CombinedOutput()
	//t.Log(string(kpartxOut), err)
	return strings.Trim(strings.Split(string(kpartxOut), " ")[4], "/dev/")
}

func mountRootPartition(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.Label != "ROOT" {
			continue
		}
		mountOut, err := exec.Command(
			"/bin/mount", partition.Device,
			partition.MountPath).CombinedOutput()
		if err != nil {
			t.Fatal("mount", err, string(mountOut))
		}
		_ = os.MkdirAll(filepath.Join(partition.MountPath, "proc"), 0755)
		_, err = exec.Command("mount", "-t", "proc", "none", strings.Join([]string{partition.MountPath, "proc"}, "/")).CombinedOutput()
		if err != nil {
			t.Log(err)
		}
		_ = os.MkdirAll(filepath.Join(partition.MountPath, "usr"), 0755)
		mountBind, err := exec.Command(
			"/bin/mount", "--bind", "/usr",
			filepath.Join(partition.MountPath, "usr")).CombinedOutput()
		if err != nil {
			t.Fatal("mount", err, string(mountBind))
		}
		return
	}
	t.Fatal("Didn't find the ROOT partition to mount")
}

func mountPartitions(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.FilesystemType == "" || partition.Label == "ROOT" {
			continue
		}
		mountOut, err := exec.Command(
			"/bin/mount", partition.Device,
			partition.MountPath).CombinedOutput()
		if err != nil {
			t.Fatal("mount", err, string(mountOut))
		}
	}
}

func updateTypeGUID(t *testing.T, partition *Partition) {
	switch partition.TypeCode {
	case "coreos-resize":
		partition.TypeGUID = "3884DD41-8582-4404-B9A8-E9B84F2DF50E"
	case "data":
		partition.TypeGUID = "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
	case "coreos-rootfs":
		partition.TypeGUID = "5DFBF5F4-2848-4BAC-AA5E-0D9A20B745A6"
	case "bios":
		partition.TypeGUID = "21686148-6449-6E6F-744E-656564454649"
	case "efi":
		partition.TypeGUID = "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
	case "coreos-reserved":
		partition.TypeGUID = "C95DC21A-DF0E-4340-8D7B-26CBFA9A03E0"
	case "", "blank":
		return
	default:
		t.Fatal("Unknown TypeCode", partition.TypeCode)
	}
}

func intJoin(ints []int, delimiter string) string {
	strArr := []string{}
	for _, i := range ints {
		strArr = append(strArr, strconv.Itoa(i))
	}
	return strings.Join(strArr, delimiter)
}

func removeEmpty(strings []string) []string {
	var r []string
	for _, str := range strings {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func generateUUID(t *testing.T) string {
	out, err := exec.Command("/usr/bin/uuidgen").CombinedOutput()
	if err != nil {
		t.Fatal("uuidgen", err)
	}
	return strings.TrimSpace(string(out))
}

func createFiles(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.Files == nil {
			continue
		}
		for _, file := range partition.Files {
			err := os.MkdirAll(strings.Join(removeEmpty([]string{
				partition.MountPath, file.Path}), "/"), 0644)
			if err != nil {
				t.Fatal("mkdirall", err)
			}
			f, err := os.Create(strings.Join(removeEmpty([]string{
				partition.MountPath, file.Path, file.Name}), "/"))
			defer f.Close()
			if err != nil {
				t.Fatal("create", err, f)
			}
			if file.Contents != nil {
				writer := bufio.NewWriter(f)
				writeStringOut, err := writer.WriteString(
					strings.Join(file.Contents, "\n"))
				if err != nil {
					t.Fatal("writeString", err, string(writeStringOut))
				}
				writer.Flush()
			}
		}
	}
}

func unmountRootPartition(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.Label != "ROOT" {
			continue
		}
		umountUsr, err := exec.Command(
			"/bin/umount", filepath.Join(partition.MountPath, "usr")).CombinedOutput()
		if err != nil {
			t.Fatal("umount", err, string(umountUsr))
		}
		umountProc, err := exec.Command(
			"/bin/umount", fmt.Sprintf("%s/proc", partition.MountPath)).CombinedOutput()
		if err != nil {
			t.Fatal("umount", err, string(umountProc))
		}
		umountOut, err := exec.Command(
			"/bin/umount", partition.Device).CombinedOutput()
		if err != nil {
			t.Fatal("umount", err, string(umountOut))
		}
	}
}

func unmountPartitions(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.FilesystemType == "" || partition.Label == "ROOT" {
			continue
		}
		umountOut, err := exec.Command(
			"/bin/umount", partition.Device).CombinedOutput()
		if err != nil {
			t.Fatal("umount", err, string(umountOut))
		}
	}
}

func setExpectedPartitionsDrive(actual []*Partition, expected []*Partition) {
	for _, a := range actual {
		for _, e := range expected {
			if a.Number == e.Number {
				e.MountPath = a.MountPath
				e.Device = a.Device
				break
			}
		}
	}
}

func validatePartitions(
	t *testing.T, expected []*Partition, fileName string) {
	for _, e := range expected {
		if e.TypeCode == "blank" {
			continue
		}
		sgdiskInfo, err := exec.Command(
			"/sbin/sgdisk", "-i", strconv.Itoa(e.Number),
			fileName).CombinedOutput()
		if err != nil {
			t.Fatal("sgdisk -i", strconv.Itoa(e.Number), err)
		}
		lines := strings.Split(string(sgdiskInfo), "\n")
		actualTypeGUID := strings.ToUpper(strings.TrimSpace(
			strings.Split(strings.Split(lines[0], ": ")[1], " ")[0]))
		actualSectors := strings.Split(strings.Split(lines[4], ": ")[1], " ")[0]
		actualLabel := strings.Split(strings.Split(lines[6], ": ")[1], "'")[1]

		// have to align the size to the nearest sector first
		expectedSectors := align(e.Length, 512)

		if e.TypeGUID != actualTypeGUID {
			t.Fatal("TypeGUID does not match!", e.TypeGUID, actualTypeGUID)
		}
		if e.Label != actualLabel {
			t.Fatal("Label does not match!", e.Label, actualLabel)
		}
		if strconv.Itoa(expectedSectors) != actualSectors {
			t.Fatal(
				"Sectors does not match!", expectedSectors, actualSectors)
		}

		if e.FilesystemType == "" {
			continue
		}

		df, err := exec.Command("/bin/df", "-T", e.Device).CombinedOutput()
		if err != nil {
			t.Fatal("df -T", err, string(df))
		}
		//t.Log(e.Device, (string(df)))
		lines = strings.Split(string(df), "\n")
		if len(lines) < 2 {
			t.Fatal("Couldn't verify FilesystemType")
		}
		actualFilesystemType := removeEmpty(strings.Split(lines[1], " "))[1]

		if e.FilesystemType != actualFilesystemType {
			t.Fatal("FilesystemType does not match!", e.Label,
				e.FilesystemType, actualFilesystemType)
		}
	}
}

func validateFiles(t *testing.T, expected []*Partition) {
	for _, partition := range expected {
		if partition.Files == nil {
			continue
		}
		for _, file := range partition.Files {
			path := strings.Join(removeEmpty([]string{
				partition.MountPath, file.Path, file.Name}), "/")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Fatal("File doesn't exist!", path)
			}

			if file.Contents != nil {
				expectedContents := strings.Join(file.Contents, "\n")
				dat, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatal("Error when reading file", path)
				}

				actualContents := string(dat)
				if expectedContents != actualContents {
					t.Fatal("Contents of file", path, "do not match!",
						expectedContents, actualContents)
				}
			}

			if file.Mode != "" {
				sout, err := exec.Command(
					"stat", "-c", "%a", path).CombinedOutput()
				statOut := string(sout)
				if err != nil {
					t.Fatal(err)
				}
				if file.Mode != statOut {
					t.Fatal("File Mode does not match", path, file.Mode, statOut)
				}
			}
		}
	}
}
