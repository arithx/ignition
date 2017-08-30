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
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, RootOnRaid())
}

func RootOnRaid() types.Test {
	name := "Root on Raid"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	mntDevices := []types.MntDevice{
		{
			Label:        "ROOT",
			Substitution: "$DEVICE",
		},
	}
	config := `{
	  "ignition": {
	    "version": "2.1.0",
	    "config": {}
	  },
	  "storage": {
	    "disks": [
	      {
	        "device": "$blackbox_ignition_secondary_disk.img",
	        "partitions": [
	          {
	            "label": "root1",
	            "number": 1,
	            "size": 524288,
	            "start": 0,
	            "typeGuid": "be9067b9-ea49-4f15-b4f6-f36f8c9e1818"
	          },
	          {
	            "label": "root2",
	            "number": 2,
	            "size": 524288,
	            "start": 0,
	            "typeGuid": "be9067b9-ea49-4f15-b4f6-f36f8c9e1818"
	          }
	        ]
	      }
	    ],
	    "raid": [
	      {
	        "name": "rootarray",
	        "level": "raid1",
	        "devices": [
	          "$blackbox_ignition_secondary_disk.imgp1",
	          "$blackbox_ignition_secondary_disk.imgp2"
	        ]
	      }
	    ],
	    "filesystems": [
	      {
	        "name": "ROOT",
	        "mount": {
	          "device": "/dev/md/rootarray",
	          "format": "ext4",
	          "create": {
	            "options": [
	              "-L",
	              "ROOT"
	            ]
	          }
	        }
	      },
	      {
	        "name": "NOT_ROOT",
	        "mount": {
	          "device": "$DEVICE",
	          "format": "ext4",
	          "create": {
	            "force": true,
	            "options": [
	              "-L",
	              "wasteland"
	            ]
	          }
	        }
	      }
	    ]
	  },
	  "systemd": {},
	  "networkd": {},
	  "passwd": {}
	}`
	in = append(in, types.Disk{
		ImageFile: "blackbox_ignition_secondary_disk.img",
		Partitions: types.Partitions{
			{
				Label:    "important-data",
				Number:   1,
				Length:   65536,
				TypeGUID: "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
				GUID:     "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
			},
			{
				Label:    "ephemeral-data",
				Number:   2,
				Length:   131072,
				TypeGUID: "CA7D7CCB-63ED-4C53-861C-1742536059CC",
				GUID:     "B921B045-1DF0-41C3-AF44-4C6F280D3FAE",
			},
		},
	})
	out[0].Partitions.GetPartition("ROOT").Label = "wasteland"
	out = append(out, types.Disk{
		ImageFile: "blackbox_ignition_secondary_disk.img",
		Partitions: types.Partitions{
			{
				Label:    "root1",
				Number:   1,
				Length:   65536,
				TypeGUID: "BE9067B9-EA49-4F15-B4F6-F36F8C9E1818",
			},
			{
				Label:    "root2",
				Number:   2,
				Length:   131072,
				TypeGUID: "BE9067B9-EA49-4F15-B4F6-F36F8C9E1818",
			},
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}
