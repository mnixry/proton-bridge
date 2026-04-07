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

	"github.com/ProtonMail/proton-bridge/v3/pkg/errmapper"
)

//nolint:gochecknoglobals
var smtpSharedErrMapper = errmapper.New(smtpErrRules)

// mapError uses the shared error mapper to resolve a given error chain to a single error.
// Ideally called only from the SMTP server boundary so that lower layers can log the full error chain.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	return smtpSharedErrMapper.Resolve(err)
}

// Sentinel for errors.Is; any *ErrRecipientAddressDoesNotExist matches via Is().
//
//nolint:gochecknoglobals
var errRecipientAddressDoesNotExistTarget = NewErrRecipientAddressDoesNotExist("")

// Sentinel for errors.Is; any *ErrCannotSendFromAddress matches via Is().
//
//nolint:gochecknoglobals
var errCannotSendFromAddress = NewErrCannotSendFromAddress("")

//nolint:gochecknoglobals
var smtpErrRules = []errmapper.Rule{
	errmapper.NewRuleWithResultFunc(
		[]error{
			ErrSendMessageOperation,
			ErrGetRecipientsOperation,
			ErrGetSendPreferencesOperation,
			ErrLookupRecipientPublicKey,
			errRecipientAddressDoesNotExistTarget,
		},
		errmapper.MatchAll,
		func(err error) error {
			if target, ok := errors.AsType[*ErrRecipientAddressDoesNotExist](err); ok {
				//nolint:revive,staticcheck //disable ST1005,
				return fmt.Errorf(
					"The address %s does not exist. Remove or correct the recipient and try again.",
					target.Address(),
				)
			}
			return errors.New("One or more addresses do not exist. Remove or correct the recipients and try again.") //nolint:revive,staticcheck //disable ST1005,
		},
	),
	errmapper.NewRuleWithResultFunc(
		[]error{
			errCannotSendFromAddress,
		},
		errmapper.MatchAny,
		func(err error) error {
			if target, ok := errors.AsType[*ErrCannotSendFromAddress](err); ok {
				//nolint:revive,staticcheck //disable ST1005,
				return fmt.Errorf(
					"You cannot send from this address: %s. Check that it is enabled in your email client or Bridge settings.",
					target.Address(),
				)
			}
			return errors.New("You cannot send from this address. Check that it is enabled in your email client or Bridge settings.") //nolint:revive,staticcheck //disable ST1005,
		},
	),
	errmapper.NewRule(
		[]error{ErrTooManyErrors},
		errmapper.MatchAny,
		errors.New("Too many failed send attempts. Wait a moment, then try again."), //nolint:revive,staticcheck //disable ST1005,
	),
	errmapper.NewRule(
		[]error{ErrSenderAddressNotOwned},
		errmapper.MatchAny,
		errors.New("The From address is not valid for this account. Choose a different sender address."), //nolint:revive,staticcheck //disable ST1005,
	),
	errmapper.NewRule(
		[]error{ErrUnsupportedOutgoingMIME},
		errmapper.MatchAny,
		errors.New("This message uses an unsupported format. Try plain text or HTML."), //nolint:revive,staticcheck //disable ST1005,
	),
	errmapper.NewRule(
		[]error{ErrInvalidRecipient, ErrInvalidReturnPath, ErrNoSuchUser},
		errmapper.MatchAny,
		errors.New("The sender or recipient address is not valid. Check To/Cc/Bcc and try again."), //nolint:revive,staticcheck //disable ST1005,
	),
}
