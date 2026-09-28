import QtQuick
import DocumentApp

// Two modes:
//  - kind: "confidence" -> shows "NN%" colored green/amber by threshold
//  - kind: "status"     -> shows a status label (Pendente/Aprovado/
//                           Revisão necessária/Em fila) colored accordingly
Rectangle {
    id: root

    property string kind: "confidence" // "confidence" | "status"
    property int confidence: 0
    property string status: "ready"    // matches DocumentModel's StatusRole strings

    readonly property var _confidenceTone: confidence >= 90
        ? { fg: Theme.colors.success, bg: Theme.colors.successSoft }
        : (confidence >= 70 ? { fg: Theme.colors.warning, bg: Theme.colors.warningSoft }
                             : { fg: Theme.colors.danger, bg: Theme.colors.dangerSoft })

    readonly property var _statusInfo: {
        switch (status) {
        case "ready":         return { label: "Aprovado", fg: Theme.colors.success, bg: Theme.colors.successSoft }
        case "pendingReview": return { label: "Pendente", fg: Theme.colors.warning, bg: Theme.colors.warningSoft }
        case "processing":    return { label: "A processar…", fg: Theme.colors.textFaintest, bg: "transparent" }
        case "uploading":     return { label: "Em fila", fg: Theme.colors.textFaint, bg: Qt.rgba(0,0,0,0.05) }
        case "failed":        return { label: "Revisão necessária", fg: Theme.colors.danger, bg: Theme.colors.dangerSoft }
        default:              return { label: status, fg: Theme.colors.textFaint, bg: Qt.rgba(0,0,0,0.05) }
        }
    }

    radius: Theme.radius.pill
    color: kind === "confidence" ? _confidenceTone.bg : _statusInfo.bg
    implicitWidth: label.implicitWidth + Theme.spacing.sm * 2
    implicitHeight: label.implicitHeight + Theme.spacing.xxs * 2 + 2

    Text {
        id: label
        anchors.centerIn: parent
        text: kind === "confidence" ? (root.confidence + "%") : root._statusInfo.label
        color: kind === "confidence" ? root._confidenceTone.fg : root._statusInfo.fg
        font.family: kind === "confidence" ? Theme.typography.monoFamily : Theme.typography.uiFamily
        font.pixelSize: Theme.typography.sizeXs
        font.weight: Theme.typography.weightBold
    }
}
