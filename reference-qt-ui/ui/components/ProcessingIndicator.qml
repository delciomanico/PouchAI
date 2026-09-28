import QtQuick
import QtQuick.Layouts
import DocumentApp

// Visualizes the document ingestion pipeline described in the product
// spec: Uploaded -> OCR completed -> Processing AI -> Indexing -> Ready.
// `status` accepts DocumentModel's StatusRole strings; `confidence` fills
// in the OCR badge once available.
Column {
    id: root

    property string status: "processing" // uploading | processing | pendingReview | ready | failed
    property int confidence: 0

    readonly property int _stepIndex: {
        switch (status) {
        case "uploading":     return 0
        case "processing":    return 1
        case "pendingReview": return 2
        case "ready":         return 3
        case "failed":        return 1
        default:              return 0
        }
    }
    readonly property var _steps: [
        "Carregado",
        "OCR concluído",
        "Classificação por IA",
        "Indexado"
    ]

    spacing: Theme.spacing.sm

    Repeater {
        model: root._steps
        delegate: RowLayout {
            width: root.width
            spacing: Theme.spacing.sm

            readonly property bool done: index < root._stepIndex || (root.status === "ready")
            readonly property bool active: index === root._stepIndex && root.status !== "ready"
            readonly property bool failed: root.status === "failed" && index === root._stepIndex

            Rectangle {
                Layout.preferredWidth: 18; Layout.preferredHeight: 18
                radius: 9
                color: failed ? Theme.colors.dangerSoft
                     : done ? Theme.colors.accent
                     : (active ? "transparent" : Theme.colors.surfaceSunken)
                border.width: active && !failed ? 2 : 0
                border.color: Theme.colors.accent

                ThemedIcon {
                    anchors.centerIn: parent
                    visible: done && !failed
                    source: "qrc:/DocumentApp/assets/icons/check.svg"
                    size: 10
                    color: "#FFFFFF"
                }
                ThemedIcon {
                    anchors.centerIn: parent
                    visible: failed
                    source: "qrc:/DocumentApp/assets/icons/x.svg"
                    size: 10
                    color: Theme.colors.danger
                }
            }

            Text {
                Layout.fillWidth: true
                text: modelData
                color: done ? Theme.colors.textPrimary : (active ? Theme.colors.textSecondary : Theme.colors.textFaintest)
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeSm
                font.weight: active ? Theme.typography.weightBold : Theme.typography.weightRegular
            }

            StatusBadge {
                visible: index === 1 && root.confidence > 0 && root._stepIndex >= 1
                kind: "confidence"
                confidence: root.confidence
            }
        }
    }
}
