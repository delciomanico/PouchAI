import QtQuick
import QtQuick.Layouts
import DocumentApp

Item {
    id: root

    property string query: "contratos assinados pela ACME em 2026"

    signal documentOpened(string documentId)

    // Client-side mock ranking: matches fileName/ocrExcerpt/tags against the
    // query terms and produces a naive relevance score. The real backend
    // will replace this with semantic/keyword search server-side —
    // SearchPage only ever calls backend.search(query) and would then bind
    // to a proper results model; this proxy exists purely so the page has
    // something real to show today.
    property var _results: {
        const terms = root.query.toLowerCase().split(/\s+/).filter(t => t.length > 2)
        const rows = []
        for (let i = 0; i < documentModel.count; ++i) {
            const row = documentModel.get(i)
            const haystack = (row.fileName + " " + row.ocrExcerpt + " " + row.tags.join(" ")).toLowerCase()
            let hits = 0
            for (const t of terms) { if (haystack.indexOf(t) !== -1) hits++ }
            if (hits > 0) {
                row.relevance = Math.min(99, 60 + hits * 12)
                rows.push(row)
            }
        }
        rows.sort((a, b) => b.relevance - a.relevance)
        return rows
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacing.xl
        spacing: Theme.spacing.md

        ColumnLayout {
            spacing: 4
            Text { text: "ÁREA DE TRABALHO / TODOS OS DOCUMENTOS"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
            Text { text: "Todos os Documentos"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeDisplay - 2; font.weight: Theme.typography.weightExtrabold }
        }

        SmartSearchBar {
            Layout.fillWidth: true
            text: root.query
            onSearchRequested: (q) => { root.query = q; backend.search(q) }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacing.xs

            IconChip { text: "Contratos"; tint: "#FFFFFF"; tintSoft: Theme.colors.accent; mono: false; pixelSize: Theme.typography.sizeSm }
            IconChip { text: "Este mês"; tint: Theme.colors.textSecondary; tintSoft: "#F3F0E7"; mono: false; pixelSize: Theme.typography.sizeSm }
            IconChip { text: "Faturas"; tint: Theme.colors.textSecondary; tintSoft: "#F3F0E7"; mono: false; pixelSize: Theme.typography.sizeSm }
            IconChip { text: "Por mim"; tint: Theme.colors.textSecondary; tintSoft: "#F3F0E7"; mono: false; pixelSize: Theme.typography.sizeSm }
            Item { Layout.fillWidth: true }
            OutlineButton { text: "Filtros avançados"; implicitHeight: 34 }
        }

        Rectangle {
            Layout.fillWidth: true
            radius: Theme.radius.lg
            color: Theme.colors.accentSoft
            implicitHeight: aiSummary.implicitHeight + Theme.spacing.md * 2

            RowLayout {
                id: aiSummary
                anchors.fill: parent
                anchors.margins: Theme.spacing.md
                spacing: Theme.spacing.sm

                Rectangle {
                    Layout.preferredWidth: 32; Layout.preferredHeight: 32; radius: 9
                    color: "#FFFFFF"
                    ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/sparkles.svg"; size: 17; color: Theme.colors.accent }
                }
                Text {
                    Layout.fillWidth: true
                    text: "Encontrei " + root._results.length + " documentos relacionados com a tua pesquisa — revê os resultados abaixo, ordenados por relevância."
                    wrapMode: Text.WordWrap
                    color: "#1E3B34"
                    font.family: Theme.typography.uiFamily
                    font.pixelSize: Theme.typography.sizeMd
                }
                IconChip { text: root._results.length + " resultados"; tint: Theme.colors.accent; tintSoft: "#FFFFFF" }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            radius: Theme.radius.lg
            color: Theme.colors.surface
            border.width: 1
            border.color: Theme.colors.border
            clip: true

            ListView {
                anchors.fill: parent
                model: root._results
                visible: root._results.length > 0
                clip: true

                delegate: Rectangle {
                    width: ListView.view.width
                    height: resultCol.implicitHeight + Theme.spacing.md * 2
                    color: "transparent"

                    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.colors.border }

                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onClicked: root.documentOpened(modelData.id)
                    }

                    RowLayout {
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.md
                        spacing: Theme.spacing.sm

                        Rectangle {
                            readonly property var grad: Theme.colors.folderGradients[modelData.accentGradient] ?? Theme.colors.folderGradients.blue
                            Layout.preferredWidth: 36; Layout.preferredHeight: 36; radius: 10
                            Layout.alignment: Qt.AlignTop
                            gradient: Gradient {
                                    orientation: Gradient.Vertical
                                    GradientStop { position: 0; color: parent.grad.start }
                                    GradientStop { position: 1; color: parent.grad.end }
                                }
                            ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/file.svg"; size: 17; color: "#EEF3FB" }
                        }

                        ColumnLayout {
                            id: resultCol
                            Layout.fillWidth: true
                            spacing: 4

                            RowLayout {
                                Layout.fillWidth: true
                                Text { Layout.fillWidth: true; text: modelData.fileName; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeBase; font.weight: Theme.typography.weightBold; elide: Text.ElideRight }
                                IconChip { text: modelData.relevance + "% relevante"; tint: modelData.relevance >= 90 ? Theme.colors.success : Theme.colors.warning; tintSoft: modelData.relevance >= 90 ? Theme.colors.successSoft : Theme.colors.warningSoft }
                            }
                            Text {
                                Layout.fillWidth: true
                                text: modelData.ocrExcerpt
                                wrapMode: Text.WordWrap
                                color: Theme.colors.textMuted
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeMd - 1
                            }
                            Text {
                                text: modelData.folderName + " · " + modelData.author + " · " + modelData.modifiedDisplay
                                color: Theme.colors.textFainter
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeXs + 1
                            }
                        }

                        ThemedIcon {
                            Layout.alignment: Qt.AlignTop
                            source: "qrc:/DocumentApp/assets/icons/kebab.svg"
                            size: 15
                            color: Theme.colors.textFaint
                        }
                    }
                }
            }

            EmptyState {
                visible: root._results.length === 0
                anchors.fill: parent
                iconSource: "qrc:/DocumentApp/assets/icons/search.svg"
                title: "Sem resultados"
                message: "Não encontrámos documentos para essa pesquisa. Tenta outros termos ou remove filtros."
            }
        }
    }
}
