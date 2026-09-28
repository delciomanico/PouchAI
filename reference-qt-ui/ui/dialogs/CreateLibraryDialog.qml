import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// "Create Library" per the product spec's network section — name,
// filesystem location, and the "make available on LAN" toggle that maps
// straight to BackendInterface.startSharing() once accepted.
Dialog {
    id: root

    signal createRequested(string name, string location, bool shareOnLan)

    modal: true
    anchors.centerIn: parent
    width: 420
    padding: Theme.spacing.lg
    title: "Criar Biblioteca"

    background: Rectangle {
        radius: Theme.radius.lg
        color: Theme.colors.surface
        border.width: 1
        border.color: Theme.colors.border
    }

    onOpened: { nameField.text = ""; locationField.text = ""; shareToggle.checked = false }

    contentItem: ColumnLayout {
        spacing: Theme.spacing.md

        Text {
            text: root.title
            color: Theme.colors.textPrimary
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeXl
            font.weight: Theme.typography.weightBold
        }

        ColumnLayout {
            spacing: 6
            Text { text: "Nome"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
            TextField {
                id: nameField
                Layout.fillWidth: true
                Layout.preferredHeight: 42
                placeholderText: "ex. Documentos da Empresa"
                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeMd
            }
        }

        ColumnLayout {
            spacing: 6
            Text { text: "Localização"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacing.xs
                TextField {
                    id: locationField
                    Layout.fillWidth: true
                    Layout.preferredHeight: 42
                    placeholderText: "~/Documentos/GestaoDocumental"
                    background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                    font.family: Theme.typography.uiFamily
                    font.pixelSize: Theme.typography.sizeMd
                }
                OutlineButton { text: "Escolher…"; implicitHeight: 42 }
            }
        }

        RowLayout {
            Layout.topMargin: Theme.spacing.xs
            spacing: Theme.spacing.sm
            ToggleSwitch { id: shareToggle }
            Text {
                text: "Disponibilizar esta biblioteca na rede local"
                color: Theme.colors.textSecondary
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeSm
            }
        }

        RowLayout {
            Layout.topMargin: Theme.spacing.sm
            Layout.alignment: Qt.AlignRight
            spacing: Theme.spacing.sm
            OutlineButton { text: "Cancelar"; onClicked: root.reject() }
            PrimaryButton {
                text: "Criar"
                onClicked: {
                    root.createRequested(nameField.text, locationField.text, shareToggle.checked)
                    root.accept()
                }
            }
        }
    }
}
