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
// along with Proton Mail Bridge.  If not, see <https://www.gnu.org/licenses/>.

package dialer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicTLSDialer_ShouldSkipCertificateChainVerification(t *testing.T) {
	tests := []struct {
		hostURL  string
		address  string
		expected bool
	}{
		{
			hostURL:  "https://mail-api.proton.me",
			address:  "mail-api.proton.me:443",
			expected: false,
		},
		{
			hostURL:  "https://proton.me",
			address:  "proton.me",
			expected: false,
		},
		{
			hostURL:  "https://api.proton.me",
			address:  "mail.proton.me:443",
			expected: false,
		},
		{
			hostURL:  "https://proton.me",
			address:  "mail-api.proton.me:443",
			expected: false,
		},
		{
			hostURL:  "https://mail-api.proton.me",
			address:  "proton.me:443",
			expected: false,
		},
		{
			hostURL:  "https://mail.google.com",
			address:  "mail-api.proton.me:443",
			expected: true,
		},
		{
			hostURL:  "https://mail-api.protonmail.com",
			address:  "mail-api.proton.me:443",
			expected: true,
		},
		{
			hostURL:  "https://proton.me",
			address:  "google.com:443",
			expected: true,
		},
		{
			hostURL:  "https://proton.me",
			address:  "proton.com:443",
			expected: true,
		},
		{
			hostURL:  "https://proton.me",
			address:  "example.me:443",
			expected: true,
		},
		{
			hostURL:  "https://proton.me",
			address:  "mail.example.com:443",
			expected: true,
		},
		{
			hostURL:  "https://proton.me",
			address:  "proton.me",
			expected: false,
		},
		{
			hostURL:  "https://proton.me:8080",
			address:  "proton.me:443",
			expected: true,
		},
		{
			hostURL:  "https://proton.me/api/v1",
			address:  "proton.me:443",
			expected: false,
		},
		{
			hostURL:  "https://proton.black",
			address:  "mail-api.pascal.proton.black",
			expected: false,
		},
		{
			hostURL:  "https://mail-api.pascal.proton.black",
			address:  "mail-api.pascal.proton.black",
			expected: false,
		},
		{
			hostURL:  "https://mail-api.pascal.proton.black",
			address:  "proton.black:332",
			expected: false,
		},
		{
			hostURL:  "https://mail-api.pascal.proton.black",
			address:  "proton.me",
			expected: true,
		},
		{
			hostURL:  "https://mail-api.pascal.proton.black",
			address:  "proton.me:332",
			expected: true,
		},
	}

	for _, tt := range tests {
		dialer := NewBasicTLSDialer(tt.hostURL)
		result := dialer.ShouldSkipCertificateChainVerification(tt.address)
		require.Equal(t, tt.expected, result)
	}
}
