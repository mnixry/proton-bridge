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

package smtp

import (
	"errors"
	"fmt"

	"github.com/ProtonMail/go-proton-api"
)

var (
	ErrInvalidRecipient             = errors.New("invalid recipient")
	ErrInvalidReturnPath            = errors.New("invalid return path")
	ErrNoSuchUser                   = errors.New("no such user")
	ErrTooManyErrors                = errors.New("too many failed requests, please try again later")
	ErrSendMessageOperation         = errors.New("smtp: send message")
	ErrGetRecipientsOperation       = errors.New("smtp: get recipients")
	ErrGetSendPreferencesOperation  = errors.New("smtp: get send preferences")
	ErrLookupRecipientPublicKey     = errors.New("smtp: lookup recipient public key")
	ErrRecipientAddressDoesNotExist = errors.New("smtp: recipient address does not exist")

	ErrCannotSendFromAddressKind = errors.New("smtp: cannot send from address")
	ErrSenderAddressNotOwned     = errors.New("smtp: sender address not owned by user")
	ErrUnsupportedOutgoingMIME   = errors.New("smtp: unsupported outgoing MIME type")
)

const errCodeAddressDoesNotExist proton.Code = 33102

type ErrCannotSendFromAddress struct {
	address string
}

func NewErrCannotSendFromAddress(address string) *ErrCannotSendFromAddress {
	return &ErrCannotSendFromAddress{address: address}
}

func (e ErrCannotSendFromAddress) Error() string {
	return fmt.Sprintf("cannot send from address: %v", e.address)
}

func (e *ErrCannotSendFromAddress) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrCannotSendFromAddressKind
}
