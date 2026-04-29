#!/usr/bin/env bash

# Copyright (c) 2026 Proton AG
#
# This file is part of Proton Mail Bridge.
#
# Proton Mail Bridge is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Proton Mail Bridge is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Proton Mail Bridge.  If not, see <https://www.gnu.org/licenses/>.

set -euo pipefail

run_gofix() {
	make gofix
}

if [ "${GOFIX_FORCE:-}" = "1" ] || [ "${CI_JOB_MANUAL:-}" = "true" ]; then
	run_gofix
elif ./utils/go-version-changed.sh; then
	run_gofix
else
	echo "Skipping gofix: go.mod changed but go/toolchain directives unchainged."
	: >"gofix-$(go env GOOS).log"
	exit 0
fi
