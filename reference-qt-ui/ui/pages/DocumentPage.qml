import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// ---------------------------------------------------------------------------
// DocumentPage
//
// No frame exists yet for the document viewer, so this is a structural
// placeholder rather than a pixel-accurate reproduction: it follows the
// same design tokens and component set as every other page, and lays out
// the regions the product spec calls for (content / metadata / AI
// insights / citations / versions) so a real PDF/image viewer and the
// AI panel can be dropped in later without reshaping the page.
//
// Worth designing a dedicated frame for this before it goes further.
// ---------------------------------------------------------------------------
Item {
    id: root

    property string documentId: ""
    readonly property var doc: {
        for (let i = 0; i < documentModel.count; ++i) {
            const row = documentModel.get(i)
            if (row.id === root.documentId)
                return row
        }
        return null
    }

    signal backRequested()

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // header
        RowLayout {
            Layout.fillWidth: true
            Layout.preferredHeight: 64
            Layout.leftMargin: Theme.spacing.xl
            Layout.rightMargin: Theme.spacing.xl
            spacing: Theme.spacing.sm

            ThemedIcon {
                source: "qrc:/DocumentApp/assets/icons/chevron-down.svg"
                rotation: 90
                size: 16
                color: Theme.colors.textFaint
                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.backRequested() }
            }
            Text {
                Layout.fillWidth: true
                text: root.doc ? root.doc.fileName : "Documento"
                color: Theme.colors.textPrimary
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeXl
                font.weight: Theme.typography.weightBold
                elide: Text.ElideRight
            }
            StatusBadge { visible: root.doc; kind: "confidence"; confidence: root.doc ? root.doc.ocrConfidence : 0 }
            OutlineButton { text: "Descarregar" }
            OutlineButton { text: "Partilhar" }
        }
        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 0

            // --- content viewer -----------------------------------------------
            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                color: "#EDEAE0"

                ColumnLayout {
                    anchors.centerIn: parent
                    spacing: Theme.spacing.sm
                    ThemedIcon { Layout.alignment: Qt.AlignHCenter; source: "qrc:/DocumentApp/assets/icons/file.svg"; size: 40; color: Theme.colors.textFaintest }
                    Text {
                        Layout.alignment: Qt.AlignHCenter
                        text: "Pré-visualização por ligar ao motor de renderização C++"
                        color: Theme.colors.textFaint
                        font.family: Theme.typography.uiFamily
                        font.pixelSize: Theme.typography.sizeMd
                    }
                }
            }

            Rectangle { Layout.preferredWidth: 1; Layout.fillHeight: true; color: Theme.colors.border }

            // --- side panel: metadata / AI / versions --------------------------
            ColumnLayout {
                Layout.preferredWidth: 360
                Layout.fillHeight: true
                spacing: 0

                TabBar {
                    id: tabs
                    Layout.fillWidth: true
                    background: Rectangle { color: Theme.colors.surface }
                    TabButton { text: "Metadados"; font.family: Theme.typography.uiFamily }
                    TabButton { text: "IA"; font.family: Theme.typography.uiFamily }
                    TabButton { text: "Versões"; font.family: Theme.typography.uiFamily }
                }

                StackLayout {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    currentIndex: tabs.currentIndex

                    // Metadados
                    ColumnLayout {
                        spacing: Theme.spacing.sm
                        Layout.margins: Theme.spacing.md

                        Repeater {
                            model: root.doc ? [
                                { label: "Tipo", value: root.doc.documentType },
                                { label: "Pasta", value: root.doc.folderName },
                                { label: "Autor", value: root.doc.author },
                                { label: "Modificado", value: root.doc.modifiedDisplay },
                                { label: "Tamanho", value: root.doc.sizeDisplay },
                            ] : []
                            delegate: RowLayout {
                                Layout.fillWidth: true
                                Text { Layout.preferredWidth: 100; text: modelData.label; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                                Text { Layout.fillWidth: true; text: modelData.value; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            }
                        }

                        ColumnLayout {
                            spacing: Theme.spacing.xs
                            Text { text: "ETIQUETAS"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                            Flow {
                                Layout.fillWidth: true
                                spacing: Theme.spacing.xs
                                Repeater {
                                    model: root.doc ? root.doc.tags : []
                                    delegate: IconChip { text: modelData; tint: Theme.colors.textSecondary; tintSoft: "#F3F0E7"; mono: false }
                                }
                            }
                        }
                        Item { Layout.fillHeight: true }
                    }

                    // IA: summary + citations, per the product spec's "Sources" example
                    ColumnLayout {
                        spacing: Theme.spacing.md
                        Layout.margins: Theme.spacing.md

                        Text { text: "Resumo"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                        Text {
                            Layout.fillWidth: true
                            wrapMode: Text.WordWrap
                            text: root.doc ? root.doc.ocrExcerpt : "Sem resumo disponível."
                            color: Theme.colors.textMuted
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeSm
                        }

                        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border }

                        Text { text: "FONTES"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                        Repeater {
                            model: root.doc ? [1, 2] : []
                            delegate: Rectangle {
                                Layout.fillWidth: true
                                implicitHeight: 34
                                radius: Theme.radius.sm
                                color: Theme.colors.surfaceSunken
                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: Theme.spacing.xs
                                    ThemedIcon { source: "qrc:/DocumentApp/assets/icons/file.svg"; size: 13; color: Theme.colors.textFaint }
                                    Text { text: (root.doc ? root.doc.fileName : "") + " · Página " + (12 + index * 6); color: Theme.colors.textMuted; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                                }
                            }
                        }
                        Item { Layout.fillHeight: true }
                    }

                    // Versões
                    EmptyState {
                        iconSource: "qrc:/DocumentApp/assets/icons/file.svg"
                        title: "Sem histórico de versões"
                        message: "Novas versões deste documento vão aparecer aqui."
                    }
                }
            }
        }
    }
}
