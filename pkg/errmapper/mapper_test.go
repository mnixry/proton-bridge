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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrMapper_WithResult(t *testing.T) {
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
					NewRule(
						[]error{
							randomErr,
						},
						MatchAll,
						matchedErr,
					),
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
					NewRule(
						[]error{
							randomErr,
							invalidErr,
							failedErr,
						},
						MatchAll,
						matchedErr,
					),
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
					NewRule(
						[]error{
							randomErr,
							invalidErr,
							failedErr,
						},
						MatchAll,
						matchedErr,
					),
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
					NewRule(
						[]error{
							randomErr,
						},
						MatchAny,
						matchedErr,
					),
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
					NewRule(
						[]error{
							randomErr,
							invalidErr,
							failedErr,
							tooManyErrorErr,
						},
						MatchAny,
						matchedErr,
					),
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
					NewRule(
						[]error{
							randomErr,
							invalidErr,
							failedErr,
							tooManyErrorErr,
						},
						MatchAny,
						matchedErr,
					),
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

type dynamicError1 struct {
	message string
}

func (e *dynamicError1) Error() string {
	return fmt.Sprintf("dynamic error 1: %s", e.message)
}

func (e *dynamicError1) Is(target error) bool {
	_, ok := target.(*dynamicError1)
	return ok
}

func (e *dynamicError1) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

type dynamicError2 struct {
	message string
}

func (e *dynamicError2) Error() string {
	return fmt.Sprintf("dynamic error 2: %s", e.message)
}

func (e *dynamicError2) Is(target error) bool {
	_, ok := target.(*dynamicError2)
	return ok
}

func (e *dynamicError2) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

func TestErrMapper_WithResultFn(t *testing.T) {
	type testCase struct {
		name            string
		buildErrorChain func() error
		expectedError   error
		build           func() Service
		shouldSucceed   bool
		expectNil       bool
	}
	randomError := errors.New("random error")
	sentinelDynamicError := &dynamicError1{message: ""}
	sentinelDynamicError2 := &dynamicError2{message: ""}

	tests := []testCase{
		{
			name: "one dynamic error matches rule return result - MatchAll",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w,%w", randomError, &dynamicError1{message: "test message"})
			},
			expectedError: fmt.Errorf("result error: dynamic error 1: test message"),
			build: func() Service {
				rules := []Rule{
					NewRuleWithResultFunc(
						[]error{
							sentinelDynamicError,
							randomError,
						},
						MatchAll,
						func(err error) error {
							if target, ok := errors.AsType[*dynamicError1](err); ok {
								return fmt.Errorf("result error: %s", target.Error())
							}
							return fmt.Errorf("result error: %w", err)
						},
					),
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name: "multiple dynamic errors match rule return result - MatchAll",
			buildErrorChain: func() error {
				return fmt.Errorf(
					"wrapped error: %w,%w,%w",
					randomError,
					&dynamicError1{message: "test message"},
					&dynamicError2{message: "test message 2"},
				)
			},
			expectedError: fmt.Errorf("result error: test message, test message 2"),
			build: func() Service {
				rules := []Rule{
					NewRuleWithResultFunc(
						[]error{
							sentinelDynamicError,
							sentinelDynamicError2,
							randomError,
						},
						MatchAll,
						func(err error) error {
							var builder strings.Builder
							if target, ok := errors.AsType[*dynamicError1](err); ok {
								builder.WriteString(target.Message())
								builder.WriteString(", ")
							}
							if target, ok := errors.AsType[*dynamicError2](err); ok {
								builder.WriteString(target.Message())
							}
							if builder.Len() > 0 {
								return fmt.Errorf("result error: %s", builder.String())
							}
							return fmt.Errorf("result error: %w", err)
						},
					),
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name:          "no dynamic errors match rule return fallback - MatchAny",
			shouldSucceed: true,
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w", randomError)
			},
			expectedError: fmt.Errorf("wrapped error: %w", randomError),
			build: func() Service {
				rules := []Rule{
					NewRuleWithResultFunc(
						[]error{
							randomError,
							sentinelDynamicError,
						},
						MatchAny,
						func(err error) error {
							if target, ok := errors.AsType[*dynamicError1](err); ok {
								return fmt.Errorf("result error: %s", target.Error())
							}
							return fmt.Errorf("%w", err)
						},
					),
				}
				return New(rules)
			},
		},
		{
			name:          "one dynamic error matches rule return result - MatchAny",
			shouldSucceed: true,
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w,%w", randomError, &dynamicError1{message: "test message"})
			},
			expectedError: fmt.Errorf("result error: dynamic error 1: test message"),
			build: func() Service {
				rules := []Rule{
					NewRuleWithResultFunc(
						[]error{
							randomError,
							sentinelDynamicError,
						},
						MatchAny,
						func(err error) error {
							if target, ok := errors.AsType[*dynamicError1](err); ok {
								return fmt.Errorf("result error: %s", target.Error())
							}
							return fmt.Errorf("result error: %w", err)
						},
					),
				}
				return New(rules)
			},
		},
		{
			name: "multiple dynamic errors match rule return result - MatchAny",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped error: %w,%w,%w", randomError, &dynamicError1{message: "test message"}, &dynamicError2{message: "test message 2"})
			},
			expectedError: fmt.Errorf("result error: test message, test message 2"),
			build: func() Service {
				rules := []Rule{
					NewRuleWithResultFunc(
						[]error{
							randomError,
							sentinelDynamicError,
							sentinelDynamicError2,
						},
						MatchAny,
						func(err error) error {
							var builder strings.Builder
							if target, ok := errors.AsType[*dynamicError1](err); ok {
								builder.WriteString(target.Message())
								builder.WriteString(", ")
							}
							if target, ok := errors.AsType[*dynamicError2](err); ok {
								builder.WriteString(target.Message())
							}
							if builder.Len() > 0 {
								return fmt.Errorf("result error: %s", builder.String())
							}
							return fmt.Errorf("result error: %w", err)
						},
					),
				}
				return New(rules)
			},
			shouldSucceed: true,
		},
		{
			name: "no rule matches returns original error without calling ResultFn",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped: %w", randomError)
			},
			build: func() Service {
				unmatched := errors.New("unmatched sentinel")
				return New([]Rule{
					NewRuleWithResultFunc(
						[]error{unmatched},
						MatchAny,
						func(_ error) error {
							panic("ResultFn must not be called when no rule matches")
						},
					),
				})
			},
			shouldSucceed: false,
		},
		{
			name: "first matching rule wins when multiple rules match",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped: %w", randomError)
			},
			expectedError: errors.New("first rule"),
			build: func() Service {
				return New([]Rule{
					NewRuleWithResultFunc(
						[]error{randomError},
						MatchAny,
						func(_ error) error {
							return errors.New("first rule")
						},
					),
					NewRuleWithResultFunc(
						[]error{randomError},
						MatchAny,
						func(_ error) error {
							return errors.New("second rule")
						},
					),
				})
			},
			shouldSucceed: true,
		},
		{
			name: "ResultFn may return nil",
			buildErrorChain: func() error {
				return fmt.Errorf("wrapped: %w", randomError)
			},
			build: func() Service {
				return New([]Rule{
					NewRuleWithResultFunc(
						[]error{randomError},
						MatchAny,
						func(_ error) error {
							return nil
						},
					),
				})
			},
			expectNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mapper := tc.build()
			err := tc.buildErrorChain()

			result := mapper.Resolve(err)
			switch {
			case tc.expectNil:
				require.NoError(t, result)
			case !tc.shouldSucceed:
				require.ErrorIs(t, result, err)
			default:
				require.Equal(t, tc.expectedError.Error(), result.Error())
			}
		})
	}
}
