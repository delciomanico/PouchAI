import QtQuick
import DocumentApp

// A small pill used everywhere for compact status/meta info: OCR
// confidence badges, the mono "GN" group avatar, status pills
// (Pendente/Aprovado/Revisão necessária). Purely presentational.
Rectangle {
    id: root

    property string text: ""
    property color tint: Theme.colors.textFaint
    property color tintSoft: "transparent"
    property bool mono: true
    property int pixelSize: Theme.typography.sizeXs

    radius: Theme.radius.pill
    color: tintSoft === "transparent" ? Qt.rgba(tint.r, tint.g, tint.b, 0.12) : tintSoft
    implicitWidth: label.implicitWidth + Theme.spacing.sm * 2
    implicitHeight: label.implicitHeight + Theme.spacing.xxs * 2

    Text {
        id: label
        anchors.centerIn: parent
        text: root.text
        color: root.tint
        font.family: root.mono ? Theme.typography.monoFamily : Theme.typography.uiFamily
        font.pixelSize: root.pixelSize
        font.weight: Theme.typography.weightBold
    }
}
