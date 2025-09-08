//go:build linux || darwin

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ProtonMail/go-proton-api"
	"github.com/fxamacker/cbor/v2"
	"github.com/keys-pub/go-libfido2"
)

func (f *frontendCLI) authWithHardwareKey(client *proton.Client, auth proton.Auth) error {
	var fido2Device *libfido2.DeviceLocation

	retryCount := 0
	for {
		locs, err := libfido2.DeviceLocations()
		if err != nil {
			f.printAndLogError("Cannot retrieve security key list: ", err)
		}
		if len(locs) == 0 {
			fmt.Print("Please insert your security key and press enter to continue.")
			f.ReadLine()
		} else {
			fido2Device = locs[0]
			break
		}

		retryCount++
		if retryCount >= 3 {
			break
		}
	}

	if fido2Device == nil {
		return errors.New("no device found")
	}

	dev, err := libfido2.NewDevice(fido2Device.Path)
	if err != nil {
		return fmt.Errorf("cannot open security key: %w", err)
	}

	// Check if the key has a PIN set first
	var pin string
	info, err := dev.Info()
	if err != nil {
		return fmt.Errorf("cannot get device info: %w", err)
	}

	// Check if clientPin option is available and set
	pinSupported := false
	for _, option := range info.Options {
		if option.Name == "clientPin" && option.Value == libfido2.True {
			pinSupported = true
			break
		}
	}

	if pinSupported {
		pin = f.readStringInAttempts("Security key PIN", f.ReadPassword, isNotEmpty)
		if pin == "" {
			return errors.New("PIN is required for this security key")
		}
	}

	fmt.Println("Please touch your security key...")

	authOptions, ok := auth.TwoFA.FIDO2.AuthenticationOptions.(map[string]interface{})
	if !ok {
		return errors.New("invalid authentication options format")
	}

	publicKey, ok := authOptions["publicKey"].(map[string]interface{})
	if !ok {
		return errors.New("no publicKey found in authentication options")
	}

	allowCredentials, ok := publicKey["allowCredentials"].([]interface{})
	if !ok || len(allowCredentials) == 0 {
		return errors.New("no allowed credentials found in authentication options")
	}

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
		return errors.New("no valid credential IDs found")
	}

	challengeArray, ok := publicKey["challenge"].([]interface{})
	if !ok {
		return errors.New("no challenge found in authentication options")
	}
	challenge := sliceAnyToByteArray(challengeArray)

	rpID, ok := publicKey["rpId"].(string)
	if !ok {
		return errors.New("could not find rpId in authentication options")
	}

	clientDataJSON := map[string]interface{}{
		"type":      "webauthn.get",
		"challenge": base64.URLEncoding.EncodeToString(challenge),
		"origin":    "https://" + rpID,
	}

	clientDataJSONBytes, err := json.Marshal(clientDataJSON)
	if err != nil {
		return fmt.Errorf("cannot marshal client data: %w", err)
	}

	clientDataHash := sha256.Sum256(clientDataJSONBytes)

	assertion, err := dev.Assertion(
		rpID,
		clientDataHash[:],
		credentialIDs,
		pin,
		&libfido2.AssertionOpts{UP: libfido2.True},
	)
	if err != nil {
		return fmt.Errorf("FIDO2 assertion failed: %w", err)
	}

	// Decode CBOR to get raw authenticator data
	var authData []byte
	err = cbor.Unmarshal(assertion.AuthDataCBOR, &authData)
	if err != nil {
		return fmt.Errorf("failed to decode CBOR authenticator data: %w", err)
	}

	// Convert CredentialID bytes to array of integers
	credentialIDInts := make([]int, len(assertion.CredentialID))
	for i, b := range assertion.CredentialID {
		credentialIDInts[i] = int(b)
	}

	fido2Req := proton.FIDO2Req{
		AuthenticationOptions: auth.TwoFA.FIDO2.AuthenticationOptions,
		ClientData:            base64.StdEncoding.EncodeToString(clientDataJSONBytes),
		AuthenticatorData:     base64.StdEncoding.EncodeToString(authData),
		Signature:             base64.StdEncoding.EncodeToString(assertion.Sig),
		CredentialID:          credentialIDInts,
	}

	fmt.Println("Submitting FIDO2 authentication request.")
	if err := client.Auth2FA(context.Background(), proton.Auth2FAReq{FIDO2: fido2Req}); err != nil {
		return fmt.Errorf("FIDO2 authentication failed: %w", err)
	}

	fmt.Println("FIDO2 authentication succeeded")
	return nil
}
