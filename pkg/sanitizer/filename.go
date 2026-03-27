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

// Package sanitizer contains common utilities to sanitize different inputs
package sanitizer

import (
	"runtime"
	"strings"
	"unicode"
)

// Filename sanitizes a filename and returns it in its appropriate form.
func Filename(input string) string {
	if input == "" {
		return ""
	}

	filename := strings.Clone(input)

	if runtime.GOOS == "windows" {
		return sanitizeFilenameWindows(filename)
	}

	return sanitizeFilenameUnix(filename)
}

func sanitizeFilenameUnix(input string) string {
	// remove trailing & leading space, dashes & dots
	input = strings.TrimLeftFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '.'
	})

	input = strings.TrimRightFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == '.' || r == '-'
	})

	illegalRunes := map[rune]bool{
		'\000': true,
		':':    true,
		127:    true,
		'/':    true,
		'\\':   true,
		'|':    true,
		'&':    true,
		'$':    true,
		'?':    true,
		'<':    true,
		'>':    true,
		';':    true,
		'"':    true,
	}
	out := removeIllegalChars(input, illegalRunes)

	if len(out) == 0 {
		return ""
	}

	result := string(out)
	sanitizedParts := make([]string, 0, strings.Count(result, ".")+1)

	// trim out space and dots inside each split part
	for part := range strings.SplitSeq(result, ".") {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		sanitizedParts = append(sanitizedParts, part)
	}

	return strings.Join(sanitizedParts, ".")
}

func sanitizeFilenameWindows(input string) string {
	// remove trailing & leading space & dots
	input = strings.TrimLeftFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == '.'
	})

	input = strings.TrimRightFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == '.'
	})

	if input == "" {
		return ""
	}

	illegalRunes := map[rune]bool{
		'<':  true,
		'>':  true,
		':':  true,
		'"':  true,
		'/':  true,
		'\\': true,
		'|':  true,
		'?':  true,
		'*':  true,
		127:  true,
	}

	out := removeIllegalChars(input, illegalRunes)
	if len(out) == 0 {
		return ""
	}

	result := string(out)
	sanitizedParts := make([]string, 0, strings.Count(result, ".")+1)

	// trim out space and dots inside each split part
	for part := range strings.SplitSeq(result, ".") {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		sanitizedParts = append(sanitizedParts, part)
	}

	reserved := map[string]int{
		"CON": 1, "PRN": 1, "AUX": 1, "NUL": 1,
		"COM1": 1, "COM2": 1, "COM3": 1, "COM4": 1, "COM5": 1, "COM6": 1, "COM7": 1, "COM8": 1, "COM9": 1,
		"LPT1": 1, "LPT2": 1, "LPT3": 1, "LPT4": 1, "LPT5": 1, "LPT6": 1, "LPT7": 1, "LPT8": 1, "LPT9": 1,
	}

	// replace reserved words with RESERVED_
	if len(sanitizedParts) > 0 && reserved[strings.ToUpper(sanitizedParts[0])] == 1 {
		sanitizedParts[0] = "RESERVED_" + sanitizedParts[0]
		sanitizedParts[0] = strings.TrimSpace(sanitizedParts[0])
		return strings.Join(sanitizedParts, ".")
	}

	return strings.Join(sanitizedParts, ".")
}

func removeIllegalChars(input string, illegalChars map[rune]bool) []rune {
	out := make([]rune, 0, len(input))
	for _, char := range input {
		if char <= 31 || !unicode.IsPrint(char) || illegalChars[char] || unicode.IsSpace(char) {
			continue
		}
		out = append(out, char)
	}
	return out
}
