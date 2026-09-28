import QtQuick
import QtQuick.Controls.Basic
import DocumentApp

Column {
    id: root
    property string message: "A carregar…"

    anchors.centerIn: parent
    spacing: Theme.spacing.md

    BusyIndicator {
        anchors.horizontalCenter: parent.horizontalCenter
        running: root.visible
        width: 36; height: 36
        palette.dark: Theme.colors.accent
    }

    Text {
        anchors.horizontalCenter: parent.horizontalCenter
        text: root.message
        color: Theme.colors.textFaint
        font.family: Theme.typography.uiFamily
        font.pixelSize: Theme.typography.sizeMd
    }
}
