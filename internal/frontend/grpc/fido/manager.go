// Copyright (c) 2025 Proton AG
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

package fido

import (
	"context"
	"sync"
)

type Manager struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (fm *Manager) withLock(fn func()) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fn()
}

func (fm *Manager) SetCancel(cancel context.CancelFunc) {
	fm.withLock(func() {
		fm.cancel = cancel
	})
}

func (fm *Manager) Cancel() {
	fm.withLock(func() {
		if fm.cancel != nil {
			fm.cancel()
			fm.cancel = nil
		}
	})
}

func (fm *Manager) Clear() {
	fm.withLock(func() {
		fm.cancel = nil
	})
}
