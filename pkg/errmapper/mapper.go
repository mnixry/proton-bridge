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

	"github.com/sirupsen/logrus"
)

type errorMapper struct {
	rules []Rule
	log   *logrus.Entry
}

func New(rules []Rule) Service {
	log := logrus.WithFields(logrus.Fields{
		"pkg":     "bridge/errmapper",
		"service": "errmapper",
	})
	return &errorMapper{
		rules: rules,
		log:   log,
	}
}

func (em *errorMapper) Resolve(err error) error {
	if err == nil {
		return nil
	}
	for _, rule := range em.rules {
		if em.match(err, rule) {
			em.log.WithFields(logrus.Fields{
				"rule":   rule,
				"error":  err,
				"result": rule.Result,
			}).Debug("Given error matches rule, returning result")
			return rule.Result
		}
	}

	em.log.WithFields(logrus.Fields{
		"error": err,
	}).Debug("Given error does not match any rule, returning original error")

	return err
}

func (em *errorMapper) match(err error, rule Rule) bool {
	switch rule.MatchType {
	case MatchAny:
		for _, target := range rule.Targets {
			if errors.Is(err, target) {
				return true
			}
		}
		return false
	case MatchAll:
		for _, target := range rule.Targets {
			if !errors.Is(err, target) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
