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

package errmapper

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrMapper(t *testing.T) {
	randomErr := errors.New("random error")
	invalidErr := errors.New("invalid error")
	failedErr := errors.New("failed error")
	tooManyErrorErr := errors.New("too many errors")

	matchedErr := errors.New("matched error")

	tests := []struct {
		name            string
		buildErrorChain func() error
		expectedError   error
		build           func() Service
		shouldSucceed   bool
	}{
		{
			name: "single error matches rule - MatchAll",

			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w", randomErr)
			},
			expectedError: matchedErr,
			build: func() Service {
				rules := []Rule{
					{
						Targets: []error{
							randomErr,
						},
						MatchType: MatchAll,
						Result:    matchedErr,
					},
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name: "multiple errors match rule - MatchAll",
			buildErrorChain: func() error {
				err := errors.Join(randomErr, invalidErr, failedErr)
				wrapped := fmt.Errorf("wrapped error: %w", err)
				return fmt.Errorf("wrapped error: %w", wrapped)
			},
			expectedError: matchedErr,
			build: func() Service {
				rules := []Rule{
					{
						Targets: []error{
							randomErr,
							invalidErr,
							failedErr,
						},
						MatchType: MatchAll,
						Result:    matchedErr,
					},
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name: "no errors match rule - MatchAll",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w", randomErr)
			},
			expectedError: randomErr,
			build: func() Service {
				rules := []Rule{
					{
						Targets: []error{
							randomErr,
							invalidErr,
							failedErr,
						},
						MatchType: MatchAll,
						Result:    matchedErr,
					},
				}
				return New(rules)
			},
			shouldSucceed: false,
		},
		{
			name: "single error matches rule - MatchAny",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w", randomErr)
			},
			expectedError: matchedErr,
			build: func() Service {
				rules := []Rule{
					{
						Targets: []error{
							randomErr,
						},
						MatchType: MatchAny,
						Result:    matchedErr,
					},
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name: "multiple errors match rule - MatchAny",
			buildErrorChain: func() error {
				err := errors.Join(randomErr, invalidErr, failedErr, tooManyErrorErr)
				wrapped := fmt.Errorf("wrapped error: %w", err)
				return fmt.Errorf("wrapped error: %w", wrapped)
			},
			expectedError: matchedErr,
			build: func() Service {
				rules := []Rule{
					{
						Targets: []error{
							randomErr,
							invalidErr,
							failedErr,
							tooManyErrorErr,
						},
						MatchType: MatchAny,
						Result:    matchedErr,
					},
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name: "no errors match rule - MatchAny",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w", errors.New("random new error"))
			},
			expectedError: matchedErr,
			build: func() Service {
				rules := []Rule{
					{
						Targets: []error{
							randomErr,
							invalidErr,
							failedErr,
							tooManyErrorErr,
						},
						MatchType: MatchAny,
						Result:    matchedErr,
					},
				}
				return New(rules)
			},
			shouldSucceed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mapper := tc.build()
			err := tc.buildErrorChain()

			result := mapper.Resolve(err)
			if !tc.shouldSucceed {
				require.ErrorIs(t, result, err)
			} else {
				require.ErrorIs(t, result, tc.expectedError)
			}
		})
	}
}
