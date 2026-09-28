import QtQuick
import QtQuick.Layouts
import DocumentApp

Item {
    id: root

    signal folderOpened(string folderId, string folderName)
    signal newFolderRequested()

    RowLayout {
        anchors.fill: parent
        spacing: 0

        // --- main content -------------------------------------------------------
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.margins: Theme.spacing.xl
            spacing: Theme.spacing.lg

            RowLayout {
                Layout.fillWidth: true
                ColumnLayout {
                    spacing: 4
                    Text { text: "ÁREA DE TRABALHO"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold; font.letterSpacing: 1 }
                    Text { text: "Todos os Documentos"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeDisplay - 2; font.weight: Theme.typography.weightExtrabold }
                }
                Item { Layout.fillWidth: true }
                PrimaryButton {
                    text: "Nova pasta"
                    implicitHeight: 36
                    onClicked: root.newFolderRequested()
                }
                Rectangle {
                    Layout.preferredWidth: 116; Layout.preferredHeight: 36; radius: Theme.radius.md
                    color: Theme.colors.surfaceSunken
                    RowLayout {
                        anchors.centerIn: parent; spacing: 2
                        ThemedIcon { source: "qrc:/DocumentApp/assets/icons/checklist.svg"; size: 15; color: Theme.colors.textFaint }
                        Rectangle { width: 28; height: 28; radius: 7; color: Theme.colors.accentSoft
                            ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/grid.svg"; size: 15; color: Theme.colors.accent } }
                        ThemedIcon { source: "qrc:/DocumentApp/assets/icons/list.svg"; size: 15; color: Theme.colors.textFaint }
                    }
                }
            }

            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true

                GridView {
                    id: grid
                    anchors.fill: parent
                    visible: folderModel.count > 0
                    cellWidth: (width - Theme.spacing.md) / 2
                    cellHeight: (height - Theme.spacing.md * 2) / 3
                    clip: true
                    model: folderModel

                    delegate: Item {
                        width: grid.cellWidth - Theme.spacing.md
                        height: grid.cellHeight - Theme.spacing.md

                        DocumentCard {
                            anchors.fill: parent
                            folderId: model.id
                            name: model.name
                            itemCount: model.itemCount
                            modifiedDisplay: model.modifiedDisplay
                            accentGradient: model.accentGradient
                            shared: model.isShared
                            sharedWith: model.sharedWith
                            onClicked: root.folderOpened(model.id, model.name)
                        }
                    }
                }

                EmptyState {
                    visible: folderModel.count === 0
                    iconSource: "qrc:/DocumentApp/assets/icons/folder.svg"
                    title: "Ainda não tens pastas"
                    message: "Cria a tua primeira pasta para começar a organizar os documentos do grupo."
                    actionLabel: "Nova pasta"
                    onActionTriggered: root.newFolderRequested()
                }
            }
        }

        AIChatPanel {
            id: aiChatPanel
            Layout.preferredWidth: 336
            Layout.fillHeight: true
            onAskRequested: (text) => {
                // Wire to backend once AI Q&A is live; mocked reply for now.
                aiChatPanel.addReply("Ainda não tenho acesso ao motor de IA — isto é só a interface.")
            }
        }
    }
}
