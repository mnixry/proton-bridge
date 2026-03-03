// Copyright (c) 2026 Proton AG
//
// This file is part of Proton Mail Bridge.
//
// Proton Mail Bridge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Proton Mail Bridge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Proton Mail Bridge. If not, see <https://www.gnu.org/licenses/>.

// Package message contains set of tools to convert message between Proton API
// and IMAP format.
package message

import (
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("pkg", "pkg/message") //nolint:gochecknoglobals

var SplitHeaderBodyV2Disabled SplitHeaderBodyV2DisabledWrapper //nolint:gochecknoglobals

type SplitHeaderBodyV2DisabledWrapper struct {
	value atomic.Bool
}

func (s *SplitHeaderBodyV2DisabledWrapper) Load() bool {
	return s.value.Load()
}

func (s *SplitHeaderBodyV2DisabledWrapper) Swap(newVal bool) {
	old := s.value.Swap(newVal)
	if old != newVal {
		log.WithFields(logrus.Fields{
			"old": old,
			"new": newVal,
		}).Warn("SplitHeaderBodyV2Disabled value has been changed")
	}
}
