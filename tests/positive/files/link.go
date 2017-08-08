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
	register.Register(register.PositiveTest, CreateHardLinkOnRoot())
	register.Register(register.PositiveTest, CreateSymlinkOnRoot())
	register.Register(register.PositiveTest, NewLinkUserGroupByID_2_1_0())
	register.Register(register.PositiveTest, ExistingLinkUserGroupByID_2_1_0())
	register.Register(register.PositiveTest, NewLinkUserGroupByName_2_1_0())
	register.Register(register.PositiveTest, ExistingLinkUserGroupByName_2_1_0())
}

func CreateHardLinkOnRoot() types.Test {
	name := "Create a Hard Link on the Root Filesystem"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "files": [{
	      "filesystem": "root",
	      "path": "/foo/target",
	      "contents": {
	        "source": "http://127.0.0.1:8080/contents"
	      }
	    }],
	    "links": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "target": "/foo/target",
		  "hard": true
	    }]
	  }
	}`
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "target",
			},
			Contents: "asdf\nfdsa",
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Directory: "foo",
				Name:      "bar",
			},
			Target: "/foo/target",
			Hard:   true,
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func CreateSymlinkOnRoot() types.Test {
	name := "Create a Symlink on the Root Filesystem"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "links": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
	      "target": "/foo/target",
	      "hard": false
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func NewLinkUserGroupByID_2_1_0() types.Test {
	name := "New Link - 2.1.0 User/Group by id"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "links": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "target": "/foo/target",
		  "user": {"id": 500},
		  "group": {"id": 500},
		  "hard": false
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      0,
				Group:     0,
			},
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func NewLinkUserGroupByName_2_1_0() types.Test {
	name := "New Link - 2.1.0 User/Group by name"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "links": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "target": "/foo/target",
		  "user": {"name": "core"},
		  "group": {"name": "core"},
		  "hard": false
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      0,
				Group:     0,
			},
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func ExistingLinkUserGroupByID_2_1_0() types.Test {
	name := "Existing Link - 2.1.0 User/Group by id"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "links": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "user": {"id": 500},
		  "group": {"id": 500}
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      0,
				Group:     0,
			},
		},
	})
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}

func ExistingLinkUserGroupByName_2_1_0() types.Test {
	name := "Existing Link - 2.1.0 User/Group by name"
	in := types.GetBaseDisk()
	out := types.GetBaseDisk()
	var mntDevices []types.MntDevice
	config := `{
	  "ignition": { "version": "2.1.0" },
	  "storage": {
	    "links": [{
	      "filesystem": "root",
	      "path": "/foo/bar",
		  "user": {"name": "core"},
		  "group": {"name": "core"}
	    }]
	  }
	}`
	in[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      0,
				Group:     0,
			},
		},
	})
	in[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})
	out[0].Partitions.AddFiles("ROOT", []types.File{
		{
			Node: types.Node{
				Name:      "target",
				Directory: "foo",
				User:      500,
				Group:     500,
			},
		},
	})
	out[0].Partitions.AddLinks("ROOT", []types.Link{
		{
			Node: types.Node{
				Name:      "bar",
				Directory: "foo",
			},
			Target: "/foo/target",
			Hard:   false,
		},
	})

	return types.Test{name, in, out, mntDevices, config}
}
