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

// Rule holds information about a rule to be applied to a given error or error chain.
type Rule struct {
	Targets   []error
	MatchType MatchType
	Result    error
}

// NewRule returns a new Rule instance.
func NewRule(targets []error, matchType MatchType, result error) Rule {
	return Rule{
		Targets:   targets,
		MatchType: matchType,
		Result:    result,
	}
}
