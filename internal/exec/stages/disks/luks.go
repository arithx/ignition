// Copyright 2020 Red Hat
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

// The storage stage is responsible for partitioning disks, creating RAID
// arrays, formatting partitions, writing files, writing systemd units, and
// writing network units.
// createRaids creates the raid arrays described in config.Storage.Raid.

package disks

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/ignition/v2/config/v3_2_experimental/types"
	"github.com/coreos/ignition/v2/internal/distro"
	execUtil "github.com/coreos/ignition/v2/internal/exec/util"
)

type Tang struct {
	URL        string `json:"url"`
	Thumbprint string `json:"thp,omitempty"`
}

type Pin struct {
	Tpm  bool   `json:"tpm"`
	Tang []Tang `json:"tang,omitempty"`
}

func (p Pin) MarshalJSON() ([]byte, error) {
	if p.Tpm {
		return json.Marshal(&struct {
			Tang []Tang   `json:"tang,omitempty"`
			Tpm  struct{} `json:"tpm2"`
		}{
			Tang: p.Tang,
			Tpm:  struct{}{},
		})
	}
	return json.Marshal(&struct {
		Tang []Tang `json:"tang"`
	}{
		Tang: p.Tang,
	})
}

type Clevis struct {
	Pins      Pin `json:"pins"`
	Threshold int `json:"t"`
}

// Initially tested generating keyfiles via dd'ing to a file from /dev/urandom
// however while cryptsetup had no problem with these keyfiles clevis seemed to
// die on them while keyfiles generated via openssl rand -hex would work...
func randHex(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *stage) createLuks(config types.Config) error {
	if len(config.Storage.Luks) == 0 {
		return nil
	}

	for _, luks := range config.Storage.Luks {
		// track whether Ignition creates the KeyFile
		// so that it can be removed creation
		var ignitionCreatedKeyFile bool
		// create keyfile inside of tmpfs, it will be copied to the
		// sysroot by ignition-copy-keyfiles
		os.MkdirAll(execUtil.LuksInitramfsKeyFilePath, 0644)
		keyFilePath := filepath.Join(execUtil.LuksInitramfsKeyFilePath, luks.Name)
		if luks.KeyFile == nil || *luks.KeyFile == "" {
			// create a keyfile
			key, err := randHex(4096)
			if err != nil {
				return fmt.Errorf("generating keyfile: %v", err)
			}
			if err := ioutil.WriteFile(keyFilePath, []byte(key), 0400); err != nil {
				return fmt.Errorf("creating keyfile: %v", err)
			}
			ignitionCreatedKeyFile = true
		} else {
			if err := ioutil.WriteFile(keyFilePath, []byte(*luks.KeyFile), 0400); err != nil {
				return fmt.Errorf("writing keyfile: %v", err)
			}
		}

		args := []string{
			"luksFormat",
			"--type", "luks2",
			"--key-file", keyFilePath,
		}

		if !util.NilOrEmpty(luks.Hash) {
			args = append(args, "--hash", *luks.Hash)
		}

		if !util.NilOrEmpty(luks.Label) {
			args = append(args, "--label", *luks.Label)
		}

		if !util.NilOrEmpty(luks.UUID) {
			args = append(args, "--uuid", *luks.UUID)
		}

		if !util.NilOrEmpty(luks.Cipher) {
			args = append(args, "--cipher", *luks.Cipher)
		}

		if len(luks.Options) > 0 {
			// golang's a really great language...
			for _, option := range luks.Options {
				args = append(args, string(option))
			}
		}

		args = append(args, luks.Device)

		if _, err := s.Logger.LogCmd(
			exec.Command(distro.CryptsetupCmd(), args...),
			"creating %q", luks.Name,
		); err != nil {
			return fmt.Errorf("cryptsetup failed: %v", err)
		}

		// open the device
		if _, err := s.Logger.LogCmd(
			exec.Command(distro.CryptsetupCmd(), "luksOpen", luks.Device, luks.Name, "--key-file", keyFilePath),
			"opening luks device %v", luks.Name,
		); err != nil {
			return fmt.Errorf("opening luks device: %v", err)
		}

		if luks.Clevis != nil {
			c := Clevis{
				Pins: Pin{},
			}
			if luks.Clevis.Threshold == nil {
				c.Threshold = 1
			} else {
				c.Threshold = *luks.Clevis.Threshold
			}
			for _, tang := range luks.Clevis.Tang {
				c.Pins.Tang = append(c.Pins.Tang, Tang{
					URL:        tang.URL,
					Thumbprint: tang.Thumbprint,
				})
			}
			if luks.Clevis.Tpm2 != nil {
				c.Pins.Tpm = *luks.Clevis.Tpm2
			}
			clevisJson, err := json.Marshal(c)
			if err != nil {
				return fmt.Errorf("creating clevis json: %v", err)
			}
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.ClevisCmd(), "luks", "bind", "-f", "-k", keyFilePath, "-d", luks.Device, "sss", string(clevisJson)), "Clevis bind",
			); err != nil {
				return fmt.Errorf("binding clevis device: %v", err)
			}

			// close & re-open Clevis devices to make sure that we can unlock them
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.CryptsetupCmd(), "luksClose", name),
				"closing luks device %v", name,
			); err != nil {
				return fmt.Errorf("closing luks device: %v", err)
			}
		}

		// assume the user does not want a key file, remove it
		if ignitionCreatedKeyFile {
			if _, err := s.Logger.LogCmd(
				exec.Command(distro.CryptsetupCmd(), "luksRemoveKey", luks.Device, keyFilePath),
				"removing key file for %v", luks.Name,
			); err != nil {
				return fmt.Errorf("removing key file: %v", err)
			}
			os.Remove(keyFilePath)
		}
	}

	return nil
}
