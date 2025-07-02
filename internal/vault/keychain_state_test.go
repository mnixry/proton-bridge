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

package vault

import (
	"runtime"
	"testing"

	"github.com/ProtonMail/proton-bridge/v3/internal/platform"
	"github.com/stretchr/testify/require"
)

func TestKeychainState(t *testing.T) {
	dir := t.TempDir()

	// Load a non-existing keychain state file. It should return the defaults if it does not exist and no error will be thrown.
	keychainState, err := LoadKeychainState(dir)
	require.NoError(t, err)
	require.Equal(t, KeychainState{}, keychainState)

	// Increment the failed attempt count. The function call will save the data to the file.
	err = IncrementKeychainFailedAttemptCount(dir)
	require.NoError(t, err)

	// Load the state from the now existing file. We isolate the behaviour of the helper to Linux.
	// Thus, a nil state is expected on other OS'.
	keychainState, err = LoadKeychainState(dir)
	require.NoError(t, err)
	if runtime.GOOS == platform.LINUX {
		require.Equal(t, KeychainState{
			FailedAttempts: 1,
		}, keychainState)
	} else {
		require.Equal(t, KeychainState{}, keychainState)
	}

	// Increment again.
	err = IncrementKeychainFailedAttemptCount(dir)
	require.NoError(t, err)

	// Same thing, we only expect linux to have data.
	keychainState, err = LoadKeychainState(dir)
	require.NoError(t, err)
	if runtime.GOOS == platform.LINUX {
		require.Equal(t, KeychainState{
			FailedAttempts: 2,
		}, keychainState)
	} else {
		require.Equal(t, KeychainState{}, keychainState)
	}

	// Reset the failed attempt count.
	err = ResetFailedKeychainAttemptCount(dir)
	require.NoError(t, err)

	// All OS' states should match in this case.
	keychainState, err = LoadKeychainState(dir)
	require.NoError(t, err)
	require.Equal(t, KeychainState{}, keychainState)
}
