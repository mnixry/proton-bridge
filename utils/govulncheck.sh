#!/usr/bin/env bash

# Copyright (c) 2025 Proton AG
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


set -eo pipefail

main(){
    echo "Using Go version:"
    go version
    echo

    ## go install golang.org/x/vuln/cmd/govulncheck@latest
    make gofiles
    GOTOOLCHAIN=auto go run golang.org/x/vuln/cmd/govulncheck@latest -json ./... > vulns.json

    jq -r '.finding | select( (.osv != null) and (.trace[0].function != null) ) | .osv ' < vulns.json > vulns_osv_ids.txt

    ignore GO-2023-2328 "GODT-3124 RESTY race condition"
    ignore GO-2025-3563 "BRIDGE-346 net/http request smuggling"
    ignore GO-2025-3754 "BRIDGE-388 github.com/cloudflare/circl indirect import from gopenpgp; need to wait for upstream to patch"
    ignore GO-2025-3849 "BRIDGE-416 database/sql race condition leading to potential data overwrite"
    ignore GO-2025-3956 "BRIDGE-428 LookPath from os/exec may result in binaries listed in the path to be returned"
    ignore GO-2025-4010 "BRIDGE-440 IPv6 parsing"
    ignore GO-2025-4007 "BRIDGE-440 non-linear scaling w.r.t cert chain lenght when validating chains"
    ignore GO-2025-4009 "BRIDGE-440 non-linear scaling w.r.t parsing PEM inputs"
    ignore GO-2025-4015 "BRIDGE-440 Reader.ReadResponse excessive CPU usage"
    ignore GO-2025-4008 "BRIDGE-440 ALPN negotiation failure contains attacker controlled information (not-escaped)"
    ignore GO-2025-4012 "BRIDGE-440 potentially excessive memory usage on HTTP servers via cookies"
    ignore GO-2025-4013 "BRIDGE-440 validating cert chains with DSA public keys may cause programs to panic"
    ignore GO-2025-4011 "BRIDGE-440 pasing a maliciously crafted DER payloads could allocate excessive memory"
    ignore GO-2025-4014 "BRIDGE-440 tarball extraction may read an unbounded amount of data from the archive into memory"

    has_vulns

    echo
    echo "No new vulnerabilities found."
}

ignore(){
    echo "ignoring $1 fix: $2"
    cp vulns_osv_ids.txt tmp
    grep -v "$1" < tmp > vulns_osv_ids.txt || true
    rm tmp
}

has_vulns(){
    has=false
    while read -r osv; do
        jq \
            --arg osvid "$osv" \
            '.osv | select ( .id == $osvid) | {"id":.id, "ranges": .affected[0].ranges,  "import": .affected[0].ecosystem_specific.imports[0].path}' \
            < vulns.json
        has=true
    done < vulns_osv_ids.txt

    if [ "$has" == true ]; then
        echo
        echo "Vulnerability found"
        return 1
    fi
}

main
