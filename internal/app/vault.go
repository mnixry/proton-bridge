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

package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"runtime"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/proton-bridge/v3/internal/certs"
	"github.com/ProtonMail/proton-bridge/v3/internal/constants"
	"github.com/ProtonMail/proton-bridge/v3/internal/locations"
	"github.com/ProtonMail/proton-bridge/v3/internal/platform"
	"github.com/ProtonMail/proton-bridge/v3/internal/sentry"
	"github.com/ProtonMail/proton-bridge/v3/internal/services/observability"
	"github.com/ProtonMail/proton-bridge/v3/internal/unleash"
	"github.com/ProtonMail/proton-bridge/v3/internal/vault"
	"github.com/ProtonMail/proton-bridge/v3/internal/vault/observabilitymetrics"
	"github.com/ProtonMail/proton-bridge/v3/pkg/keychain"
	"github.com/sirupsen/logrus"
)

func WithVault(reporter *sentry.Reporter, locations *locations.Locations, keychains *keychain.List, obsSender observability.BasicSender, featureFlags unleash.FeatureFlagStartupStore, panicHandler async.PanicHandler, fn func(*vault.Vault, bool, bool) error) error {
	logrus.Debug("Creating vault")
	defer logrus.Debug("Vault stopped")

	// Create the encVault.
	encVault, insecure, corrupt, err := newVault(reporter, locations, keychains, obsSender, featureFlags, panicHandler)
	if err != nil {
		return fmt.Errorf("could not create vault: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"insecure": insecure,
		"corrupt":  corrupt != nil,
	}).Debug("Vault created")

	if corrupt != nil {
		logrus.WithError(corrupt).Warn("Failed to load existing vault, vault has been reset")
	}

	cert, _ := encVault.GetBridgeTLSCert()
	certs.NewInstaller().LogCertInstallStatus(cert)

	// GODT-1950: Add teardown actions (e.g. to close the vault).

	return fn(encVault, insecure, corrupt != nil)
}

func newVault(reporter *sentry.Reporter, locations *locations.Locations, keychains *keychain.List, obsSender observability.BasicSender, featureFlags unleash.FeatureFlagStartupStore, panicHandler async.PanicHandler) (*vault.Vault, bool, error, error) {
	vaultDir, err := locations.ProvideSettingsPath()
	if err != nil {
		return nil, false, nil, fmt.Errorf("could not get vault dir: %w", err)
	}

	logrus.WithField("vaultDir", vaultDir).Debug("Loading vault from directory")

	var (
		vaultKey       []byte
		insecure       bool
		lastUsedHelper string
	)

	if key, helper, err := loadVaultKey(vaultDir, keychains, featureFlags); err != nil {
		if errors.Is(err, keychain.ErrPreferredKeychainNotAvailable) {
			if err := vault.IncrementKeychainFailedAttemptCount(vaultDir); err != nil {
				logrus.WithError(err).Error("Failed to increment failed keychain attempt count")
			}
			return &vault.Vault{}, false, nil, err
		}

		if reporter != nil {
			if rerr := reporter.ReportMessageWithContext("Could not load/create vault key", map[string]any{
				"keychainDefaultHelper":       keychains.GetDefaultHelper(),
				"keychainUsableHelpersLength": len(keychains.GetHelpers()),
				"error":                       err.Error(),
			}); rerr != nil {
				logrus.WithError(err).Info("Failed to report keychain issue to Sentry")
			}
		}

		logrus.WithError(err).Error("Could not load/create vault key")
		insecure = true

		// We store the insecure vault in a separate directory
		vaultDir = path.Join(vaultDir, "insecure")

		// Schedule the relevant observability metric for sending.
		obsSender.AddMetrics(observabilitymetrics.GenerateVaultKeyFetchGenericErrorMetric())
	} else {
		vaultKey = key
		lastUsedHelper = helper
		logHashedVaultKey(vaultKey) // Log a hash of the vault key.
	}

	gluonCacheDir, err := locations.ProvideGluonCachePath()
	if err != nil {
		return nil, false, nil, fmt.Errorf("could not provide gluon path: %w", err)
	}

	userVault, corrupt, err := vault.New(vaultDir, gluonCacheDir, vaultKey, panicHandler)
	if err != nil {
		obsSender.AddMetrics(observabilitymetrics.GenerateVaultCreationGenericErrorMetric())
		return nil, false, corrupt, fmt.Errorf("could not create vault: %w", err)
	}

	if corrupt != nil {
		obsSender.AddMetrics(observabilitymetrics.GenerateVaultCreationCorruptErrorMetric())
	}

	// Remember the last successfully used keychain on Linux and store that as the user preference.
	if runtime.GOOS == platform.LINUX {
		if err := vault.SetHelper(vaultDir, lastUsedHelper); err != nil {
			logrus.WithError(err).Error("Could not store last used keychain helper")
		}

		if err := vault.ResetFailedKeychainAttemptCount(vaultDir); err != nil {
			logrus.WithError(err).Error("Could not reset and save failed keychain attempt count")
		}
	}

	return userVault, insecure, corrupt, nil
}

// loadVaultKey - loads the key used to encrypt the vault alongside the keychain helper used to access it.
func loadVaultKey(vaultDir string, keychains *keychain.List, featureFlags unleash.FeatureFlagStartupStore) (key []byte, keychainHelper string, err error) {
	keychainHelper, err = vault.GetHelper(vaultDir)
	if err != nil {
		return nil, keychainHelper, fmt.Errorf("could not get keychain helper: %w", err)
	}

	keychainFailedAttemptCount, err := vault.GetKeychainFailedAttemptCount(vaultDir)
	if err != nil {
		return nil, keychainHelper, fmt.Errorf("could not get keychain failed attempt count: %w", err)
	}

	kc, keychainHelper, err := keychain.NewKeychain(
		keychainHelper, constants.KeyChainName,
		keychains.GetHelpers(),
		keychains.GetDefaultHelper(),
		keychainFailedAttemptCount,
		featureFlags,
	)
	if err != nil {
		return nil, keychainHelper, fmt.Errorf("could not create keychain: %w", err)
	}

	logrus.WithField("keychainHelper", keychainHelper).Info("Initialized keychain helper")

	key, err = vault.GetVaultKey(kc)
	if err != nil {
		if keychain.IsErrKeychainNoItem(err) {
			logrus.WithError(err).Warn("no vault key found, generating new")
			key, err := vault.NewVaultKey(kc)
			return key, keychainHelper, err
		}

		if keychain.ShouldRetryPreferredKeychain(featureFlags, keychainHelper) {
			if keychainFailedAttemptCount < keychain.MaxFailedKeychainAttemptsLinux {
				return nil, keychainHelper, keychain.PreferredKeychainRetryError(keychainFailedAttemptCount)
			}
		}

		return nil, keychainHelper, fmt.Errorf("could not check for vault key: %w", err)
	}

	return key, keychainHelper, nil
}

// logHashedVaultKey - computes a sha256 hash and encodes it to base 64. The resulting string is logged.
func logHashedVaultKey(vaultKey []byte) {
	hashedKey := sha256.Sum256(vaultKey)
	logrus.WithField("hashedKey", hex.EncodeToString(hashedKey[:])).Info("Found vault key")
}
