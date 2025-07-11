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

package observability

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/stretchr/testify/require"
)

func TestService_cacheFile_NoCachePath(t *testing.T) {
	s := NewTestService()
	s.readCacheFile()
	s.writeCacheFile()
	require.Empty(t, s.metricStore)
}

func TestService_cacheFile_ValidCachePath(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache.json")

	s := NewTestService()
	s.cachePath = cachePath

	s.readCacheFile()
	s.writeCacheFile()
	require.Empty(t, s.metricStore)
}

func TestService_cacheFile_AllMetricsCacheable(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Clean(filepath.Join(tempDir, "test_cache.json"))

	s := NewTestService()
	s.cachePath = cachePath
	s.ctx = context.Background()

	testMetrics := []proton.ObservabilityMetric{
		{
			Name:        "test1",
			Version:     1,
			Timestamp:   time.Now().Unix(),
			Data:        nil,
			ShouldCache: true,
		},
		{

			Name:        "test2",
			Version:     2,
			Timestamp:   time.Now().Unix(),
			Data:        nil,
			ShouldCache: true,
		},
		{

			Name:        "test3",
			Version:     3,
			Timestamp:   time.Now().Unix(),
			Data:        nil,
			ShouldCache: true,
		},
	}

	s.readCacheFile()
	require.Empty(t, s.metricStore)

	s.addMetrics(testMetrics...)
	require.Equal(t, s.metricStore, testMetrics)

	s.writeCacheFile()
	s.flushMetricsTest()
	require.Empty(t, s.metricStore)

	s.readCacheFile()
	require.Equal(t, s.metricStore, testMetrics)
}

func TestService_cacheFile_SomeMetricsCacheable(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Clean(filepath.Join(tempDir, "test_cache.json"))

	s := NewTestService()
	s.cachePath = cachePath
	s.ctx = context.Background()

	testMetricsCacheable := []proton.ObservabilityMetric{
		{
			Name:        "test1",
			Version:     1,
			Timestamp:   time.Now().Unix(),
			Data:        nil,
			ShouldCache: true,
		},
		{

			Name:        "test2",
			Version:     2,
			Timestamp:   time.Now().Unix(),
			Data:        nil,
			ShouldCache: true,
		},
	}

	testMetricsNonCacheable := []proton.ObservabilityMetric{
		{
			Name:      "test3",
			Version:   3,
			Timestamp: time.Now().Unix(),
		},
		{

			Name:      "test2",
			Version:   2,
			Timestamp: time.Now().Unix(),
		},
	}

	s.readCacheFile()
	require.Empty(t, s.metricStore)

	s.addMetrics(testMetricsCacheable...)
	s.addMetrics(testMetricsNonCacheable...)

	require.Equal(t, s.metricStore, append(testMetricsCacheable, testMetricsNonCacheable...))

	s.writeCacheFile()
	s.flushMetricsTest()
	require.Empty(t, s.metricStore)

	s.readCacheFile()
	require.Equal(t, s.metricStore, testMetricsCacheable)
}
