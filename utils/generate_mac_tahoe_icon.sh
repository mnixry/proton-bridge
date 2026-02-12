#!/bin/sh

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
# along with Proton Mail Bridge. If not, see <https://www.gnu.org/licenses/>.

# Uses 'actool' to compile an asset catalog into a single binary `Assets.car` which is used primarily 
# for the new icon support in macOS26+.

ICON_PATH="./dist/Bridge.icon/"
OUTPUT_PATH="./dist"
PLIST_PATH="$OUTPUT_PATH/assetcatalog_generated_info.plist"
DEVELOPMENT_REGION="en"

actool "$ICON_PATH" --compile "$OUTPUT_PATH" \
    --output-format human-readable-text \
    --output-partial-info-plist "$PLIST_PATH" \
    --app-icon MyApp --include-all-app-icons \
    --enable-on-demand-resources NO \
    --development-region "$DEVELOPMENT_REGION" \
    --target-device mac \
    --minimum-deployment-target 26.0 \
    --platform macosx

rm "$PLIST_PATH"
