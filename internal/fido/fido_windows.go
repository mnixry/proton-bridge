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
// along with Proton Mail Bridge. If not, see <https://www.gnu.org/licenses/>.

//go:build windows

package fido

import (
	"fmt"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/go-ctap/ctaphid/pkg/webauthntypes"
	"github.com/go-ctap/winhello"
	"github.com/go-ctap/winhello/window"
)

func AuthWithHardwareKeyCLI(_ CLIProvider, client *proton.Client, auth proton.Auth) error {
	return AuthWithHardwareKeyGUI(client, auth, true)
}

func AuthWithHardwareKeyGUI(client *proton.Client, auth proton.Auth, onCLI bool) error {
	fidoAuthData, err := extractFidoAuthData(auth)
	if err != nil {
		return fmt.Errorf("could not extract security key authentication data: %w", err)
	}

	var credentialDescriptors []webauthntypes.PublicKeyCredentialDescriptor
	for _, cred := range fidoAuthData.AllowCredentials {
		credMap, ok := cred.(map[string]interface{})
		if !ok {
			continue
		}
		idArray, ok := credMap["id"].([]interface{})
		if !ok {
			continue
		}
		credID := sliceAnyToByteArray(idArray)
		credentialDescriptors = append(credentialDescriptors, webauthntypes.PublicKeyCredentialDescriptor{
			ID:   credID,
			Type: webauthntypes.PublicKeyCredentialTypePublicKey,
		})
	}

	if len(credentialDescriptors) == 0 {
		return fmt.Errorf("no valid credential descriptors found")
	}

	windowHandler, err := window.GetForegroundWindow()
	if err != nil {
		return fmt.Errorf("failed to obtain window handle: %w", err)
	}

	if onCLI {
		fmt.Println("Please use Windows Hello to authenticate.")
	}
	assertion, err := winhello.GetAssertion(windowHandler,
		fidoAuthData.RpID,
		fidoAuthData.ClientDataJSONBytes,
		credentialDescriptors,
		nil,
		&winhello.AuthenticatorGetAssertionOptions{
			Timeout:                     time.Second * 60,
			AuthenticatorAttachment:     winhello.WinHelloAuthenticatorAttachmentCrossPlatform,
			UserVerificationRequirement: winhello.WinHelloUserVerificationRequirementPreferred,
			CredentialHints: []webauthntypes.PublicKeyCredentialHint{
				webauthntypes.PublicKeyCredentialHintSecurityKey,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("windows Hello assertion failed: %w", err)
	}

	if onCLI {
		fmt.Println("Submitting FIDO2 authentication request.")
	}

	return authWithFido(client, auth, assertion.Credential.ID, fidoAuthData.ClientDataJSONBytes, assertion.AuthDataRaw, assertion.Signature)
}
