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
import QtQml
import QtQuick
import QtQuick.Layouts
import QtQuick.Controls
import Proton
import Notifications

Dialog {
    id: root

    default property alias data: additionalChildrenContainer.children
    property var notification
    property bool isUserNotification: false

    // Placeholder for text input label text.
    property string textFieldText: ""

    modal: true
    shouldShow: notification && notification.active && !notification.dismissed

    ColumnLayout {
        spacing: 0

        Image {
            Layout.alignment: Qt.AlignHCenter
            Layout.bottomMargin: 16
            Layout.preferredHeight: 64
            Layout.preferredWidth: 64
            source: {
                if (!root.notification) {
                    return "";
                }
                switch (root.notification.type) {
                    case Notification.NotificationType.Info:
                        return "/qml/icons/ic-info.svg";
                    case Notification.NotificationType.Success:
                        return "/qml/icons/ic-success.svg";
                    case Notification.NotificationType.Warning:
                    case Notification.NotificationType.Danger:
                        return "/qml/icons/ic-alert.svg";
                }
            }
            sourceSize.height: 64
            sourceSize.width: 64
            visible: source != ""
        }

        Label {
            Layout.alignment: Qt.AlignHCenter
            Layout.bottomMargin: 8
            colorScheme: root.colorScheme
            horizontalAlignment: Text.AlignHCenter
            text: root.notification.title
            type: Label.LabelType.Title
        }
        Label {
            Layout.bottomMargin: 16
            Layout.fillWidth: true
            Layout.preferredWidth: 240
            colorScheme: root.colorScheme
            horizontalAlignment: Text.AlignHCenter
            text: root.notification.description
            type: Label.LabelType.Body
            wrapMode: Text.WordWrap

            onLinkActivated: function (link) {
                Backend.openExternalLink(link);
            }
        }
        Item {
            id: additionalChildrenContainer
            Layout.bottomMargin: 16
            Layout.fillWidth: true
            implicitHeight: additionalChildrenContainer.childrenRect.height
            implicitWidth: additionalChildrenContainer.childrenRect.width
            visible: children.length > 0
        }

        Image {
            Layout.alignment: Qt.AlignHCenter
            Layout.bottomMargin: 16
            Layout.preferredHeight: 64
            Layout.preferredWidth: 64
            source: root.notification.additionalImageSrc
            sourceSize.height: 64
            sourceSize.width: 64
            visible: root.notification.additionalImageSrc != ""
        }

        TextField {
            id: textField
            Layout.fillWidth: true
            Layout.preferredWidth: 240
            Layout.bottomMargin: 16
            colorScheme: root.colorScheme
            echoMode: root.notification.isTextFieldPassword ? TextInput.Password : TextInput.Normal
            text: root.textFieldText
            visible: root.notification && root.notification.useTextField

            onTextChanged: root.notification.textFieldChanged(text)

            Connections {
                target: root.notification
                function onClearTextFieldRequested() {
                    root.notification.textFieldChanged("")
                    textField.clear();
                }
                function onFocusTextField() {
                    textField.focus = true;
                }
                function onHideTextFieldPassword() {
                    if (root.notification.isTextFieldPassword) {
                        textField.hidePassword();
                    }
                }
            }
        }


        LinkLabel {
            Layout.alignment: Qt.AlignHCenter
            Layout.bottomMargin: 32
            colorScheme: root.colorScheme
            external: true
            link: notification.linkUrl
            text: notification.linkText
            visible: notification.linkUrl.length > 0

        }

        Spinner {
            Layout.alignment: Qt.AlignHCenter
            colorScheme: root.colorScheme
            Layout.bottomMargin: 16
            Layout.preferredHeight: 64
            Layout.preferredWidth: 64
            size: 64
            running: true
            visible: root.notification && root.notification.busyIndicator
        }


        ColumnLayout {
            spacing: 8

            Repeater {
                model: root.notification.action

                delegate: Button {
                    Layout.fillWidth: true
                    action: modelData
                    colorScheme: root.colorScheme
                    loading: modelData.loading
                    secondary: modelData.forceSecondary !== undefined ? modelData.forceSecondary : index > 0                }
            }
        }
    }
}
