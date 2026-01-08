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

package fido

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ProtonMail/go-proton-api"
)

type authData struct {
	AllowCredentials    []interface{}
	ClientDataJSONBytes []byte
	RpID                string
}

func extractFidoAuthData(auth proton.Auth) (authData, error) {
	authOptions, ok := auth.TwoFA.FIDO2.AuthenticationOptions.(map[string]interface{})
	if !ok {
		return authData{}, fmt.Errorf("invalid authentication options format")
	}

	publicKey, ok := authOptions["publicKey"].(map[string]interface{})
	if !ok {
		return authData{}, fmt.Errorf("no publicKey found in authentication options")
	}

	rpID, ok := publicKey["rpId"].(string)
	if !ok {
		return authData{}, fmt.Errorf("could not find rpId in authentication options")
	}

	challengeArray, ok := publicKey["challenge"].([]interface{})
	if !ok {
		return authData{}, fmt.Errorf("no challenge found in authentication options")
	}
	challenge := sliceAnyToByteArray(challengeArray)

	allowCredentials, ok := publicKey["allowCredentials"].([]interface{})
	if !ok || len(allowCredentials) == 0 {
		return authData{}, fmt.Errorf("no allowed credentials found in authentication options")
	}

	clientDataJSON := map[string]interface{}{
		"type":      "webauthn.get",
		"challenge": base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(challenge),
		"origin":    "https://" + rpID,
	}

	clientDataJSONBytes, err := json.Marshal(clientDataJSON)
	if err != nil {
		return authData{}, fmt.Errorf("failed to marshal client data JSON: %w", err)
	}

	return authData{
		AllowCredentials:    allowCredentials,
		ClientDataJSONBytes: clientDataJSONBytes,
		RpID:                rpID,
	}, nil
}

func sliceAnyToByteArray(s []any) []byte {
	result := make([]byte, len(s))
	for i, val := range s {
		if intVal, ok := val.(float64); ok {
			result[i] = byte(intVal)
		} else {
			panic("boom")
		}
	}
	return result
}

func authWithFido(client *proton.Client, auth proton.Auth, credentialIDs []byte, clientDataJSON []byte, authDataRaw []byte, signature []byte) error {
	credentialIDInts := make([]int, len(credentialIDs))
	for i, b := range credentialIDs {
		credentialIDInts[i] = int(b)
	}

	fido2Req := proton.FIDO2Req{
		AuthenticationOptions: auth.TwoFA.FIDO2.AuthenticationOptions,
		ClientData:            base64.StdEncoding.EncodeToString(clientDataJSON),
		AuthenticatorData:     base64.StdEncoding.EncodeToString(authDataRaw),
		Signature:             base64.StdEncoding.EncodeToString(signature),
		CredentialID:          credentialIDInts,
	}

	if err := client.Auth2FA(context.Background(), proton.Auth2FAReq{FIDO2: fido2Req}); err != nil {
		return fmt.Errorf("FIDO2 authentication failed: %w", err)
	}

	return nil
}
