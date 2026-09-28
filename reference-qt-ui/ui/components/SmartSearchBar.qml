import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// The frames' natural-language search field: a sparkles icon, an "IA"
// badge, and a search button — recessed background with an accent-tinted
// glow ring to signal it's semantic, not a plain keyword box.
Rectangle {
    id: root

    property alias text: field.text
    property string placeholder: "Pesquisa por conteúdo, pessoa, data ou pasta…"

    signal searchRequested(string query)

    implicitHeight: 56
    radius: Theme.radius.lg
    color: "#FCFBF7"
    border.width: 1
    border.color: Qt.rgba(Theme.colors.accent.r, Theme.colors.accent.g, Theme.colors.accent.b, field.activeFocus ? 0.35 : 0.12)

    Behavior on border.color { ColorAnimation { duration: Theme.motion.fast } }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacing.md
        anchors.rightMargin: Theme.spacing.sm
        spacing: Theme.spacing.sm

        ThemedIcon {
            source: "qrc:/DocumentApp/assets/icons/sparkles.svg"
            size: 19
            color: Theme.colors.accent
        }

        TextField {
            id: field
            Layout.fillWidth: true
            placeholderText: root.placeholder
            background: null
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeBase + 1
            color: Theme.colors.textPrimary
            verticalAlignment: TextInput.AlignVCenter
            onAccepted: root.searchRequested(text)
        }

        IconChip {
            text: "IA"
            tint: Theme.colors.accent
            tintSoft: Theme.colors.accentSoft
        }

        PrimaryButton {
            text: "Pesquisar"
            implicitHeight: 38
            onClicked: root.searchRequested(field.text)
        }
    }
}
