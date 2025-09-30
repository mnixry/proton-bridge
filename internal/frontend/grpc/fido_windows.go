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

package grpc

import (
	"context"
	"fmt"
	"github.com/ProtonMail/go-proton-api"

	"github.com/ProtonMail/gluon/async"
	"github.com/ProtonMail/proton-bridge/v3/internal/fido"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Service) LoginFido(_ context.Context, login *LoginRequest) (*emptypb.Empty, error) {
	s.log.WithField("username", login.Username).Debug("LoginFido")
	go func() {
		defer async.HandlePanic(s.panicHandler)

		if s.auth.UID == "" || s.authClient == nil {
			s.log.Errorf("Login FIDO: authentication incomplete %s %p", s.auth.UID, s.authClient)
			_ = s.SendEvent(NewLoginError(LoginErrorType_TFA_ABORT, "Missing authentication, try again."))
			s.loginClean()
			return
		}

		if err := fido.AuthWithHardwareKeyGUI(s.authClient, s.auth, false); err != nil {
			_ = s.SendEvent(NewLoginError(LoginErrorType_FIDO_ERROR, fmt.Sprintf("Security key authentication failed: %s", err)))
			s.loginClean()
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
