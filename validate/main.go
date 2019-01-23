// Copyright 2018 CoreOS, Inc.
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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	//config "github.com/coreos/ignition/config/v2_4_experimental"
	config "github.com/coreos/ignition/internal/config"
	//"github.com/coreos/ignition/internal/version"

	"github.com/spf13/cobra"
)

var (
	flagVersion bool
	rootCmd     = &cobra.Command{
		Use:   "ignition-validate config.ign",
		Short: "ignition-validate will validate Ignition configs",
		Run:   runIgnValidate,
	}
)

func main() {
	rootCmd.Flags().BoolVar(&flagVersion, "version", false, "print the version of ignition-validate")
	rootCmd.Execute()
}

func stdout(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, strings.TrimSpace(format)+"\n", a...)
}

func stderr(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.TrimSpace(format)+"\n", a...)
}

func die(format string, a ...interface{}) {
	stderr(format, a...)
	os.Exit(1)
}

func runIgnValidate(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		os.Exit(1)
	}

	c1, err := ioutil.ReadFile(args[0])
	if err != nil {
		stderr("reading config1: %v", err)
	}
	oc1, _, err := config.Parse(c1)
	if err != nil {
		stderr("parsing config1: %v", err)
	}
	c2, err := ioutil.ReadFile(args[1])
	if err != nil {
		stderr("reading config2: %v", err)
	}
	oc2, _, err := config.Parse(c2)
	if err != nil {
		stderr("parsing config2: %v", err)
	}
	c3 := config.Append(oc1, oc2)
	s, err := json.Marshal(c3)
	if err != nil {
		stderr("converting config3 to json: %v", err)
	}
	stdout(string(s))
}

/*
func runIgnValidate(cmd *cobra.Command, args []string) {
	if flagVersion {
		stdout(version.String)
		return
	}
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(1)
	}
	var blob []byte
	var err error
	if args[0] == "-" {
		blob, err = ioutil.ReadAll(os.Stdin)
	} else {
		blob, err = ioutil.ReadFile(args[0])
	}
	if err != nil {
		die("couldn't read config: %v", err)
	}
	_, rpt, err := config.Parse(blob)
	if len(rpt.Entries) > 0 {
		stdout(rpt.String())
	}
	if rpt.IsFatal() {
		os.Exit(1)
	}
	if err != nil {
		die("couldn't parse config: %v", err)
	}
}
*/
