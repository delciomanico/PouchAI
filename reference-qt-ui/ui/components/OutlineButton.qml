import QtQuick
import QtQuick.Controls.Basic
import Qt5Compat.GraphicalEffects
import DocumentApp

// The frames' `.btn-outline-3d`: white-to-off-white gradient, subtle
// border and a soft ambient shadow. Optional `tone` recolors the label
// for destructive actions ("Rejeitar") without a whole new component.
Button {
    id: control

    enum Tone { Neutral, Danger }
    property int tone: OutlineButton.Tone.Neutral

    implicitHeight: 40
    implicitWidth: Math.max(90, label.implicitWidth + Theme.spacing.lg * 2)
    hoverEnabled: true

    background: Rectangle {
        radius: Theme.radius.md
        border.width: 1
        border.color: Theme.colors.borderStrong
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#FFFFFF" }
            GradientStop { position: 1.0; color: control.pressed ? "#EDE9DD" : "#F6F4EC" }
        }

        layer.enabled: true
        layer.effect: DropShadow {
            color: Theme.elevation.shadowColor
            radius: 10
            samples: 16
            verticalOffset: 3
        }
    }

    contentItem: Text {
        id: label
        text: control.text
        color: control.tone === OutlineButton.Tone.Danger ? Theme.colors.danger : Theme.colors.textSecondary
        font.family: Theme.typography.uiFamily
        font.pixelSize: Theme.typography.sizeMd
        font.weight: Theme.typography.weightBold
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
    }
}
