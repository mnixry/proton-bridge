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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestReadStartupCacheFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "valid_cache")
	file, err := os.Create(filePath)
	require.NoError(t, err)

	testData := map[string]bool{
		"feature1": true,
		"feature2": false,
	}
	err = json.NewEncoder(file).Encode(testData)
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	startupCache, err := readStartupCacheFile(filePath)
	require.NoError(t, err)
	require.Equal(t, testData, startupCache)
}

func TestReadStartupCacheFile_InvalidFilePath(t *testing.T) {
	filePath := "badFilepath/hello"
	startupCache, err := readStartupCacheFile(filePath)
	require.Error(t, err)
	require.Empty(t, startupCache)
}

func TestSaveStartupCacheFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_cache")

	testData := map[string]bool{
		"feature1": true,
		"feature2": false,
		"feature3": true,
	}

	err := saveStartupCacheFile(testData, filePath)
	require.NoError(t, err)

	savedData, err := readStartupCacheFile(filePath)
	require.NoError(t, err)
	require.Equal(t, testData, savedData)
}

func TestSaveStartupCacheFile_InvalidFilePath(t *testing.T) {
	badFilePath := "/some_random_dir/hey/hello"

	testData := map[string]bool{
		"feature1": true,
		"feature2": false,
	}

	err := saveStartupCacheFile(testData, badFilePath)
	require.Error(t, err)
}

func TestGetStartupFeatureFlagsAndStore_FakeAPIURL(t *testing.T) {
	apiURL := "https://example.com"
	cacheProvider := func() (string, error) {
		return t.TempDir(), nil
	}

	version, err := semver.NewVersion("3.99.99+test")
	require.NoError(t, err)

	featureFlags := GetStartupFeatureFlagsAndStore(apiURL, version, cacheProvider)
	require.Empty(t, featureFlags)
}

func TestGetStartupFeatureFlagsAndStore_RealAPIURL(t *testing.T) {
	apiURL := "https://mail-api.proton.me"
	cacheProvider := func() (string, error) {
		return t.TempDir(), nil
	}

	version, err := semver.NewVersion("3.99.99+test")
	require.NoError(t, err)

	featureFlags := GetStartupFeatureFlagsAndStore(apiURL, version, cacheProvider)
	require.NotEmpty(t, featureFlags)
}

func TestGetStartupFeatureFlagsAndStore_FeatureFlagCacheRetention(t *testing.T) {
	fakeAPIURL := "https://example.com"
	realAPIURL := "https://mail-api.proton.me"

	cacheDir := t.TempDir()
	cacheProvider := func() (string, error) {
		return cacheDir, nil
	}

	version, err := semver.NewVersion("3.99.99+test")
	require.NoError(t, err)

	featureFlags := GetStartupFeatureFlagsAndStore(realAPIURL, version, cacheProvider)
	require.NotEmpty(t, featureFlags)

	featureFlagsFromCache := GetStartupFeatureFlagsAndStore(fakeAPIURL, version, cacheProvider)
	require.NotEmpty(t, featureFlagsFromCache)
	require.Equal(t, featureFlags, featureFlagsFromCache)
}

func Test(t *testing.T) {
	fakeAPIURL := "https://example.com"

	tmpDir := t.TempDir()
	cacheProvider := func() (string, error) {
		return tmpDir, nil
	}
	filePath := filepath.Join(tmpDir, startupCacheFilename)

	testData := map[string]bool{
		"feature1": true,
		"feature2": false,
		"feature3": true,
	}
	err := saveStartupCacheFile(testData, filePath)
	require.NoError(t, err)

	version, err := semver.NewVersion("3.99.99+git")
	require.NoError(t, err)

	featureFlagsFromCache := GetStartupFeatureFlagsAndStore(fakeAPIURL, version, cacheProvider)
	require.NotEmpty(t, featureFlagsFromCache)
	require.Equal(t, testData, featureFlagsFromCache)
}
