import QtQuick
import QtQuick.Layouts
import DocumentApp

// One row in a document table. `columns` controls which meta columns show
// (DocumentsPage wants type/modified/size; UploadPage's bulk review wants
// classification/confidence/status), so this single component serves both
// rather than forking into two near-identical rows.
Rectangle {
    id: root

    property string fileName: ""
    property string documentType: ""
    property string modifiedDisplay: ""
    property string sizeDisplay: ""
    property string accentGradient: "teal"
    property bool selectable: false
    property bool selected: false

    // "folder" -> Tipo | Modificado | Tamanho   (DocumentsPage)
    // "review" -> Classificação | Confiança | Estado (UploadPage bulk table)
    property string columns: "folder"
    property string classificationLabel: ""
    property int confidence: 0
    property string status: "ready"

    signal clicked()
    signal checkedChanged(bool checked)

    readonly property var _grad: Theme.colors.folderGradients[accentGradient] ?? Theme.colors.folderGradients.teal

    color: "transparent"
    implicitHeight: 52

    Rectangle { anchors.bottom: parent.bottom; width: parent.width; height: 1; color: Theme.colors.border }

    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacing.lg
        anchors.rightMargin: Theme.spacing.lg
        spacing: Theme.spacing.md

        Rectangle {
            visible: root.selectable
            Layout.preferredWidth: 17; Layout.preferredHeight: 17
            radius: 5
            color: root.selected ? Theme.colors.accent : Theme.colors.surfaceSunken
            border.width: root.selected ? 0 : 1
            border.color: Theme.colors.borderStrong
            ThemedIcon {
                anchors.centerIn: parent
                visible: root.selected
                source: "qrc:/DocumentApp/assets/icons/check.svg"
                size: 11
                color: "#FFFFFF"
            }
            MouseArea { anchors.fill: parent; onClicked: root.checkedChanged(!root.selected) }
        }

        Rectangle {
            Layout.preferredWidth: 30; Layout.preferredHeight: 30
            radius: 8
            gradient: Gradient {
                orientation: Gradient.Vertical
                GradientStop { position: 0.0; color: root._grad.start }
                GradientStop { position: 1.0; color: root._grad.end }
            }
            ThemedIcon {
                anchors.centerIn: parent
                source: "qrc:/DocumentApp/assets/icons/file.svg"
                size: 14
                color: Qt.lighter(root._grad.start, 1.6)
            }
        }

        Text {
            Layout.fillWidth: true
            text: root.fileName
            color: Theme.colors.textPrimary
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeMd
            font.weight: Theme.typography.weightSemibold
            elide: Text.ElideRight
        }

        // folder-table columns
        Text {
            visible: root.columns === "folder"
            Layout.preferredWidth: 130
            text: root.documentType
            color: Theme.colors.textMuted
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeSm
        }
        Text {
            visible: root.columns === "folder"
            Layout.preferredWidth: 170
            text: root.modifiedDisplay
            color: Theme.colors.textFaint
            font.family: Theme.typography.monoFamily
            font.pixelSize: Theme.typography.sizeSm
        }
        Text {
            visible: root.columns === "folder"
            Layout.preferredWidth: 90
            text: root.sizeDisplay
            color: Theme.colors.textFaint
            font.family: Theme.typography.monoFamily
            font.pixelSize: Theme.typography.sizeSm
        }

        // review-table columns
        Text {
            visible: root.columns === "review"
            Layout.preferredWidth: 190
            text: root.classificationLabel
            color: Theme.colors.textSecondary
            font.family: Theme.typography.uiFamily
            font.pixelSize: Theme.typography.sizeSm
        }
        StatusBadge {
            visible: root.columns === "review"
            Layout.preferredWidth: 60
            kind: "confidence"
            confidence: root.confidence
        }
        StatusBadge {
            visible: root.columns === "review"
            Layout.preferredWidth: 140
            kind: "status"
            status: root.status
        }

        Rectangle {
            Layout.preferredWidth: 26; Layout.preferredHeight: 26
            color: "transparent"
            ThemedIcon {
                anchors.centerIn: parent
                source: "qrc:/DocumentApp/assets/icons/kebab.svg"
                size: 15
                color: Theme.colors.textFaint
            }
            MouseArea { anchors.fill: parent; onClicked: root.clicked() }
        }
    }
}
