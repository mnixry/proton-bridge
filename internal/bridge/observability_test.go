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

package bridge_test

import (
	"context"
	"testing"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server"
	"github.com/ProtonMail/proton-bridge/v3/internal/bridge"
	"github.com/ProtonMail/proton-bridge/v3/internal/services/observability"
	"github.com/stretchr/testify/require"
)

func TestBridge_Observability(t *testing.T) {
	testMetric := proton.ObservabilityMetric{
		Name:      "test1",
		Version:   1,
		Timestamp: time.Now().Unix(),
		Data:      nil,
	}

	withEnv(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, vaultKey []byte) {
		throttlePeriod := time.Millisecond * 500
		observability.ModifyThrottlePeriod(throttlePeriod)

		withBridge(ctx, t, s.GetHostURL(), netCtl, locator, vaultKey, func(bridge *bridge.Bridge, _ *bridge.Mocks) {
			require.NoError(t, getErr(bridge.LoginFull(ctx, username, password, nil, nil)))

			bridge.PushObservabilityMetric(testMetric)
			time.Sleep(time.Millisecond * 50) // Wait for the metric to be sent
			require.Equal(t, 1, len(s.GetObservabilityStatistics().Metrics))

			for i := 0; i < 10; i++ {
				time.Sleep(time.Millisecond * 5) // Minor delay between each so our tests aren't flaky
				bridge.PushObservabilityMetric(testMetric)
			}
			// We should still have only 1 metric sent as the throttleDuration has not passed
			require.Equal(t, 1, len(s.GetObservabilityStatistics().Metrics))

			// Wait for throttle duration to pass; we should have our remaining metrics posted
			time.Sleep(throttlePeriod)
			require.Equal(t, 11, len(s.GetObservabilityStatistics().Metrics))

			// Wait for the throttle duration to reset; i.e. so we have enough time to send a request immediately
			time.Sleep(throttlePeriod)
			for i := 0; i < 10; i++ {
				time.Sleep(time.Millisecond * 5)
				bridge.PushObservabilityMetric(testMetric)
			}
			// We should only have one additional metric sent immediately
			require.Equal(t, 12, len(s.GetObservabilityStatistics().Metrics))

			// Wait for the others to be sent
			time.Sleep(throttlePeriod)
			require.Equal(t, 21, len(s.GetObservabilityStatistics().Metrics))

			// Spam the endpoint a bit
			for i := 0; i < 300; i++ {
				if i < 200 {
					time.Sleep(time.Millisecond * 10)
				}
				bridge.PushObservabilityMetric(testMetric)
			}

			// Ensure we've sent all metrics
			time.Sleep(throttlePeriod)

			observabilityStats := s.GetObservabilityStatistics()
			require.Equal(t, 321, len(observabilityStats.Metrics))

			// Verify that each request had a throttleDuration time difference between each request
			for i := 0; i < len(observabilityStats.RequestTime)-1; i++ {
				tOne := observabilityStats.RequestTime[i]
				tTwo := observabilityStats.RequestTime[i+1]
				require.True(t, tTwo.Sub(tOne).Abs() > throttlePeriod)
			}
		})
	})
}

func TestBridge_Observability_Heartbeat(t *testing.T) {
	withEnv(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, vaultKey []byte) {
		throttlePeriod := time.Millisecond * 300
		observability.ModifyThrottlePeriod(throttlePeriod)

		withBridge(ctx, t, s.GetHostURL(), netCtl, locator, vaultKey, func(bridge *bridge.Bridge, _ *bridge.Mocks) {
			require.NoError(t, getErr(bridge.LoginFull(ctx, username, password, nil, nil)))
			bridge.ModifyObservabilityHeartbeatInterval(throttlePeriod)

			require.Equal(t, 0, len(s.GetObservabilityStatistics().Metrics))
			time.Sleep(time.Millisecond * 150)
			require.Equal(t, 0, len(s.GetObservabilityStatistics().Metrics))
			time.Sleep(time.Millisecond * 200)
			require.Equal(t, 1, len(s.GetObservabilityStatistics().Metrics))
			time.Sleep(time.Millisecond * 350)
			require.Equal(t, 2, len(s.GetObservabilityStatistics().Metrics))
			time.Sleep(time.Millisecond * 350)
			require.Equal(t, 3, len(s.GetObservabilityStatistics().Metrics))
		})
	})
}

