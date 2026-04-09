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

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCase[T any] struct {
	name     string
	input    []T
	keep     func(T) bool
	expected []T
}

func TestFilter_Int(t *testing.T) {
	testCases := []testCase[int]{
		{
			name:  "empty",
			input: []int{},
			keep: func(_ int) bool {
				return true
			},
			expected: []int{},
		},
		{
			name:  "all",
			input: []int{1, 2, 3},
			keep: func(_ int) bool {
				return true
			},
			expected: []int{1, 2, 3},
		},
		{
			name:  "none",
			input: []int{1, 2, 3},
			keep: func(_ int) bool {
				return false
			},
			expected: []int{},
		},
		{
			name:  "only one",
			input: []int{1, 2, 3},
			keep: func(i int) bool {
				return i == 2
			},
			expected: []int{2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.input, tc.keep)

			require.Equal(t, tc.expected, result)
			require.Equal(t, len(tc.expected), len(result))
		})
	}
}

func TestFilter_String(t *testing.T) {
	testCases := []testCase[string]{
		{
			name:  "empty",
			input: []string{},
			keep: func(_ string) bool {
				return true
			},
			expected: []string{},
		},
		{
			name:  "all",
			input: []string{"a", "b", "c"},
			keep: func(_ string) bool {
				return true
			},
			expected: []string{"a", "b", "c"},
		},
		{
			name:  "none",
			input: []string{"a", "b", "c"},
			keep: func(_ string) bool {
				return false
			},
			expected: []string{},
		},
		{
			name:  "only one",
			input: []string{"a", "b", "c"},
			keep: func(s string) bool {
				return s == "b"
			},
			expected: []string{"b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.input, tc.keep)

			require.Equal(t, tc.expected, result)
			require.Equal(t, len(tc.expected), len(result))
		})
	}
}

func TestFilter_Struct(t *testing.T) {
	type testStruct struct {
		ID   int
		Name string
	}

	newRandomTestStruct := func(id int) testStruct {
		return testStruct{
			ID:   id,
			Name: fmt.Sprintf("test-%d", id),
		}
	}

	testCases := []testCase[testStruct]{
		{
			name:  "empty",
			input: []testStruct{},
			keep: func(_ testStruct) bool {
				return true
			},
			expected: []testStruct{},
		},
		{
			name:  "all",
			input: []testStruct{newRandomTestStruct(1), newRandomTestStruct(2), newRandomTestStruct(3)},
			keep: func(_ testStruct) bool {
				return true
			},
			expected: []testStruct{newRandomTestStruct(1), newRandomTestStruct(2), newRandomTestStruct(3)},
		},
		{
			name:  "none",
			input: []testStruct{newRandomTestStruct(1), newRandomTestStruct(2), newRandomTestStruct(3)},
			keep: func(_ testStruct) bool {
				return false
			},
			expected: []testStruct{},
		},
		{
			name:  "specific id",
			input: []testStruct{newRandomTestStruct(1), newRandomTestStruct(2), newRandomTestStruct(3)},
			keep: func(ts testStruct) bool {
				return ts.ID == 2
			},
			expected: []testStruct{newRandomTestStruct(2)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.input, tc.keep)

			require.Equal(t, tc.expected, result)
			require.Equal(t, len(tc.expected), len(result))
		})
	}
}
