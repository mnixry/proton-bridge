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
	ErrInvalidRecipient            = errors.New("invalid recipient")
	ErrInvalidReturnPath           = errors.New("invalid return path")
	ErrNoSuchUser                  = errors.New("no such user")
	ErrSMTPAuthFailed              = errors.New("the account did not pass the smtp authentication check")
	ErrTooManyErrors               = errors.New("too many failed requests, please try again later")
	ErrSendMessageOperation        = errors.New("smtp: send message")
	ErrGetRecipientsOperation      = errors.New("smtp: get recipients")
	ErrGetSendPreferencesOperation = errors.New("smtp: get send preferences")
	ErrLookupRecipientPublicKey    = errors.New("smtp: lookup recipient public key")
	ErrSenderAddressNotOwned       = errors.New("smtp: sender address not owned by user")
	ErrUnsupportedOutgoingMIME     = errors.New("smtp: unsupported outgoing MIME type")
	ErrInvalidListOfRecipients     = errors.New("smtp: invalid list of recipients draft")
	ErrMessageTooLarge             = errors.New("smtp: message too large draft")
	ErrValidationFailed            = errors.New("smtp: validation failed")
)

const errCodeAddressDoesNotExist proton.Code = 33102
const errCodeValidationFailed proton.Code = 2001
const errCodeMessageTooLarge proton.Code = 2024
const errCodeInvalidListOfRecipients proton.Code = 2002

// ErrRecipientAddressDoesNotExist is an error that is returned when a recipient address could not be resolved.
type ErrRecipientAddressDoesNotExist struct {
	address string
}

func NewErrRecipientAddressDoesNotExist(address string) *ErrRecipientAddressDoesNotExist {
	return &ErrRecipientAddressDoesNotExist{address: address}
}

func (e *ErrRecipientAddressDoesNotExist) Address() string {
	if e == nil {
		return ""
	}
	return e.address
}

func (e *ErrRecipientAddressDoesNotExist) Is(target error) bool {
	_, ok := target.(*ErrRecipientAddressDoesNotExist)
	return ok
}

func (e *ErrRecipientAddressDoesNotExist) Error() string {
	return fmt.Sprintf("recipient address does not exist: %v", e.address)
}

// ErrCannotSendFromAddress is an error that is returned when a sender address could not be used.
type ErrCannotSendFromAddress struct {
	address string
}

func NewErrCannotSendFromAddress(address string) *ErrCannotSendFromAddress {
	return &ErrCannotSendFromAddress{address: address}
}

func (e *ErrCannotSendFromAddress) Error() string {
	return fmt.Sprintf("cannot send from address: %v", e.address)
}

func (e *ErrCannotSendFromAddress) Is(target error) bool {
	_, ok := target.(*ErrCannotSendFromAddress)
	return ok
}

func (e *ErrCannotSendFromAddress) Address() string {
	if e == nil {
		return ""
	}
	return e.address
}
