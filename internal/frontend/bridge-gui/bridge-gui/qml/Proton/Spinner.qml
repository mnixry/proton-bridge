// Copyright (c) 2026 Proton AG
// This file is part of Proton Mail Bridge.
// Proton Mail Bridge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// Proton Mail Bridge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// You should have received a copy of the GNU General Public License
// along with Proton Mail Bridge. If not, see <https://www.gnu.org/licenses/>.

import QtQuick
import QtQuick.Controls.impl

Item {
    id: root

    property ColorScheme colorScheme: ProtonStyle.currentStyle
    property color color: colorScheme.interaction_norm
    property int size: 16
    property bool running: true
    property int duration: 1000
    property string source: "/qml/icons/Loader_48.svg"

    implicitWidth: size
    implicitHeight: size

    ColorImage {
        id: spinnerImage
        anchors.centerIn: parent
        width: root.size
        height: root.size
        source: root.source
        color: root.color
        sourceSize.width: root.size
        sourceSize.height: root.size
        visible: root.running

        RotationAnimation {
            target: spinnerImage
            property: "rotation"
            from: 0
            to: 360
            duration: root.duration
            loops: Animation.Infinite
            running: root.running
            direction: RotationAnimation.Clockwise
        }
    }
}