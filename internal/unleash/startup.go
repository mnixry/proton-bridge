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

package unleash

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/proton-bridge/v3/internal/constants"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const startupCacheFilename = "unleash_startup_flags.json"

var logger = logrus.WithField("pkg", "unleash-startup") //nolint:gochecknoglobals

type FeatureFlagStartupStore map[string]bool

func (f FeatureFlagStartupStore) GetFlagValue(key string) bool {
	val, ok := f[key]
	if !ok {
		return false
	}

	return val
}

func newAPIOptions(
	apiURL string,
	version *semver.Version,
) []proton.Option {
	return []proton.Option{
		proton.WithHostURL(apiURL),
		proton.WithAppVersion(constants.AppVersion(version.Original())),
		proton.WithLogger(logrus.WithField("pkg", "gpa/unleash-startup")),
		proton.WithRetryCount(0),
	}
}

func readStartupCacheFile(filepath string) (map[string]bool, error) {
	ffStore := make(map[string]bool)
	if filepath == "" {
		return ffStore, nil
	}

	file, err := os.Open(filepath) //nolint:gosec
	if err != nil {
		return ffStore, err
	}

	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			logger.WithError(err).Error("Unable to close cache file after read")
		}
	}(file)

	if err := json.NewDecoder(file).Decode(&ffStore); err != nil {
		return ffStore, err
	}
	return ffStore, nil
}

func saveStartupCacheFile(ffStore map[string]bool, filepath string) error {
	if filepath == "" {
		return nil
	}

	file, err := os.Create(filepath) //nolint:gosec
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			logger.WithError(err).Error("Unable to close cache file after write")
		}
	}(file)

	if err := json.NewEncoder(file).Encode(ffStore); err != nil {
		return err
	}
	return nil
}

func GetStartupFeatureFlagsAndStore(apiURL string, curVersion *semver.Version, unleashCachePathProvider func() (string, error)) map[string]bool {
	var cacheFilepath string
	cacheDir, err := unleashCachePathProvider()
	if err != nil {
		logger.WithError(err).Warn("Unable to obtain feature flag cache filepath")
	} else {
		cacheFilepath = filepath.Clean(filepath.Join(cacheDir, startupCacheFilename))
	}

	ffStore, err := readStartupCacheFile(cacheFilepath)
	if err != nil {
		logger.WithError(err).Warn("An issue occurred when reading the cache file")
	}

	manager := proton.New(newAPIOptions(apiURL, curVersion)...)
	featureFlagResult, err := manager.GetFeatures(context.Background(), uuid.New())
	if err == nil {
		ffStore = readResponseData(featureFlagResult)
	} else {
		logger.WithError(err).Warn("Failed to obtain feature flags from API")
	}

	if err := saveStartupCacheFile(ffStore, cacheFilepath); err != nil {
		logger.WithError(err).Warn("An issue occurred when saving the cache file")
	}

	return ffStore
}
