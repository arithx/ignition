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

package files

import (
	"github.com/coreos/ignition/tests/register"
	"github.com/coreos/ignition/tests/types"
)

func init() {
	register.Register(register.PositiveTest, CreateDirectoryOnRoot())
	register.Register(register.PositiveTest, NewDirUserGroupByID_2_1_0())
	register.Register(register.PositiveTest, ExistingDirUserGroupByID_2_1_0())
	register.Register(register.PositiveTest, NewDirUserGroupByName_2_1_0())
	register.Register(register.PositiveTest, ExistingDirUserGroupByName_2_1_0())
}

func CreateDirectoryOnRoot() types.Test {
	name := "Create a Directory on the Root Dirsystem"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "directories": [{
	      "filesystem": "root",
	      "path": "/foo/bar"
	    }]
	  }
	}`
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func NewDirUserGroupByID_2_1_0() types.Test {
	name := "New Directory - 2.1.0 User/Group by id"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "directories": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "user": {"id": 500},
		  "group": {"id": 500}
	    }]
	  }
	}`
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func NewDirUserGroupByName_2_1_0() types.Test {
	name := "New Directory - 2.1.0 User/Group by name"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "directories": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "user": {"name": "core"},
		  "group": {"name": "core"}
	    }]
	  }
	}`
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func ExistingDirUserGroupByID_2_1_0() types.Test {
	name := "Existing Directory - 2.1.0 User/Group by id"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "directories": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "user": {"id": 500},
		  "group": {"id": 500}
	    }]
	  }
	}`
	in[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      0,
				Group:     0,
			},
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func ExistingDirUserGroupByName_2_1_0() types.Test {
	name := "Existing Directory - 2.1.0 User/Group by name"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "directories": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "user": {"name": "core"},
		  "group": {"name": "core"}
	    }]
	  }
	}`
	in[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      0,
				Group:     0,
			},
		},
	})
	out[0].Partitions.AddDirectories("ROOT", []types.Directory{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}
