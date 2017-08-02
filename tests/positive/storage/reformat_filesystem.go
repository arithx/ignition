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

package storage

import (
	"github.com/coreos/ignition/tests/types"
)

func ReformatToBTRFS() types.Test {
	name := "Reformat a Filesystem to Btrfs"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
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
	out[0].Partitions.GetPartition("OEM").FilesystemType = "btrfs"

	return types.Test{name, in, out, mntDevices, config}
}

func ReformatToXFS() types.Test {
	name := "Reformat a Filesystem to XFS"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "OEM",
			Substitution: "$DEVICE",
		},
	}
	config := `{
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
	out[0].Partitions.GetPartition("OEM").FilesystemType = "xfs"

	return types.Test{name, in, out, mntDevices, config}
}