func TestBridge_Observability_UserMetric(t *testing.T) {
	testMetric := proton.ObservabilityMetric{
		Name:      "test1",
		Version:   1,
		Timestamp: time.Now().Unix(),
		Data:      nil,
	}

	withEnv(t, func(ctx context.Context, s *server.Server, netCtl *proton.NetCtl, locator bridge.Locator, vaultKey []byte) {
		// Keep a long heartbeat so that only user-metric pushes are tested not heartbeat ones.
		userMetricPeriod := time.Second
		heartbeatPeriod := time.Hour
		throttlePeriod := time.Millisecond * 300

		observability.ModifyUserMetricInterval(userMetricPeriod)
		observability.ModifyThrottlePeriod(throttlePeriod)

		withBridge(ctx, t, s.GetHostURL(), netCtl, locator, vaultKey, func(bridge *bridge.Bridge, _ *bridge.Mocks) {
			require.NoError(t, getErr(bridge.LoginFull(ctx, username, password, nil, nil)))
			bridge.ModifyObservabilityHeartbeatInterval(heartbeatPeriod)

			requireObservabilityMetricCountEventually(t, s, 0, 5*time.Second, 20*time.Millisecond)

			bridge.PushDistinctObservabilityMetrics(observability.SyncError, testMetric)
			// We're expecting two observability metrics to be sent, the actual metric + the user metric.
			requireObservabilityMetricCountEventually(t, s, 2, 5*time.Second, 20*time.Millisecond)

			bridge.PushDistinctObservabilityMetrics(observability.SyncError, testMetric)
			// We're expecting only a single metric to be sent, since the user metric update has been sent already within the predefined period.
			requireObservabilityMetricCountEventually(t, s, 3, 5*time.Second, 20*time.Millisecond)

			time.Sleep(userMetricPeriod + throttlePeriod)
			requireObservabilityMetricCountEventually(t, s, 3, 5*time.Second, 20*time.Millisecond) // check after timeout if no new events have been received

			bridge.PushDistinctObservabilityMetrics(observability.SyncError, testMetric)
			requireObservabilityMetricCountEventually(t, s, 5, 5*time.Second, 20*time.Millisecond)

			bridge.PushDistinctObservabilityMetrics(observability.SyncError, testMetric)
			requireObservabilityMetricCountEventually(t, s, 6, 5*time.Second, 20*time.Millisecond)

			time.Sleep(userMetricPeriod + throttlePeriod)
			requireObservabilityMetricCountEventually(t, s, 6, 5*time.Second, 20*time.Millisecond) // check after timeout if no new events have been received

			bridge.PushDistinctObservabilityMetrics(observability.SyncError, testMetric)
			requireObservabilityMetricCountEventually(t, s, 8, 5*time.Second, 20*time.Millisecond)

			bridge.PushDistinctObservabilityMetrics(observability.SyncError, testMetric)
			requireObservabilityMetricCountEventually(t, s, 9, 5*time.Second, 20*time.Millisecond)
		})
	})
}

// maxCooldownDuration & cooldownDuration always receive the same value, but I'd rather keep them as arguments rather than constants
//
//nolint:unparam
func requireObservabilityMetricCountEventually(t *testing.T, s *server.Server, expected int, maxCooldownDuration, cooldownDuration time.Duration) {
	t.Helper()
	require.Eventually(t, func() bool {
		return len(s.GetObservabilityStatistics().Metrics) == expected
	}, maxCooldownDuration, cooldownDuration)
}
