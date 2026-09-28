import QtQuick
import QtQuick.Controls.Basic
import Qt5Compat.GraphicalEffects
import DocumentApp

// Accent-filled, raised button — the frames' `.btn-3d`: a vertical gradient
// plus a soft ambient shadow tinted with the accent color.
Button {
    id: control

    property color accentColor: Theme.colors.accent
    property alias iconSource: icon.source
    property bool busy: false

    implicitHeight: 44
    implicitWidth: Math.max(120, label.implicitWidth + (icon.visible ? 34 : 0) + Theme.spacing.lg * 2)
    hoverEnabled: true

    background: Rectangle {
        id: bg
        radius: Theme.radius.md
        border.width: 1
        border.color: Qt.darker(control.accentColor, 1.25)
        gradient: Gradient {
            GradientStop { position: 0.0; color: Qt.lighter(control.accentColor, control.pressed ? 1.02 : 1.14) }
            GradientStop { position: 0.55; color: control.accentColor }
            GradientStop { position: 1.0; color: Qt.darker(control.accentColor, control.pressed ? 1.35 : 1.22) }
        }
        opacity: control.enabled ? 1.0 : 0.55

        layer.enabled: true
        layer.effect: DropShadow {
            color: Qt.rgba(control.accentColor.r, control.accentColor.g, control.accentColor.b, control.pressed ? 0.22 : 0.42)
            radius: control.pressed ? 8 : 14
            samples: 20
            verticalOffset: control.pressed ? 1 : Theme.elevation.buttonOffsetY
        }

        Behavior on opacity { NumberAnimation { duration: Theme.motion.fast } }
    }

    contentItem: Row {
        spacing: Theme.spacing.xs
        anchors.centerIn: control ? undefined : undefined
        ThemedIcon {
            id: icon
            visible: source != ""
            size: 16
            color: "#FFFFFF"
            anchors.verticalCenter: parent.verticalCenter
        }
        BusyIndicator {
            visible: control.busy
            running: control.busy
            width: 16; height: 16
            anchors.verticalCenter: parent.verticalCenter
        }
        Text {
            id: label
            text: control.text
            visible: !control.busy
            color: "#FFFFFF"
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeMd
            font.weight: Theme.typography.weightBold
            anchors.verticalCenter: parent.verticalCenter
        }
    }

    // full-bleed clickable row layout used inline (e.g. inside Row spacing)
    padding: 0
    leftPadding: Theme.spacing.lg
    rightPadding: Theme.spacing.lg
}
