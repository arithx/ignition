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
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
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

func newTest(name string, in []*Partition, out []*Partition,
	mntDevices []MntDevice, config string) Test {
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
			Mode:     "644",
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
	        "source": "http://127.0.0.1:8080/contents"
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
	        "source": "http://127.0.0.1:8080/config"
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
						"uid": 1020
					}
				}
			]
		}
	}`
	in[8].Files = []File{
		{
			Name:     "passwd",
			Path:     "etc",
			Contents: []string{"root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\n"},
		},
		{
			Name:     "shadow",
			Path:     "etc",
			Contents: []string{"root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\n"},
		},
		{
			Name:     "group",
			Path:     "etc",
			Contents: []string{"root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\n"},
		},
		{
			Name:     "gshadow",
			Path:     "etc",
			Contents: []string{"root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\n"},
		},
		{
			Name:     "nsswitch.conf",
			Path:     "etc",
			Contents: []string{"# /etc/nsswitch.conf:\n\npasswd:      files\nshadow:      files\ngroup:       files\n\nhosts:       files dns myhostname\nnetworks:    files dns\n\nservices:    files\nprotocols:   files\nrpc:         files\n\nethers:      files\nnetmasks:    files\nnetgroup:    files\nbootparams:  files\nautomount:   files\naliases:     files\n"},
		},
		{
			Name: "login.defs",
			Path: "etc",
			Contents: []string{`#
# Please note that the parameters in this configuration file control the
# behavior of the tools from the shadow-utils component. None of these
# tools uses the PAM mechanism, and the utilities that use PAM (such as the
# passwd command) should therefore be configured elsewhere. Refer to
# /etc/pam.d/system-auth for more information.
#

# *REQUIRED*
#   Directory where mailboxes reside, _or_ name of file, relative to the
#   home directory.  If you _do_ define both, MAIL_DIR takes precedence.
#   QMAIL_DIR is for Qmail
#
#QMAIL_DIR	Maildir
MAIL_DIR	/var/spool/mail
#MAIL_FILE	.mail

# Password aging controls:
#
#	PASS_MAX_DAYS	Maximum number of days a password may be used.
#	PASS_MIN_DAYS	Minimum number of days allowed between password changes.
#	PASS_MIN_LEN	Minimum acceptable password length.
#	PASS_WARN_AGE	Number of days warning given before a password expires.
#
PASS_MAX_DAYS	99999
PASS_MIN_DAYS	0
PASS_MIN_LEN	5
PASS_WARN_AGE	7

#
# Min/max values for automatic uid selection in useradd
#
UID_MIN                  1000
UID_MAX                 60000
# System accounts
SYS_UID_MIN               201
SYS_UID_MAX               999

#
# Min/max values for automatic gid selection in groupadd
#
GID_MIN                  1000
GID_MAX                 60000
# System accounts
SYS_GID_MIN               201
SYS_GID_MAX               999

#
# If defined, this command is run when removing a user.
# It should remove any at/cron/print jobs etc. owned by
# the user to be removed (passed as the first argument).
#
#USERDEL_CMD	/usr/sbin/userdel_local

#
# If useradd should create home directories for users by default
# On RH systems, we do. This option is overridden with the -m flag on
# useradd command line.
#
CREATE_HOME	yes

# The permission mask is initialized to this value. If not specified,
# the permission mask will be initialized to 022.
UMASK           077

# This enables userdel to remove user groups if no members exist.
#
USERGROUPS_ENAB yes

