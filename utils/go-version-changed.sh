#!/usr/bin/env bash
#
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
#
# Compare go.mod "go" and "toolchain" directives between two revisions.
# Exit 0 if either directive differs (or base go.mod is missing). Exit 1 if both match.

set -euo pipefail

base=$CI_MERGE_REQUEST_DIFF_BASE_SHA
head=$CI_COMMIT_SHA

get_directives() {
  git show "$1:go.mod" | awk '/^go[[:space:]]+[0-9]/ || /^toolchain[[:space:]]+go/'
}

base_dirs=$(get_directives "$base")
head_dirs=$(get_directives "$head")

if [ "$base_dirs" != "$head_dirs" ]; then
  echo "Go directives changed:" >&2
  echo "  base: $base_dirs" >&2
  echo "  head: $head_dirs" >&2
  exit 0
fi
exit 1
