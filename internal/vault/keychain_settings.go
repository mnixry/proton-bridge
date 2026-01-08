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

import "github.com/ProtonMail/proton-bridge/v3/internal/vault/storage"

const keychainSettingsFileName = "keychain.json"

// KeychainSettings holds settings related to the keychain. It is serialized in the vault directory.
type KeychainSettings struct {
	Helper      string // The helper used for keychain.
	DisableTest bool   // Is the keychain test on startup disabled?
}

var keychainSettingsFile = storage.NewJSONStorageFile[KeychainSettings](keychainSettingsFileName, "keychain settings") //nolint:gochecknoglobals

// LoadKeychainSettings load keychain settings from the vaultDir folder, or returns a default one if the file
// does not exists or is invalid.
func LoadKeychainSettings(vaultDir string) (KeychainSettings, error) {
	return keychainSettingsFile.Load(vaultDir)
}

// Save saves the keychain settings in a file in the vaultDir folder.
func (k KeychainSettings) Save(vaultDir string) error {
	return keychainSettingsFile.Save(vaultDir, k)
}