# Use SHA512 to encrypt password.
ENCRYPT_METHOD SHA512
`},
		},
	}
	out[8].Files = []File{
		{
			Name:     "passwd",
			Path:     "etc",
			Contents: []string{"root:x:0:0:root:/root:/bin/bash\ncore:x:500:500:CoreOS Admin:/home/core:/bin/bash\nsystemd-coredump:x:998:998:systemd Core Dumper:/:/sbin/nologin\nfleet:x:253:253::/:/sbin/nologin\ntest:x:1000:1000::/home/test:/bin/bash\njenkins:x:1020:1001::/home/jenkins:/bin/bash\n"},
		},
		{
			Name:     "group",
			Path:     "etc",
			Contents: []string{"root:x:0:root\nwheel:x:10:root,core\nsudo:x:150:\ndocker:x:233:core\nsystemd-coredump:x:998:\nfleet:x:253:core\ncore:x:500:\nrkt-admin:x:999:\nrkt:x:251:core\ntest:x:1000:\njenkins:x:1001:\n"},
		},
		{
			Name:     "shadow",
			Path:     "etc",
			Contents: []string{"root:*:15887:0:::::\ncore:*:15887:0:::::\nsystemd-coredump:!!:17301::::::\nfleet:!!:17301::::::\ntest:zJW/EKqqIk44o:17331:0:99999:7:::\njenkins:*:17331:0:99999:7:::\n"},
		},
		{
			Name:     "gshadow",
			Path:     "etc",
			Contents: []string{"root:*::root\nusers:*::\nsudo:*::\nwheel:*::root,core\nsudo:*::\ndocker:*::core\nsystemd-coredump:!!::\nfleet:!!::core\nrkt-admin:!!::\nrkt:!!::core\ncore:*::\ntest:!::\njenkins:!::\n"},
		},
		{
			Name:     "authorized_keys",
			Path:     "home/test/.ssh",
			Contents: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDBRZPFJNOvQRfokigTtl0IBi71LHZrFOk4EJ3Zowtk/bX5uIVai0Cd4+hqlocYL10idgtFBH28skeKfsmHwgS9XwOvP+g+kqAl7yCz8JEzIUzl1fxNZDToi0jA3B5MwXkpt+IWfnabwi2cRZhlzrz9rO+eExu5s3NfaRmmmCYrjCJIRPKSCrW8U0n9fVSbX4PDdMXVmH7r+t8MtR8523vCbakFR/Y0YIqkPVdfuUXHh9rDCdH4B7mt7nYX2LWQXGUvmI13mgQoy04ifkaR3ImuOMp3Y1J1gm6clO74IMCq/sK9+XJhbxMPPHUoUJ2EwbaG7Dbh3iqz47e9oVki4gIH stephenlowrie@localhost.localdomain\n\n"},
		},
	}

	tests = append(tests, newTest(name, in, out, mntDevices, config))

	return tests
}

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

func TestIgnitionBlackBox(t *testing.T) {
	t.Log("Entered TestIgnitionBlackBox")
	tests := createTests()

	server := &HTTPServer{}
	server.Start()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outer(t, test)
		})
	}
}

type HTTPServer struct{}

func (server *HTTPServer) Start() {
	http.HandleFunc("/contents", server.Contents)
	http.HandleFunc("/config", server.Config)

	s := &http.Server{Addr: ":8080"}
	go s.ListenAndServe()
}

func PreCleanup(t *testing.T) {
	mountpoints, _ := exec.Command(
		"findmnt", "-l", "-o", "target").CombinedOutput()
	points := strings.Split(string(mountpoints), "\n")
	for i := len(points) - 1; i >= 0; i-- {
		pat := "/tmp/hd1p*"
		match, err := filepath.Match(pat, points[i])
		if err != nil {
			t.Log(err)
		}
		if match {
			_, _ = exec.Command("umount", points[i]).CombinedOutput()
		}
	}
}

func outer(t *testing.T, test Test) {
	PreCleanup(t)
	t.Log(test.name)

	path := os.Getenv("PATH")
	cwd, _ := os.Getwd()
	_ = os.Setenv("PATH", fmt.Sprintf(
		"%s:%s", filepath.Join(cwd, "bin/amd64"), path))

	// the image file is written into cwd because sgdisk fails when the file
	// is located in /tmp/
	imageFile := "blackbox_ignition_test.img"
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
	createVolume(t, imageFile, imageSize, 20, 16, 63, test.in)
	setDevices(t, imageFile, test.in)
	mountRootPartition(t, test.in)
	if strings.Contains(test.config, "passwd") {
		prepareRootPartitionForPasswd(t, test.in)
	}
	mountPartitions(t, test.in)
	createFiles(t, test.in)
	unmountPartitions(t, test.in)

	// Ignition
	config := test.config
	for _, d := range test.mntDevices {
		device := pickDevice(t, test.in, imageFile, d.label)
		config = strings.Replace(config, d.code, device, -1)
	}
	configDir := writeIgnitionConfig(t, config)
	root := getRootLocation(t, test.in)
	runIgnition(t, "disks", root, configDir)
	runIgnition(t, "files", root, configDir)

	// Update out structure with mount points & devices
	setExpectedPartitionsDrive(test.in, test.out)

	// Validation
	mountPartitions(t, test.out)
	validatePartitions(t, test.out, imageFile)
	validateFiles(t, test.out)

	// Cleanup
	_ = os.Setenv("PATH", path)
	unmountPartitions(t, test.out)
	unmountRootPartition(t, test.out)
	removeMountFolders(t, test.out)
	removeFile(t, filepath.Join(configDir, "config.ign"))
	removeFile(t, imageFile)
}

func prepareRootPartitionForPasswd(t *testing.T, partitions []*Partition) {
	for _, p := range partitions {
		if p.Label == "ROOT" {
			_ = os.MkdirAll(filepath.Join(p.MountPath, "home"), 0755)
			_ = os.MkdirAll(filepath.Join(p.MountPath, "usr", "bin"), 0755)
			_ = os.MkdirAll(filepath.Join(p.MountPath, "usr", "sbin"), 0755)
			_ = os.MkdirAll(filepath.Join(p.MountPath, "usr", "lib64"), 0755)
			_ = os.MkdirAll(filepath.Join(p.MountPath, "etc"), 0755)

			_ = os.Symlink(
				filepath.Join(p.MountPath, "usr", "lib64"), filepath.Join(
					p.MountPath, "lib64"))
			_ = os.Symlink(
				filepath.Join(p.MountPath, "usr", "bin"), filepath.Join(
					p.MountPath, "bin"))
			_ = os.Symlink(
				filepath.Join(p.MountPath, "usr", "sbin"), filepath.Join(
					p.MountPath, "sbin"))

			_, _ = exec.Command(
				"cp", "/etc/ld.so.cache", filepath.Join(
					p.MountPath, "etc")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "/lib64/libblkid.so.1", filepath.Join(
					p.MountPath, "usr", "lib64")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "/lib64/libpthread.so.0", filepath.Join(
					p.MountPath, "usr", "lib64")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "/lib64/libc.so.6", filepath.Join(
					p.MountPath, "usr", "lib64")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "/lib64/libuuid.so.1", filepath.Join(
					p.MountPath, "usr", "lib64")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "/lib64/ld-linux-x86-64.so.2", filepath.Join(
					p.MountPath, "usr", "lib64")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "/lib64/libnss_files.so.2", filepath.Join(
					p.MountPath, "usr", "lib64")).CombinedOutput()

			_, _ = exec.Command(
				"cp", "bin/amd64/id", filepath.Join(
					p.MountPath, "usr", "bin")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "bin/amd64/useradd", filepath.Join(
					p.MountPath, "usr", "sbin")).CombinedOutput()
			_, _ = exec.Command(
				"cp", "bin/amd64/usermod", filepath.Join(
					p.MountPath, "usr", "sbin")).CombinedOutput()
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

func removeFile(t *testing.T, imageFile string) {
	err := os.Remove(imageFile)
	if err != nil {
		t.Log(err)
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

func runIgnition(t *testing.T, stage, root, configDir string) {
	cmd := exec.Command(
		"ignition", "-clear-cache", "-oem",
		"file", "-stage", stage, "-root", root)
	cmd.Dir = configDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("ignition", err, string(out))
	}
}

func pickDevice(t *testing.T, partitions []*Partition,
	fileName string, label string) string {
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
	return fmt.Sprintf("/dev/mapper/%sp%d",
		strings.Trim(strings.Split(string(kpartxOut), " ")[4], "/dev/"), number)
}

func writeIgnitionConfig(t *testing.T, config string) string {
	tmpDir, err := ioutil.TempDir("", "config")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(
		filepath.Join(tmpDir, "config.ign"), []byte(config), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func calculateImageSize(partitions []*Partition) int64 {
	// 63 is the number of sectors cpgt uses when generating a hybrid MBR
	size := int64(63 * 512)
	for _, p := range partitions {
		size += int64(align(p.Length, 512) * 512)
	}
	size = size + int64(4096*512) // extra room to allow for alignments
	return size
}

func createVolume(
	t *testing.T, imageFile string, size int64, cylinders int, heads int,
	sectorsPerTrack int, partitions []*Partition) {
	// attempt to create the file, will leave already existing files alone.
	// os.Truncate requires the file to already exist
	out, err := os.Create(imageFile)
	if err != nil {
		t.Fatal("create", err, out)
	}
	out.Close()

	// Truncate the file to the given size
	err = os.Truncate(imageFile, size)
	if err != nil {
		t.Fatal("truncate", err)
	}

	createPartitionTable(t, imageFile, partitions)

	for counter, partition := range partitions {
		if partition.TypeCode == "blank" || partition.FilesystemType == "" {
			continue
		}

		mntPath, err := ioutil.TempDir("", fmt.Sprintf("hd1p%d", counter))
		if err != nil {
			t.Fatal(err)
		}
		partition.MountPath = mntPath
	}
}

func setDevices(t *testing.T, imageFile string, partitions []*Partition) {
	loopDevice := kpartxAdd(t, imageFile)

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
	case "xfs":
		formatXFS(t, partition)
	case "swap":
		formatSWAP(t, partition)
	default:
		if partition.FilesystemType == "blank" ||
			partition.FilesystemType == "" {
			return
		}
		t.Fatal("Unknown partition", partition.FilesystemType)
	}
}

func formatSWAP(t *testing.T, partition *Partition) {
	opts := []string{}
	if partition.Label != "" {
		opts = append(opts, "-L", partition.Label)
	}
	if partition.GUID != "" {
		opts = append(opts, "-U", partition.GUID)
	}
	opts = append(
		opts, partition.Device)
	out, err := exec.Command("mkswap", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("mkswap", err, string(out))
	}
}

func formatVFAT(t *testing.T, partition *Partition) {
	opts := []string{}
	if partition.Label != "" {
		opts = append(opts, "-n", partition.Label)
	}
	opts = append(
		opts, partition.Device)
	out, err := exec.Command("mkfs.vfat", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("mkfs.vfat", err, string(out))
	}
}

func formatXFS(t *testing.T, partition *Partition) {
	opts := []string{}
	if partition.Label != "" {
		opts = append(opts, "-L", partition.Label)
	}
	if partition.GUID != "" {
		opts = append(opts, "-m", partition.GUID)
	}
	opts = append(
		opts, partition.Device)
	out, err := exec.Command("mkfs.xfs", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("mkfs.xfs", err, string(out))
	}
}

func formatEXT(t *testing.T, partition *Partition) {
	out, err := exec.Command(
		"mke2fs", "-q", "-t", partition.FilesystemType, "-b", "4096",
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
	tuneOut, err := exec.Command("tune2fs", opts...).CombinedOutput()
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
	out, err := exec.Command("mkfs.btrfs", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("mkfs.btrfs", err, string(out))
	}
}

func align(count int, alignment int) int {
	offset := count % alignment
	if offset != 0 {
		count += alignment - offset
	}
	return count
}

func setOffsets(partitions []*Partition) {
	// 34 is the first non-reserved GPT sector
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
	t *testing.T, imageFile string, partitions []*Partition) {
	opts := []string{imageFile}
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
	sgdiskOut, err := exec.Command("sgdisk", opts...).CombinedOutput()
	if err != nil {
		t.Fatal("sgdisk", err, string(sgdiskOut))
	}
}

func kpartxAdd(t *testing.T, imageFile string) string {
	kpartxOut, err := exec.Command(
		"kpartx", "-av", imageFile).CombinedOutput()
	if err != nil {
		t.Fatal("kpartx", err, string(kpartxOut))
	}
	kpartxOut, err = exec.Command(
		"kpartx", "-l", imageFile).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(kpartxOut))
	}
	return strings.Trim(strings.Split(string(kpartxOut), " ")[4], "/dev/")
}

func mountRootPartition(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.Label != "ROOT" {
			continue
		}
		mountOut, err := exec.Command(
			"mount", partition.Device,
			partition.MountPath).CombinedOutput()
		if err != nil {
			t.Fatal("mount", err, string(mountOut))
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
			"mount", partition.Device,
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
	out, err := exec.Command("uuidgen").CombinedOutput()
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
			err := os.MkdirAll(filepath.Join(
				partition.MountPath, file.Path), 0644)
			if err != nil {
				t.Fatal("mkdirall", err)
			}
			f, err := os.Create(filepath.Join(
				partition.MountPath, file.Path, file.Name))
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

		umountOut, err := exec.Command(
			"umount", partition.Device).CombinedOutput()
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
			"umount", partition.Device).CombinedOutput()
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
	t *testing.T, expected []*Partition, imageFile string) {
	for _, e := range expected {
		if e.TypeCode == "blank" {
			continue
		}
		sgdiskInfo, err := exec.Command(
			"sgdisk", "-i", strconv.Itoa(e.Number),
			imageFile).CombinedOutput()
		if err != nil {
			t.Error("sgdisk -i", strconv.Itoa(e.Number), err)
			return
		}
		lines := strings.Split(string(sgdiskInfo), "\n")
		actualTypeGUID := strings.ToUpper(strings.TrimSpace(
			strings.Split(strings.Split(lines[0], ": ")[1], " ")[0]))
		actualSectors := strings.Split(strings.Split(lines[4], ": ")[1], " ")[0]
		actualLabel := strings.Split(strings.Split(lines[6], ": ")[1], "'")[1]

		// have to align the size to the nearest sector first
		expectedSectors := align(e.Length, 512)

		if e.TypeGUID != actualTypeGUID {
			t.Error("TypeGUID does not match!", e.TypeGUID, actualTypeGUID)
		}
		if e.Label != actualLabel {
			t.Error("Label does not match!", e.Label, actualLabel)
		}
		if strconv.Itoa(expectedSectors) != actualSectors {
			t.Error(
				"Sectors does not match!", expectedSectors, actualSectors)
		}

		if e.FilesystemType == "" {
			continue
		}

		df, err := exec.Command("df", "-T", e.Device).CombinedOutput()
		if err != nil {
			t.Error("df -T", err, string(df))
			return
		}

		lines = strings.Split(string(df), "\n")
		if len(lines) < 2 {
			t.Error("Couldn't verify FilesystemType")
			return
		}
		actualFilesystemType := removeEmpty(strings.Split(lines[1], " "))[1]

		if e.FilesystemType != actualFilesystemType {
			t.Error("FilesystemType does not match!", e.Label,
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
			path := filepath.Join(partition.MountPath, file.Path, file.Name)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Error("File doesn't exist!", path)
				return
			}

			if file.Contents != nil {
				expectedContents := strings.Join(file.Contents, "\n")
				dat, err := ioutil.ReadFile(path)
				if err != nil {
					t.Error("Error when reading file", path)
					return
				}

				actualContents := string(dat)
				if expectedContents != actualContents {
					t.Error("Contents of file", path, "do not match!",
						expectedContents, actualContents)
				}
			}

			if file.Mode != "" {
				sout, err := exec.Command(
					"stat", "-c", "%a", path).CombinedOutput()
				statOut := strings.TrimSpace(string(sout))
				if err != nil {
					t.Error("Error running stat on file", err)
					return
				}
				if file.Mode != statOut {
					t.Error(
						"File Mode does not match", path, file.Mode, statOut)
				}
			}
		}
	}
}
