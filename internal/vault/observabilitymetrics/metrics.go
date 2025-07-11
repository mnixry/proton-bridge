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

package observabilitymetrics

import (
	"time"

	"github.com/ProtonMail/go-proton-api"
)

const (
	vaultErrorsSchemaName    = "bridge_vault_errors_total"
	vaultErrorsSchemaVersion = 1
)

func generateVaultErrorObservabilityMetric(errorType string) proton.ObservabilityMetric {
	return proton.ObservabilityMetric{
		Name:        vaultErrorsSchemaName,
		Version:     vaultErrorsSchemaVersion,
		Timestamp:   time.Now().Unix(),
		ShouldCache: true,
		Data: map[string]interface{}{
			"Value": 1,
			"Labels": map[string]string{
				"errorType": errorType,
			},
		},
	}
}

func GenerateVaultCreationCorruptErrorMetric() proton.ObservabilityMetric {
	return generateVaultErrorObservabilityMetric("vaultCorrupt")
}

func GenerateVaultCreationGenericErrorMetric() proton.ObservabilityMetric {
	return generateVaultErrorObservabilityMetric("vaultError")
}

func GenerateVaultKeyFetchGenericErrorMetric() proton.ObservabilityMetric {
	return generateVaultErrorObservabilityMetric("keychainError")
}
