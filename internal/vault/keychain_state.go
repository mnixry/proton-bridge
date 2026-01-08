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

package vault

import (
	"runtime"

	"github.com/ProtonMail/proton-bridge/v3/internal/platform"
	"github.com/ProtonMail/proton-bridge/v3/internal/vault/storage"
)

const keychainStateFileName = "keychain_state.json"

type KeychainState struct {
	FailedAttempts int
}

var keychainStateFile = storage.NewJSONStorageFile[KeychainState](keychainStateFileName, "keychain state") //nolint:gochecknoglobals

func LoadKeychainState(vaultDir string) (KeychainState, error) {
	if runtime.GOOS != platform.LINUX {
		return KeychainState{}, nil
	}
	return keychainStateFile.Load(vaultDir)
}

func (k KeychainState) Save(vaultDir string) error {
	if runtime.GOOS != platform.LINUX {
		return nil
	}

	return keychainStateFile.Save(vaultDir, k)
}

func (k KeychainState) ResetAndSave(vaultDir string) error {
	k.FailedAttempts = 0
	return k.Save(vaultDir)
}
