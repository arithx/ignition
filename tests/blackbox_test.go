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
	"strconv"
	"strings"
	"testing"

	yaml "github.com/go-yaml/yaml"
)

type File struct {
	Name     string   `yaml:"name"`
	Path     string   `yaml:"path"`
	Contents []string `yaml:"contents,omitempty"`
}

type Partition struct {
	Number         int    `yaml:"number"`
	Label          string `yaml:"label"`
	TypeCode       string `yaml:"typecode,omitempty"`
	TypeGUID       string `yaml:"typeguid,omitempty"`
	GUID           string `yaml:"guid,omitempty"`
	Device         string `yaml:"device,omitempty"`
	Offset         int    `yaml:"offset,omitempty"`
	Length         int    `yaml:"length"`
	FilesystemType string `yaml:"filesystemtype"`
	MountPath      string `yaml:"mountpath,omitempty"`
	Hybrid         bool   `yaml:"hybrid,omitempty"`
	Files          []File `yaml:"files"`
}

func TestIgnitionBlackBox(t *testing.T) {
	tests := []struct {
		in, out, config string
	}{
		{
			in:     "data/disk.yaml",
			out:    "data/diskOut.yaml",
			config: "data/config.ign",
		},
	}

	for _, test := range tests {
		outer(t, test.in, test.out, test.config)
	}
}

func outer(t *testing.T, inPath string, outPath string, config string) {
	imgName := "test.img"
	in := parseYAML(t, inPath, false)
	out := parseYAML(t, outPath, true)
	imageSize := calculateImageSize(in)

	// Creation
	createVolume(t, imgName, imageSize, 20, 16, 63, in)
	setDevices(t, imgName, in)
	mountPartitions(t, in)
	createFiles(t, in)
	unmountPartitions(t, in, imgName)

	// Ignition
	device := pickDevice(t, in, imgName)
	t.Log("Loop Device:", device)
	updateIgnitionConfig(t, config, device)
	//runIgnition(ignition, "disk")
	runIgnition(t, "files")

	// Update out structure with mount points & devices
	setExpectedPartitionsDrive(in, out)

	// Validation
	mountPartitions(t, out)
	dumpDiskInfo(t, imgName, out)
	validatePartitions(t, out, imgName)
	validateFiles(t, out)
}

func runIgnition(t *testing.T, stage string) {
	out, err := exec.Command(
		"../bin/amd64/ignition", "-clear-cache", "-oem",
		"file", "-stage", stage).CombinedOutput()
	if err != nil {
		t.Fatal("ignition", err, string(out))
	}

}

func pickDevice(t *testing.T, partitions []*Partition, fileName string) string {
	number := -1
	for _, p := range partitions {
		if p.Label == "ROOT" {
			number = p.Number
		}
	}
	if number == -1 {
		t.Fatal("Didn't find a ROOT drive")
		return ""
	}

	kpartxOut, err := exec.Command("kpartx", "-l", fileName).CombinedOutput()
	if err != nil {
		t.Fatal("kpartx -l", err, string(kpartxOut))
	}
	t.Log(string(kpartxOut))
	return fmt.Sprintf("/dev/mapper/%sp%d",
		strings.Trim(strings.Split(string(kpartxOut), " ")[4], "/dev/"), number)
}

func updateIgnitionConfig(t *testing.T, config, device string) {
	input, err := ioutil.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	data := string(input)
	data = strings.Replace(data, "$DEVICE", device, -1)
	err = ioutil.WriteFile("config.ign", []byte(data), 0644)
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

func parseYAML(t *testing.T, fileName string, out bool) []*Partition {
	dat, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatal(err, string(dat))
	}
	p := []*Partition{}
	err = yaml.Unmarshal(dat, &p)

	for _, part := range p {
		if !out {
			if part.GUID == "" {
				part.GUID = generateUUID(t)
			}
		}
		updateTypeGUID(t, part)
	}

	setOffsets(p)

	return p
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

		mntPath := fmt.Sprintf("%s%s%d", "/mnt/", "hd1p", counter)
		err := os.Mkdir(mntPath, 0644)
		if err != nil {
			t.Fatal("mkdir", err)
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
	t.Log("/sbin/sgdisk", strings.Join(opts, " "))
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
	return strings.Trim(strings.Split(string(kpartxOut), " ")[7], "/dev/")
}

func mountPartitions(t *testing.T, partitions []*Partition) {
	for _, partition := range partitions {
		if partition.FilesystemType == "" {
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

func unmountPartitions(t *testing.T, partitions []*Partition, fileName string) {
	for _, partition := range partitions {
		if partition.FilesystemType == "" {
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
		t.Log(e.Device, (string(df)))
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
					t.Fatal("Error when reading file ", path)
				}

				actualContents := string(dat)
				if expectedContents != actualContents {
					t.Fatal("Contents of file ", path, "do not match!",
						expectedContents, actualContents)
				}
			}
		}
	}
}
