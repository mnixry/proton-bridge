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

package message

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/ProtonMail/proton-bridge/v3/utils"
	"github.com/stretchr/testify/require"
)

// TestBoundaryIsValidHex verifies that newBoundary() produces a 64-character lowercase hex string.
func TestBoundaryIsValidHex(t *testing.T) {
	val, err := newBoundary()

	require.NoError(t, err)
	require.Len(t, val, 64)
	decoded, err := hex.DecodeString(val)
	require.NoError(t, err)
	require.Len(t, decoded, 32)
}

// TestBuildMultipartBoundariesAreUnique verifies that two builds of the same message
// produce different MIME boundaries.
func TestBuildMultipartBoundariesAreUnique(t *testing.T) {
	kr := utils.MakeKeyRing(t)
	msg := newTestMessage(t, kr, "messageID", "addressID", "text/plain", "body", time.Now())
	att := addTestAttachment(t, kr, &msg, "attID", "file.txt", "text/plain", "attachment", "content")

	res1, err := DecryptAndBuildRFC822(kr, msg, [][]byte{att}, JobOptions{})
	require.NoError(t, err)

	res2, err := DecryptAndBuildRFC822(kr, msg, [][]byte{att}, JobOptions{})
	require.NoError(t, err)
	require.NotEqual(t, res1, res2, "two builds should produce different boundaries")
}
