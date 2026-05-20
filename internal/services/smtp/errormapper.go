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
					"No email was sent. The address %s does not exist. Correct the recipient and resend the message",
					target.Address(),
				)
			}
			return errors.New("No email was sent. One or more addresses do not exist. Correct the recipients and resend the message") //nolint:revive,staticcheck //disable ST1005
		},
	),
	errmapper.NewRule(
		[]error{
			ErrSendMessageOperation,
			ErrValidationFailed,
		},
		errmapper.MatchAll,
		//nolint:revive,staticcheck //disable ST1005
		errors.New("This message couldn't be sent because the email client sent a malformed message"),
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
					"You cannot send from this address: %s. Check your email client and Bridge settings",
					target.Address(),
				)
			}

			//nolint:revive,staticcheck //disable ST1005
			return errors.New("You cannot send from this address. Check your email client and Bridge settings")
		},
	),
	errmapper.NewRule(
		[]error{ErrTooManyErrors},
		errmapper.MatchAny,
		errors.New("Too many failed send attempts. Try again later"), //nolint:revive,staticcheck //disable ST1005
	),
	errmapper.NewRule(
		[]error{ErrSenderAddressNotOwned},
		errmapper.MatchAny,
		errors.New("The sender address is not valid for this account. Check your email client and Proton settings, or choose a different sender address"), //nolint:revive,staticcheck //disable ST1005
	),
	errmapper.NewRule(
		[]error{ErrUnsupportedOutgoingMIME},
		errmapper.MatchAny,
		errors.New("This message uses an unsupported message format. Try plain text or HTML"), //nolint:revive,staticcheck //disable ST1005
	),
	errmapper.NewRule(
		[]error{ErrSMTPAuthFailed},
		errmapper.MatchAny,
		errors.New("User Authentication failed. Please check the outgoing server configuration in your mail client and try again"), //nolint:revive,staticcheck //disable ST1005
	),
	errmapper.NewRule(
		[]error{ErrNoSuchUser},
		errmapper.MatchAll,
		errors.New("The account is not available in Bridge. Verify the outgoing server settings in your mail client and try again"), //nolint:revive,staticcheck //disable ST1005
	),
	errmapper.NewRule(
		[]error{ErrInvalidRecipient, ErrInvalidReturnPath},
		errmapper.MatchAny,
		errors.New("The sender or recipient address is not valid. Review the addresses and resend the message"), //nolint:revive,staticcheck //disable ST1005,
	),
}
