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
// along with Proton Mail Bridge.  If not, see <https://www.gnu.org/licenses/>.

//go:build build_qa || test_integration

package hv

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/ProtonMail/go-proton-api"
)

// parseAPIHost parses BRIDGE_API_HOST which is on the `account` subdomain & ends with /api
// It strips both of those parts and changes the subdomain to `verify`.
func parseAPIHost() (string, error) {
	host := os.Getenv("BRIDGE_API_HOST")

	if host == "" {
		return host, errors.New("invalid BRIDGE_API_HOST")
	}

	url, err := url.Parse(host)
	if err != nil {
		return "", errors.New("invalid BRIDGE_API_HOST")
	}

	host = url.Hostname()
	parts := strings.Split(host, ".")

	if len(parts) <= 2 {
		host = "https://verify" + "." + host
	} else {
		host = "https://verify" + "." + strings.Join(parts[1:], ".")
	}
	return host, nil
}

func FormatHvURL(details *proton.APIHVDetails) string {
	host, err := parseAPIHost()
	if err != nil {
		host = "https://verify.proton.me"
	}

	return fmt.Sprintf(
		"%v/?methods=%v&token=%v",
		host,
		strings.Join(details.Methods, ","),
		details.Token,
	)
}
