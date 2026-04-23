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
	"maps"
	"slices"
)

// Keys returns a slice of keys from the map.
// Alternative to using maps.Keys which returns an iterator instead of a slice.
func Keys[M ~map[K]V, K comparable, V any](m M) []K {
	keys := make([]K, 0, len(m))

	return slices.AppendSeq(
		keys,
		maps.Keys(m),
	)
}

// Values returns a slice of values from the map.
// Alternative to using maps.Values which returns an iterator instead of a slice.
func Values[M ~map[K]V, K comparable, V any](m M) []V {
	values := make([]V, 0, len(m))

	return slices.AppendSeq(
		values,
		maps.Values(m),
	)
}
