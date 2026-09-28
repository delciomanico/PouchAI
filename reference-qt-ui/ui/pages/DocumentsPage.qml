import QtQuick
import QtQuick.Layouts
import DocumentApp

Item {
    id: root

    property string folderId: ""
    property string folderName: ""
    property string accentGradient: "teal"

    signal backRequested()
    signal uploadRequested()

    // simple client-side filter proxy: fine at mock-data scale; a real
    // C++-backed QSortFilterProxyModel replaces this once documentModel
    // is fed from SQLite (see Performance notes in DocumentModel.h).
    property var _filtered: {
        const rows = []
        for (let i = 0; i < documentModel.count; ++i) {
            const row = documentModel.get(i)
            if (row.folderId === root.folderId)
                rows.push(row)
        }
        return rows
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacing.xl
        spacing: Theme.spacing.md

        RowLayout {
            spacing: 6
            Text { text: "Área de Trabalho"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm
                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.backRequested() } }
            Text { text: "/"; color: Theme.colors.textFainter; font.pixelSize: Theme.typography.sizeSm }
            Text { text: "Todos os Documentos"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
            Text { text: "/"; color: Theme.colors.textFainter; font.pixelSize: Theme.typography.sizeSm }
            Text { text: root.folderName; color: Theme.colors.textMuted; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacing.md

            Rectangle {
                readonly property var grad: Theme.colors.folderGradients[root.accentGradient] ?? Theme.colors.folderGradients.teal
                Layout.preferredWidth: 48; Layout.preferredHeight: 48; radius: 13
                gradient: Gradient {
                                    orientation: Gradient.Vertical
                                    GradientStop { position: 0; color: parent.grad.start }
                                    GradientStop { position: 1; color: parent.grad.end }
                                }
                ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/folder.svg"; size: 24; color: "#EAF4F1" }
            }
            ColumnLayout {
                spacing: 2
                Text { text: root.folderName; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: 24; font.weight: Theme.typography.weightExtrabold }
                Text { text: root._filtered.length + " itens"; color: Theme.colors.textFaint; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeSm }
            }
            Item { Layout.fillWidth: true }
            OutlineButton { text: "Partilhar" }
            PrimaryButton { text: "Novo Documento"; onClicked: root.uploadRequested() }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            radius: Theme.radius.lg
            color: Theme.colors.surface
            border.width: 1
            border.color: Theme.colors.border
            clip: true

            ColumnLayout {
                anchors.fill: parent
                spacing: 0

                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: 40
                    color: "#FAF8F2"
                    RowLayout {
                        anchors.fill: parent
                        anchors.leftMargin: Theme.spacing.lg
                        anchors.rightMargin: Theme.spacing.lg
                        Text { Layout.fillWidth: true; text: "NOME"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold; font.letterSpacing: 1 }
                        Text { Layout.preferredWidth: 130; text: "TIPO"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold; font.letterSpacing: 1 }
                        Text { Layout.preferredWidth: 170; text: "MODIFICADO"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold; font.letterSpacing: 1 }
                        Text { Layout.preferredWidth: 90; text: "TAMANHO"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold; font.letterSpacing: 1 }
                        Item { Layout.preferredWidth: 26 }
                    }
                }

                ListView {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    clip: true
                    model: root._filtered
                    visible: root._filtered.length > 0

                    delegate: DocumentRow {
                        width: ListView.view.width
                        columns: "folder"
                        fileName: modelData.fileName
                        documentType: modelData.documentType
                        modifiedDisplay: modelData.modifiedDisplay
                        sizeDisplay: modelData.sizeDisplay
                        accentGradient: modelData.accentGradient
                    }
                }

                EmptyState {
                    visible: root._filtered.length === 0
                    anchors.fill: parent
                    iconSource: "qrc:/DocumentApp/assets/icons/file.svg"
                    title: "Esta pasta está vazia"
                    message: "Arrasta ficheiros para aqui ou usa \"Novo Documento\" para começar."
                    actionLabel: "Novo Documento"
                    onActionTriggered: root.uploadRequested()
                }
            }
        }
    }
}
