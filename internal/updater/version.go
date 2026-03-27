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

package updater

import (
	"github.com/Masterminds/semver/v3"
	"github.com/ProtonMail/proton-bridge/v3/internal/updater/versioncompare"
)

type File struct {
	URL            string         `json:"Url"`
	Sha512CheckSum string         `json:"Sha512CheckSum,omitempty"`
	Identifier     FileIdentifier `json:"Identifier"`
}

type Release struct {
	ReleaseCategory   ReleaseCategory `json:"CategoryName"`
	Version           *semver.Version
	SystemVersion     versioncompare.SystemVersion `json:"SystemVersion"`
	RolloutProportion float64
	MinAuto           *semver.Version `json:"MinAuto,omitempty"`
	ReleaseNotesPage  string
	LandingPage       string
	File              []File `json:"File"`
}

func (rel Release) IsEmpty() bool {
	return rel.Version == nil && len(rel.File) == 0
}

type VersionInfo struct {
	Releases []Release `json:"Releases"`
}
