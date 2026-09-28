import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Dialogs
import QtQuick.Layouts
import DocumentApp

Item {
    id: root

    property int mode: 0 // 0 Individual, 1 Em Massa
    property string activeDocumentId: ""

    signal approved(string documentId)
    signal rejected(string documentId)

    // rows currently awaiting review or still processing, across the library
    property var _reviewRows: {
        const rows = []
        for (let i = 0; i < documentModel.count; ++i) {
            const row = documentModel.get(i)
            if (row.status === "pendingReview" || row.status === "processing" || row.status === "uploading")
                rows.push(row)
        }
        return rows
    }

    readonly property var _activeDoc: {
        for (let i = 0; i < documentModel.count; ++i) {
            const row = documentModel.get(i)
            if (row.id === root.activeDocumentId)
                return row
        }
        return root._reviewRows.length > 0 ? root._reviewRows[0] : null
    }

    FileDialog {
        id: fileDialog
        title: "Escolher ficheiro"
        fileMode: FileDialog.OpenFiles
        nameFilters: ["Documentos (*.pdf *.png *.jpg *.jpeg)"]
        onAccepted: {
            for (const url of selectedFiles)
                backend.ingestFile(url.toString(), "financeiro")
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacing.xl
        spacing: Theme.spacing.lg

        RowLayout {
            Layout.fillWidth: true
            ColumnLayout {
                spacing: 4
                Text { text: "ÁREA DE TRABALHO / NOVO DOCUMENTO"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                Text { text: root.mode === 0 ? "Adicionar Documento" : "Adicionar Documentos"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeDisplay - 2; font.weight: Theme.typography.weightExtrabold }
            }
            Item { Layout.fillWidth: true }

            Rectangle {
                Layout.preferredWidth: 220; Layout.preferredHeight: 44; radius: Theme.radius.md
                color: Theme.colors.surfaceSunken
                RowLayout {
                    anchors.fill: parent
                    anchors.margins: 4
                    spacing: 2
                    Rectangle {
                        Layout.fillWidth: true; Layout.fillHeight: true; radius: Theme.radius.sm
                        color: root.mode === 0 ? Theme.colors.accentSoft : "transparent"
                        Text { anchors.centerIn: parent; text: "Individual"; color: root.mode === 0 ? Theme.colors.accent : Theme.colors.textMuted; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.mode = 0 }
                    }
                    Rectangle {
                        Layout.fillWidth: true; Layout.fillHeight: true; radius: Theme.radius.sm
                        color: root.mode === 1 ? Theme.colors.accentSoft : "transparent"
                        Text { anchors.centerIn: parent; text: "Em Massa"; color: root.mode === 1 ? Theme.colors.accent : Theme.colors.textMuted; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.mode = 1 }
                    }
                }
            }
        }

        Text {
            Layout.fillWidth: true
            text: root.mode === 0
                  ? "O OCR lê o ficheiro e sugere a classificação — tu revês e aprovas antes de guardar."
                  : "O OCR classifica cada ficheiro automaticamente — revê e aprova em lote, ou um a um."
            color: Theme.colors.textFaint
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeMd
        }

        // ---------------------------------------------------------- Individual
        RowLayout {
            visible: root.mode === 0
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: Theme.spacing.lg

            ColumnLayout {
                Layout.preferredWidth: 420
                Layout.fillHeight: true
                spacing: Theme.spacing.md

                FileDropZone {
                    Layout.fillWidth: true
                    onBrowseRequested: fileDialog.open()
                    onFilesDropped: (urls) => { for (const u of urls) backend.ingestFile(u.toString(), "financeiro") }
                }

                Rectangle {
                    visible: root._activeDoc !== null
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: Theme.radius.lg
                    color: Theme.colors.surface
                    border.width: 1
                    border.color: Theme.colors.border

                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.md
                        spacing: Theme.spacing.md

                        RowLayout {
                            spacing: Theme.spacing.sm
                            Rectangle {
                                readonly property var grad: root._activeDoc ? (Theme.colors.folderGradients[root._activeDoc.accentGradient] ?? Theme.colors.folderGradients.blue) : Theme.colors.folderGradients.blue
                                Layout.preferredWidth: 40; Layout.preferredHeight: 40; radius: 10
                                gradient: Gradient {
                                    orientation: Gradient.Vertical
                                    GradientStop { position: 0; color: parent.grad.start }
                                    GradientStop { position: 1; color: parent.grad.end }
                                }
                                ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/file.svg"; size: 19; color: "#E8F0FB" }
                            }
                            ColumnLayout {
                                spacing: 1
                                Text { text: root._activeDoc ? root._activeDoc.fileName : ""; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold; elide: Text.ElideRight }
                                Text { text: root._activeDoc ? root._activeDoc.sizeDisplay : ""; color: Theme.colors.textFainter; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeXs }
                            }
                        }

                        ProcessingIndicator {
                            Layout.fillWidth: true
                            status: root._activeDoc ? root._activeDoc.status : "processing"
                            confidence: root._activeDoc ? root._activeDoc.ocrConfidence : 0
                        }

                        ColumnLayout {
                            spacing: Theme.spacing.xs
                            Text { text: "TEXTO EXTRAÍDO"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold; font.letterSpacing: 1 }
                            Rectangle {
                                Layout.fillWidth: true
                                Layout.preferredHeight: 90
                                radius: Theme.radius.md
                                color: Theme.colors.surfaceSunken
                                Text {
                                    anchors.fill: parent
                                    anchors.margins: Theme.spacing.sm
                                    text: root._activeDoc ? root._activeDoc.ocrExcerpt : ""
                                    wrapMode: Text.WordWrap
                                    color: Theme.colors.textMuted
                                    font.family: Theme.typography.monoFamily
                                    font.pixelSize: 11
                                    lineHeight: 1.5
                                }
                            }
                        }

                        Item { Layout.fillHeight: true }
                    }
                }
            }

            // --- classification review panel -------------------------------------
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                radius: Theme.radius.xl
                color: Theme.colors.surface
                border.width: 1
                border.color: Theme.colors.border
                visible: root._activeDoc !== null

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: Theme.spacing.lg
                    spacing: Theme.spacing.md

                    RowLayout {
                        spacing: Theme.spacing.sm
                        Rectangle {
                            Layout.preferredWidth: 30; Layout.preferredHeight: 30; radius: 9
                            color: Theme.colors.accentSoft
                            ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/sparkles.svg"; size: 16; color: Theme.colors.accent }
                        }
                        ColumnLayout {
                            spacing: 1
                            Text { text: "Classificação sugerida pela IA"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }
                            Text { text: "Revê os campos antes de aprovar"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                        }
                    }

                    ColumnLayout {
                        spacing: 6
                        Text { text: "Nome do documento"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                        TextField {
                            Layout.fillWidth: true; Layout.preferredHeight: 44
                            text: root._activeDoc ? root._activeDoc.fileName : ""
                            background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                            leftPadding: Theme.spacing.md
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeMd
                        }
                    }

                    RowLayout {
                        spacing: Theme.spacing.md
                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 6
                            Text { text: "Tipo de documento"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            Rectangle {
                                Layout.fillWidth: true; Layout.preferredHeight: 44; radius: Theme.radius.md; color: Theme.colors.surfaceSunken
                                RowLayout {
                                    anchors.fill: parent; anchors.margins: Theme.spacing.sm
                                    Text { Layout.fillWidth: true; text: root._activeDoc ? root._activeDoc.documentType : ""; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                                    StatusBadge { kind: "confidence"; confidence: root._activeDoc ? root._activeDoc.ocrConfidence : 0 }
                                }
                            }
                        }
                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 6
                            Text { text: "Pasta de destino"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            Rectangle {
                                Layout.fillWidth: true; Layout.preferredHeight: 44; radius: Theme.radius.md; color: Theme.colors.surfaceSunken
                                RowLayout {
                                    anchors.fill: parent; anchors.margins: Theme.spacing.sm
                                    Text { Layout.fillWidth: true; text: root._activeDoc ? root._activeDoc.folderName : ""; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                                    ThemedIcon { source: "qrc:/DocumentApp/assets/icons/chevron-down.svg"; size: 14; color: Theme.colors.textFaint }
                                }
                            }
                        }
                    }

                    ColumnLayout {
                        spacing: 6
                        Text { text: "Etiquetas"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                        Flow {
                            Layout.fillWidth: true
                            spacing: Theme.spacing.xs
                            Repeater {
                                model: root._activeDoc ? root._activeDoc.tags : []
                                delegate: IconChip { text: modelData; tint: Theme.colors.textSecondary; tintSoft: "#F3F0E7"; mono: false }
                            }
                            IconChip { text: "+ Adicionar"; tint: Theme.colors.textFainter; tintSoft: Theme.colors.surfaceSunken; mono: false }
                        }
                    }

                    Item { Layout.fillHeight: true }

                    RowLayout {
                        spacing: Theme.spacing.sm
                        OutlineButton {
                            text: "Rejeitar"
                            tone: OutlineButton.Tone.Danger
                            onClicked: { if (root._activeDoc) { backend.rejectDocument(root._activeDoc.id); root.rejected(root._activeDoc.id) } }
                        }
                        PrimaryButton {
                            Layout.fillWidth: true
                            text: "Aprovar e Guardar"
                            onClicked: { if (root._activeDoc) { backend.approveDocument(root._activeDoc.id); root.approved(root._activeDoc.id) } }
                        }
                    }
                }
            }

            EmptyState {
                visible: root._activeDoc === null
                Layout.fillWidth: true
                Layout.fillHeight: true
                iconSource: "qrc:/DocumentApp/assets/icons/upload.svg"
                title: "Nenhum ficheiro em revisão"
                message: "Carrega um documento à esquerda para veres aqui a classificação sugerida pela IA."
            }
        }

        // ------------------------------------------------------------ Em Massa
        ColumnLayout {
            visible: root.mode === 1
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: Theme.spacing.md

            FileDropZone {
                Layout.fillWidth: true
                compact: true
                title: "Arrasta vários ficheiros aqui ou clica para escolher"
                onBrowseRequested: fileDialog.open()
                onFilesDropped: (urls) => { for (const u of urls) backend.ingestFile(u.toString(), "financeiro") }
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacing.sm
                Text { text: root._reviewRows.length + " ficheiros em revisão"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                Item { Layout.fillWidth: true }
                OutlineButton { text: "Rejeitar Selecionados"; tone: OutlineButton.Tone.Danger }
                PrimaryButton { text: "Aprovar Selecionados" }
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
                            Item { Layout.preferredWidth: 17 }
                            Text { Layout.fillWidth: true; text: "FICHEIRO"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                            Text { Layout.preferredWidth: 190; text: "CLASSIFICAÇÃO SUGERIDA"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                            Text { Layout.preferredWidth: 60; text: "CONF."; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                            Text { Layout.preferredWidth: 140; text: "ESTADO"; color: Theme.colors.textFaint; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                            Item { Layout.preferredWidth: 26 }
                        }
                    }

                    ListView {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        clip: true
                        model: root._reviewRows
                        visible: root._reviewRows.length > 0

                        delegate: DocumentRow {
                            width: ListView.view.width
                            columns: "review"
                            selectable: true
                            fileName: modelData.fileName
                            classificationLabel: modelData.folderName + " / " + modelData.documentType
                            confidence: modelData.ocrConfidence
                            status: modelData.status
                            accentGradient: modelData.accentGradient
                        }
                    }

                    EmptyState {
                        visible: root._reviewRows.length === 0
                        anchors.fill: parent
                        iconSource: "qrc:/DocumentApp/assets/icons/upload.svg"
                        title: "Sem ficheiros em revisão"
                        message: "Carrega vários documentos para os classificar e aprovar em lote."
                    }
                }
            }
        }
    }
}
