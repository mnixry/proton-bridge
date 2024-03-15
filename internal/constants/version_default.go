// Copyright (c) 2026 Proton AG
//
// This file is part of Proton Mail Bridge.Bridge.
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

//go:build !build_qa
// +build !build_qa

package constants

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/sirupsen/logrus"
)

var versionCache string

// Get newest app version from proton web mail
func getNewestAppVersion() (string, error) {
	resp, err := http.Get("https://mail.proton.me/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	versionRegexp := regexp.MustCompile(`\?v=([0-9\.]+)`)
	versionMatches := versionRegexp.FindStringSubmatch(string(body))
	if len(versionMatches) < 2 {
		return "", errors.New("failed to find version")
	}
	return versionMatches[1], nil
}

// AppVersion returns the full rendered version of the app (to be used in request headers).
func AppVersion(version string) string {
	if versionCache == "" {
		newestVersion, err := getNewestAppVersion()
		if err != nil {
			logrus.WithError(err).Warn("Failed to get newest app version")
		} else {
			versionCache = newestVersion
		}
	}
	logrus.WithField("version", versionCache).Debug("App version")
	return fmt.Sprintf("%v@%v", AppName, versionCache)
}
