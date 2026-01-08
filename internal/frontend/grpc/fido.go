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

//go:build linux || darwin

package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/proton-bridge/v3/internal/fido"
	"github.com/keys-pub/go-libfido2"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) LoginFido(_ context.Context, login *LoginRequest) (*emptypb.Empty, error) {
	s.log.WithField("username", login.Username).Debug("LoginFido")

	go func() {
		defer async.HandlePanic(s.panicHandler)

		fidoCtx, cancelFido := context.WithCancel(context.Background())
		s.fidoManager.SetCancel(cancelFido)
		defer s.fidoManager.Clear()

		if s.auth.UID == "" || s.authClient == nil {
			s.log.Errorf("Login FIDO: authentication incomplete %s %p", s.auth.UID, s.authClient)
			_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_ERROR, "Missing authentication, try again."))
			s.loginClean()
			return
		}

		pinSupported, err := fido.IsPinSupported()
		if err != nil {
			s.log.WithError(err).Warn("could not determine security key PIN requirements")
			_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_ERROR, fmt.Sprintf("Could not obtain security key pin requirements: %s", err)))
			s.loginClean()
			return
		}

		if pinSupported && len(login.Password) == 0 {
			_ = s.SendEvent(NewLoginFidoPinRequired(login.Username))
			return
		}

		pin, err := base64Decode(login.Password)
		if err != nil {
			s.log.WithError(err).Error("cannot decode security key device pin")
			_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_PIN_INVALID, "Could not decode security key PIN"))
			return
		}

		touchCh := make(chan struct{})
		touchConfirmCh := make(chan struct{})

		defer func() {
			close(touchCh)
			close(touchConfirmCh)
		}()

		go func() {
			if _, ok := <-touchCh; ok {
				_ = s.SendEvent(NewLoginFidoTouchRequested(login.Username))
				if _, ok := <-touchConfirmCh; ok {
					_ = s.SendEvent(NewLoginFidoTouchCompleted(login.Username))
				}
			}
		}()

		if err := fido.AuthWithHardwareKeyGUI(fidoCtx, s.authClient, s.auth, touchCh, touchConfirmCh, string(pin)); err != nil {
			s.log.WithError(err).Warn("Login FIDO: failed")
			switch {
			case errors.Is(err, libfido2.ErrPinAuthBlocked):
				_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_PIN_BLOCKED, "Security key PIN code is blocked"))

			case errors.Is(err, libfido2.ErrPinInvalid):
				_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_PIN_INVALID, "Security key PIN code is incorrect"))

			case errors.Is(err, fido.ErrAssertionCancelled): // User cancellation, they can click re-auth again.
				return

			default:
				_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_ERROR, fmt.Sprintf("Security key authentication failed: %s", err)))
				s.loginClean()
			}

			return
		}

		if s.auth.PasswordMode == proton.TwoPasswordMode {
			_ = s.SendEvent(NewLoginTwoPasswordsRequestedEvent(login.Username))
			return
		}

		s.finishLogin()
	}()

	return &emptypb.Empty{}, nil
}
