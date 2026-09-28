import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

Dialog {
    id: root

    property string message: ""
    property string confirmLabel: "Confirmar"
    property bool danger: false

    modal: true
    anchors.centerIn: parent
    width: 380
    padding: Theme.spacing.lg
    closePolicy: Popup.CloseOnEscape

    background: Rectangle {
        radius: Theme.radius.lg
        color: Theme.colors.surface
        border.width: 1
        border.color: Theme.colors.border
    }

    contentItem: ColumnLayout {
        spacing: Theme.spacing.md

        Text {
            text: root.title
            color: Theme.colors.textPrimary
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeXl
            font.weight: Theme.typography.weightBold
        }
        Text {
            text: root.message
            Layout.fillWidth: true
            wrapMode: Text.WordWrap
            color: Theme.colors.textFaint
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeMd
        }

        RowLayout {
            Layout.topMargin: Theme.spacing.sm
            Layout.alignment: Qt.AlignRight
            spacing: Theme.spacing.sm

            OutlineButton {
                text: "Cancelar"
                onClicked: root.reject()
            }
            PrimaryButton {
                text: root.confirmLabel
                accentColor: root.danger ? Theme.colors.danger : Theme.colors.accent
                onClicked: root.accept()
            }
        }
    }
}
