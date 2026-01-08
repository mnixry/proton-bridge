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

//go:build darwin || linux

package fido

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/fxamacker/cbor/v2"
	"github.com/keys-pub/go-libfido2"
)

var ErrAssertionCancelled = errors.New("FIDO assertion cancelled")

const (
	clientPinOption        = "clientPin"
	touchNotificationDelay = 500 * time.Millisecond
)

func getFidoDevice() (fido2Device *libfido2.Device, err error) {
	locs, err := libfido2.DeviceLocations()
	if err != nil {
		return nil, fmt.Errorf("could not find security key device location: %w", err)
	}

	if len(locs) == 0 {
		return nil, errors.New("no device found")
	}

	if len(locs) > 1 {
		return nil, errors.New("multiple security keys detected, please disconnect all but one device and try again")
	}

	fido2Device, err = libfido2.NewDevice(locs[0].Path)
	if err != nil {
		return nil, fmt.Errorf("cannot open security key: %w", err)
	}

	return fido2Device, nil
}

func deviceHasOption(dev *libfido2.Device, name string) (bool, error) {
	info, err := dev.Info()
	if err != nil {
		return false, fmt.Errorf("cannot get device info: %w", err)
	}

	for _, opt := range info.Options {
		if opt.Name == name && opt.Value == libfido2.True {
			return true, err
		}
	}

	return false, err
}

func IsPinSupported() (bool, error) {
	dev, err := getFidoDevice()
	if err != nil {
		return false, err
	}

	return deviceHasOption(dev, clientPinOption)
}

func constructCredentialIDs(allowCredentials []interface{}) ([][]byte, error) {
	var credentialIDs [][]byte //nolint:prealloc
	for _, cred := range allowCredentials {
		credMap, ok := cred.(map[string]interface{})
		if !ok {
			continue
		}
		idArray, ok := credMap["id"].([]interface{})
		if !ok {
			continue
		}
		credID := sliceAnyToByteArray(idArray)
		credentialIDs = append(credentialIDs, credID)
	}

	if len(credentialIDs) == 0 {
		return nil, errors.New("no valid credential IDs found")
	}

	return credentialIDs, nil
}

func prepareFidoAuth(auth proton.Auth) (*libfido2.Device, authData, [][]byte, [32]byte, error) {
	dev, err := getFidoDevice()
	if err != nil {
		return nil, authData{}, nil, [32]byte{}, fmt.Errorf("could not obtain security key device: %w", err)
	}

	data, err := extractFidoAuthData(auth)
	if err != nil {
		return nil, authData{}, nil, [32]byte{}, fmt.Errorf("could not extract security key authentication data: %w", err)
	}

	credentialIDs, err := constructCredentialIDs(data.AllowCredentials)
	if err != nil {
		return nil, authData{}, nil, [32]byte{}, err
	}

	clientDataHash := sha256.Sum256(data.ClientDataJSONBytes)

	return dev, data, credentialIDs, clientDataHash, nil
}

func processFidoAssertion(assertion *libfido2.Assertion) ([]byte, error) {
	var authData []byte
	if err := cbor.Unmarshal(assertion.AuthDataCBOR, &authData); err != nil {
		return nil, fmt.Errorf("failed to decode CBOR authenticator data: %w", err)
	}
	return authData, nil
}

func performAssertion(dev *libfido2.Device, rpID string, clientDataHash []byte, credentialIDs [][]byte, pin string) (*libfido2.Assertion, error) {
	assertion, err := dev.Assertion(
		rpID,
		clientDataHash,
		credentialIDs,
		pin,
		&libfido2.AssertionOpts{UP: libfido2.True},
	)
	if err != nil {
		return nil, fmt.Errorf("FIDO2 assertion failed: %w", err)
	}
	return assertion, nil
}

// performAssertationWithTimeout - initializes the assertion and sends data to the touchEventCh (with some delay) in parallel.
func performAssertionWithTimeout(ctx context.Context, dev *libfido2.Device, rpID string, clientDataHash []byte, credentialIDs [][]byte, pin string, touchEventCh chan struct{}) (*libfido2.Assertion, error) {
	type assertionResult struct {
		assertion *libfido2.Assertion
		err       error
	}

	resultCh := make(chan assertionResult, 1)
	go func() {
		assertion, err := performAssertion(dev, rpID, clientDataHash, credentialIDs, pin)
		resultCh <- assertionResult{assertion: assertion, err: err}
	}()

	nearTimeout := time.NewTimer(touchNotificationDelay)
	defer nearTimeout.Stop()

	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return result.assertion, nil

	case <-nearTimeout.C:
		// Notify that touch is required.
		select {
		case touchEventCh <- struct{}{}:
		default:
		}

		// Wait for either completion or cancellation.
		select {
		case result := <-resultCh:
			if result.err != nil {
				return nil, result.err
			}
			return result.assertion, nil

		case <-ctx.Done():
			if err := dev.Cancel(); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrAssertionCancelled, err)
			}
			return nil, ErrAssertionCancelled
		}
	}
}

func AuthWithHardwareKeyGUI(ctx context.Context, client *proton.Client, auth proton.Auth, touchEventCh chan struct{}, touchConfirmCh chan struct{}, pin string) error {
	dev, fidoAuthData, credentialIDs, clientDataHash, err := prepareFidoAuth(auth)
	if err != nil {
		return err
	}

	assertion, err := performAssertionWithTimeout(ctx, dev, fidoAuthData.RpID, clientDataHash[:], credentialIDs, pin, touchEventCh)
	if err != nil {
		return err
	}

	// Notify that spinner should be displayed, as assertion has finished.
	touchConfirmCh <- struct{}{}

	// Decode CBOR to get raw authenticator data.
	authData, err := processFidoAssertion(assertion)
	if err != nil {
		return err
	}

	return authWithFido(client,
		auth,
		assertion.CredentialID,
		fidoAuthData.ClientDataJSONBytes,
		authData,
		assertion.Sig)
}

func AuthWithHardwareKeyCLI(cliProvider CLIProvider, client *proton.Client, auth proton.Auth) error {
	cliProvider.PromptAndWaitReturn("Please insert your security key")

	dev, fidoAuthData, credentialIDs, clientDataHash, err := prepareFidoAuth(auth)
	if err != nil {
		return err
	}

	pinSupported, err := IsPinSupported()
	if err != nil {
		return fmt.Errorf("could not determine security key PIN support: %w", err)
	}

	var pin string
	if pinSupported {
		pin = cliProvider.ReadSecurityKeyPin()
		if pin == "" {
			return errors.New("a PIN is required for this security key")
		}
	}

	fmt.Println("Please touch the button or sensor on your security key.")
	assertion, err := performAssertion(dev, fidoAuthData.RpID, clientDataHash[:], credentialIDs, pin)
	if err != nil {
		return err
	}

	authData, err := processFidoAssertion(assertion)
	if err != nil {
		return err
	}

	fmt.Println("Submitting FIDO2 authentication request.")
	return authWithFido(
		client,
		auth,
		assertion.CredentialID,
		fidoAuthData.ClientDataJSONBytes,
		authData,
		assertion.Sig)
}
