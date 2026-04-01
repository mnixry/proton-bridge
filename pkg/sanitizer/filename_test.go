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

package sanitizer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeFilename_Windows(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "Reserved device name - 1",
			filename: "CON",
			want:     "RESERVED_CON",
		},
		{
			name:     "Reserved device name - 2",
			filename: "PRN.txt",
			want:     "RESERVED_PRN.txt",
		},
		{
			name:     "Reserved device name - 3",
			filename: "AUX.png",
			want:     "RESERVED_AUX.png",
		},
		{
			name:     "Reserved device name - 4",
			filename: "NUL.log",
			want:     "RESERVED_NUL.log",
		},
		{
			name:     "Reserved device name - 5",
			filename: "COM1.txt",
			want:     "RESERVED_COM1.txt",
		},
		{
			name:     "Illegal characters - 1",
			filename: "bad<name>.txt",
			want:     "badname.txt",
		},
		{
			name:     "Illegal characters - 2",
			filename: "pipe|file.doc",
			want:     "pipefile.doc",
		},
		{
			name:     "Illegal characters - 3",
			filename: "question?.txt",
			want:     "question.txt",
		},
		{
			name:     "Illegal characters - 4",
			filename: "star*.log",
			want:     "star.log",
		},
		{
			name:     "Illegal characters - 5",
			filename: "quote\".exe",
			want:     "quote.exe",
		},
		{
			name:     "Illegal characters - 6",
			filename: "backslash\\file.exe",
			want:     "backslashfile.exe",
		},
		{
			name:     "Illegal characters - 7",
			filename: "color:file.txt",
			want:     "colorfile.txt",
		},
		{
			name:     "Illegal characters - 8",
			filename: "slash/file.txt",
			want:     "slashfile.txt",
		},
		{
			name:     "Trailing dot-space - 1",
			filename: "file.txt ",
			want:     "file.txt",
		},
		{
			name:     "Trailing dot-space - 2",
			filename: "file.txt .",
			want:     "file.txt",
		},
		{
			name:     "Trailing dot-space - 3",
			filename: "trailingdot.",
			want:     "trailingdot",
		},
		{
			name:     "Leading whitespace - 1",
			filename: " file.txt",
			want:     "file.txt",
		},
		{
			name:     "Leading whitespace - 2",
			filename: "  file.txt ",
			want:     "file.txt",
		},
		{
			name:     "Unicode - 1 ",
			filename: "🤔.txt",
			want:     "🤔.txt",
		},
		{
			name:     "Path traversal - 1",
			filename: `..\..\windows\system32\cmd.exe`,
			want:     "windowssystem32cmd.exe",
		},
		{
			name:     "Path traversal - 2",
			filename: "../../../../etc/passwd",
			want:     "etcpasswd",
		},
		{
			name:     "Reserved name + space",
			filename: "CON .txt",
			want:     "RESERVED_CON.txt",
		},
		{
			name:     "Multiple dots",
			filename: "file.. .txt",
			want:     "file.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFilenameWindows(tc.filename)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSanitizeFilename_Unix(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "Illegal characters - 1",
			filename: "file/slash/name.bob",
			want:     "fileslashname.bob",
		},
		{
			name:     "Illegal characters - 2",
			filename: "test\000byte.log",
			want:     "testbyte.log",
		},
		{
			name:     "Leading dash - 1",
			filename: "--file.txt",
			want:     "file.txt",
		},
		{
			name:     "Leading dash - 2",
			filename: "--help",
			want:     "help",
		},
		{
			name:     "Illegal characters - 1",
			filename: "bad<name>.txt",
			want:     "badname.txt",
		},
		{
			name:     "Illegal characters - 2",
			filename: "pipe|file.doc",
			want:     "pipefile.doc",
		},
		{
			name:     "Illegal characters - 3",
			filename: "question?.txt",
			want:     "question.txt",
		},
		{
			name:     "Illegal characters - 4",
			filename: "star*.log",
			want:     "star*.log",
		},
		{
			name:     "Illegal characters - 5",
			filename: "quote\".exe",
			want:     "quote.exe",
		},
		{
			name:     "Illegal characters - 6",
			filename: "backslash\\file.exe",
			want:     "backslashfile.exe",
		},
		{
			name:     "Illegal characters - 7",
			filename: "color:file.txt",
			want:     "colorfile.txt",
		},
		{
			name:     "Illegal characters - 8",
			filename: "slash/file.txt",
			want:     "slashfile.txt",
		},
		{
			name:     "Trailing dot-space - 1",
			filename: "file.txt ",
			want:     "file.txt",
		},
		{
			name:     "Trailing dot-space - 2",
			filename: "file.txt .",
			want:     "file.txt",
		},
		{
			name:     "Trailing dot-space - 3",
			filename: "trailingdot.",
			want:     "trailingdot",
		},
		{
			name:     "Leading whitespace - 1",
			filename: " file.txt",
			want:     "file.txt",
		},
		{
			name:     "Leading whitespace - 2",
			filename: "  file.txt ",
			want:     "file.txt",
		},
		{
			name:     "Unicode - 1 ",
			filename: "🤔.txt",
			want:     "🤔.txt",
		},
		{
			name:     "Path traversal - 1",
			filename: `../../var/program/bash`,
			want:     "varprogrambash",
		},
		{
			name:     "Path traversal - 2",
			filename: "../../../../etc/passwd",
			want:     "etcpasswd",
		},
		{
			name:     "Path traversal - 3",
			filename: `..\/..\/file.txt`,
			want:     "file.txt",
		},
		{
			name:     "Shell commands - 1",
			filename: "file;rm -rf *.txt",
			want:     "filerm-rf*.txt",
		},
		{
			name:     "Shell commands - 2",
			filename: "file&&echo exec.txt",
			want:     "fileechoexec.txt",
		},
		{
			name:     "Shell commands - 3",
			filename: "file || echo .env",
			want:     "fileecho.env",
		},
		{
			name:     "Shell commands - 4",
			filename: "file$(whoami).log",
			want:     "file(whoami).log",
		},
		{
			name:     "Reserved name + space",
			filename: "file .txt",
			want:     "file.txt",
		},
		{
			name:     "Multiple dots",
			filename: "file.. .txt",
			want:     "file.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeFilenameUnix(tc.filename)
			require.Equal(t, tc.want, got)
		})
	}
}
