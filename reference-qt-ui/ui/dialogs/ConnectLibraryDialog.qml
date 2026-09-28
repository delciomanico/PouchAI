import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// Lists libraries discovered via mDNS/Bonjour on the local network (mocked
// here) and lets the user connect as a client — see BackendInterface::
// connectToLibrary. Each row mirrors the product spec's "Libraries found"
// example, styled with the same tokens as everywhere else.
Dialog {
    id: root

    signal connectRequested(string hostAddress)

    property ListModel discovered: ListModel {
        ListElement { name: "Company Documents"; host: "OFFICE-PC"; address: "192.168.1.42"; available: true }
        ListElement { name: "Arquivo CASSFREI";  host: "CASSFREI-SRV"; address: "192.168.1.58"; available: true }
    }

    modal: true
    anchors.centerIn: parent
    width: 440
    padding: Theme.spacing.lg
    title: "Ligar a uma Biblioteca"

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
            text: "Bibliotecas encontradas na rede local"
            color: Theme.colors.textFaint
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeSm
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: Theme.spacing.xs

            Repeater {
                model: root.discovered
                delegate: Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 60
                    radius: Theme.radius.md
                    color: Theme.colors.surfaceSunken
                    border.width: 1
                    border.color: Theme.colors.border

                    RowLayout {
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.sm
                        spacing: Theme.spacing.sm

                        Rectangle {
                            Layout.preferredWidth: 8; Layout.preferredHeight: 8; radius: 4
                            color: available ? Theme.colors.success : Theme.colors.textFaintest
                        }
                        ColumnLayout {
                            spacing: 1
                            Text { text: name; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                            Text { text: host + " · " + address; color: Theme.colors.textFaint; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeXs }
                        }
                        Item { Layout.fillWidth: true }
                        OutlineButton {
                            text: "Ligar"
                            enabled: available
                            onClicked: { root.connectRequested(address); root.accept() }
                        }
                    }
                }
            }
        }

        RowLayout {
            Layout.topMargin: Theme.spacing.sm
            Layout.alignment: Qt.AlignRight
            OutlineButton { text: "Fechar"; onClicked: root.reject() }
        }
    }
}
