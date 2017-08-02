// Copyright 2017 CoreOS, pInc.
//
// Licensed under the Apache License, pVersion 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, psoftware
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, peither express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blackbox

import (
	nfiles "github.com/coreos/ignition/tests/negative/files"
	ngeneral "github.com/coreos/ignition/tests/negative/general"
	nregression "github.com/coreos/ignition/tests/negative/regression"
	nstorage "github.com/coreos/ignition/tests/negative/storage"
	ntimeouts "github.com/coreos/ignition/tests/negative/timeouts"
	pfiles "github.com/coreos/ignition/tests/positive/files"
	pgeneral "github.com/coreos/ignition/tests/positive/general"
	pnetworkd "github.com/coreos/ignition/tests/positive/networkd"
	ppasswd "github.com/coreos/ignition/tests/positive/passwd"
	pregression "github.com/coreos/ignition/tests/positive/regression"
	pstorage "github.com/coreos/ignition/tests/positive/storage"
	psystemd "github.com/coreos/ignition/tests/positive/systemd"
	ptimeouts "github.com/coreos/ignition/tests/positive/timeouts"
	"github.com/coreos/ignition/tests/types"
)

func createNegativeTests() []types.Test {
	tests := []types.Test{}

	tests = append(tests, nfiles.InvalidHash())
	tests = append(tests, nfiles.InvalidHashFromHTTPURL())
	tests = append(tests, ngeneral.ReplaceConfigWithInvalidHash())
	tests = append(tests, ngeneral.AppendConfigWithInvalidHash())
	tests = append(tests, ngeneral.InvalidVersion())
	tests = append(tests, nregression.VFATIgnoresWipeFilesystem())
	tests = append(tests, nstorage.InvalidFilesystem())
	tests = append(tests, nstorage.NoDevice())
	tests = append(tests, nstorage.NoDeviceWithForce())
	tests = append(tests, nstorage.NoDeviceWithWipeFilesystemTrue())
	tests = append(tests, nstorage.NoDeviceWithWipeFilesystemFalse())
	tests = append(tests, nstorage.NoFilesystemType())
	tests = append(tests, nstorage.NoFilesystemTypeWithForce())
	tests = append(tests, nstorage.NoFilesystemTypeWithWipeFilesystem())
	tests = append(tests, ntimeouts.DecreaseHTTPResponseHeadersTimeout())

	return tests
}

func createTests() []types.Test {
	tests := []types.Test{}

	tests = append(tests, pfiles.CreateDirectoryOnRoot())
	tests = append(tests, pfiles.CreateFileOnRoot())
	tests = append(tests, pfiles.UserGroupByID_2_0_0())
	tests = append(tests, pfiles.UserGroupByID_2_1_0())
	// TODO: Investigate why ignition's C code hates our environment
	// tests = append(tests, pfiles.UserGroupByName_2_1_0())
	tests = append(tests, pfiles.ValidateFileHashFromDataURL())
	tests = append(tests, pfiles.ValidateFileHashFromHTTPURL())
	tests = append(tests, pfiles.CreateHardLinkOnRoot())
	tests = append(tests, pfiles.CreateSymlinkOnRoot())
	tests = append(tests, pfiles.CreateFileFromRemoteContents())
	tests = append(tests, pgeneral.ReformatFilesystemAndWriteFile())
	tests = append(tests, pgeneral.SetHostname())
	tests = append(tests, pgeneral.ReplaceConfigWithRemoteConfig())
	tests = append(tests, pgeneral.AppendConfigWithRemoteConfig())
	tests = append(tests, pgeneral.VersionOnlyConfig())
	tests = append(tests, pgeneral.EmptyUserdata())
	tests = append(tests, pnetworkd.CreateNetworkdUnit())
	tests = append(tests, ppasswd.AddPasswdUsers())
	tests = append(tests, pregression.EquivalentFilesystemUUIDsTreatedDistinctEXT4())
	tests = append(tests, pregression.EquivalentFilesystemUUIDsTreatedDistinctVFAT())
	tests = append(tests, pstorage.ForceNewFilesystemOfSameType())
	tests = append(tests, pstorage.WipeFilesystemWithSameType())
	tests = append(tests, pstorage.CreateNewPartitions())
	tests = append(tests, pstorage.ReuseExistingFilesystem())
	tests = append(tests, pstorage.ReformatToBTRFS())
	tests = append(tests, pstorage.ReformatToXFS())
	tests = append(tests, psystemd.CreateSystemdService())
	tests = append(tests, psystemd.ModifySystemdService())
	tests = append(tests, psystemd.MaskSystemdServices())
	tests = append(tests, ptimeouts.IncreaseHTTPResponseHeadersTimeout())
	tests = append(tests, ptimeouts.ConfirmHTTPBackoffWorks())

	return tests
}
