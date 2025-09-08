//go:build windows

package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ProtonMail/go-proton-api"
	"github.com/go-ctap/ctaphid/pkg/webauthntypes"
	"github.com/go-ctap/winhello"
	"github.com/go-ctap/winhello/hiddenwindow"
)

func (f *frontendCLI) authWithHardwareKey(client *proton.Client, auth proton.Auth) error {
	// Windows Hello requires a window handle to work, as indicated by the docs of the lib.
	wnd, err := hiddenwindow.New(slog.New(slog.DiscardHandler), "Proton Bridge Auth")
	if err != nil {
		return fmt.Errorf("failed to create window for Windows Hello: %w", err)
	}
	defer wnd.Close()

	authOptions, ok := auth.TwoFA.FIDO2.AuthenticationOptions.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid authentication options format")
	}

	publicKey, ok := authOptions["publicKey"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no publicKey found in authentication options")
	}

	rpId, ok := publicKey["rpId"].(string)
	if !ok {
		return fmt.Errorf("could not find rpId in authentication options")
	}

	challengeArray, ok := publicKey["challenge"].([]interface{})
	if !ok {
		return fmt.Errorf("no challenge found in authentication options")
	}
	challenge := sliceAnyToByteArray(challengeArray)

	allowCredentials, ok := publicKey["allowCredentials"].([]interface{})
	if !ok || len(allowCredentials) == 0 {
		return fmt.Errorf("no allowed credentials found in authentication options")
	}

	var credentialDescriptors []webauthntypes.PublicKeyCredentialDescriptor
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

		credentialDescriptors = append(credentialDescriptors, webauthntypes.PublicKeyCredentialDescriptor{
			ID:   credID,
			Type: webauthntypes.PublicKeyCredentialTypePublicKey,
		})
	}

	if len(credentialDescriptors) == 0 {
		return fmt.Errorf("no valid credential descriptors found")
	}

	clientDataJSON := map[string]interface{}{
		"type":      "webauthn.get",
		"challenge": base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(challenge),
		"origin":    "https://" + rpId,
	}

	clientDataJSONBytes, err := json.Marshal(clientDataJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal client data JSON: %w", err)
	}

	fmt.Println("Please use Windows Hello to authenticate.")

	assertion, err := winhello.GetAssertion(
		wnd.WindowHandle(),
		rpId,
		clientDataJSONBytes,
		credentialDescriptors,
		nil,
		&winhello.AuthenticatorGetAssertionOptions{
			AuthenticatorAttachment:     winhello.WinHelloAuthenticatorAttachmentCrossPlatform,
			UserVerificationRequirement: winhello.WinHelloUserVerificationRequirementDiscouraged,
			CredentialHints: []webauthntypes.PublicKeyCredentialHint{
				webauthntypes.PublicKeyCredentialHintSecurityKey,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("windows Hello assertion failed: %w", err)
	}

	authData := assertion.AuthDataRaw
	credentialIDInts := make([]int, len(assertion.Credential.ID))
	for i, b := range assertion.Credential.ID {
		credentialIDInts[i] = int(b)
	}

	fido2Req := proton.FIDO2Req{
		AuthenticationOptions: auth.TwoFA.FIDO2.AuthenticationOptions,
		ClientData:            base64.StdEncoding.EncodeToString(clientDataJSONBytes),
		AuthenticatorData:     base64.StdEncoding.EncodeToString(authData),
		Signature:             base64.StdEncoding.EncodeToString(assertion.Signature),
		CredentialID:          credentialIDInts,
	}

	fmt.Println("Submitting FIDO2 authentication request.")
	if err := client.Auth2FA(context.Background(), proton.Auth2FAReq{FIDO2: fido2Req}); err != nil {
		return fmt.Errorf("FIDO2 authentication failed: %w", err)
	} else {
		fmt.Println("FIDO2 authentication succeeded")
	}
	return nil
}
