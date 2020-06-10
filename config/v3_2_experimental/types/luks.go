// Copyright 2020 Red Hat, Inc.
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

package types

import (
	"github.com/coreos/ignition/v2/config/shared/errors"
	"github.com/coreos/ignition/v2/config/util"

	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

func (l Luks) Key() string {
	return l.Name
}

func (l Luks) IgnoreDuplicates() map[string]struct{} {
	return map[string]struct{}{
		"Options": {},
	}
}

func (l Luks) Validate(c path.ContextPath) (r report.Report) {
	r.AddOnError(c.Append("label"), l.validateLabel())
	r.AddOnError(c.Append("device"), validatePath(l.Device))

	if util.NilOrEmpty(l.KeyFile) && l.Clevis == nil {
		r.AddOnError(c.Append("keys"), errors.ErrInvalidLuksVolume)
	}
	return
}

func (l Luks) validateLabel() error {
	if util.NilOrEmpty(l.Label) {
		return nil
	}

	if len(*l.Label) > 16 {
		// cryptsetup does not specify a limit on label size
		return errors.ErrLuksLabelTooLong
	}

	return nil
}
