import QtQuick
import QtQuick.Controls.Basic
import DocumentApp

// One item in AppTopBar's navigation: an icon + label pill that is either
// "active" (accentSoft background, accent text — the current page) or
// plain (transparent, muted text; highlights on hover).
Button {
    id: control

    property bool active: false
    property url iconSource: ""

    implicitHeight: 40
    implicitWidth: row.implicitWidth + Theme.spacing.md * 2
    hoverEnabled: true

    background: Rectangle {
        radius: Theme.radius.pill
        color: control.active
               ? Theme.colors.accentSoft
               : (control.hovered ? Qt.rgba(0, 0, 0, 0.03) : "transparent")
        Behavior on color { ColorAnimation { duration: Theme.motion.fast } }
    }

    contentItem: Row {
        id: row
        spacing: Theme.spacing.xs
        anchors.centerIn: parent

        ThemedIcon {
            source: control.iconSource
            visible: source != ""
            size: 16
            color: control.active ? Theme.colors.accent : Theme.colors.textMuted
            anchors.verticalCenter: parent.verticalCenter
        }
        Text {
            text: control.text
            color: control.active ? Theme.colors.accent : Theme.colors.textMuted
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeMd
            font.weight: control.active ? Theme.typography.weightBold : Theme.typography.weightSemibold
            anchors.verticalCenter: parent.verticalCenter
        }
    }
}
