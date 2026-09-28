import QtQuick
import QtQuick.Layouts
import DocumentApp

// Dashed drop target used on UploadPage. Accepts real OS drag-and-drop
// (DropArea) as well as a click-to-browse fallback via `browseRequested`,
// which the page wires to a native FileDialog.
Rectangle {
    id: root

    property string title: "Arrasta um ficheiro ou clica para escolher"
    property string subtitle: "PDF, PNG ou JPG · até 20 MB"
    property bool compact: false

    signal filesDropped(var urls)
    signal browseRequested()

    implicitHeight: compact ? 60 : 96
    radius: Theme.radius.lg
    border.width: 1.5
    border.color: dropArea.containsDrag ? Theme.colors.accent : Theme.colors.borderStrong
    gradient: Gradient {
        GradientStop { position: 0.0; color: dropArea.containsDrag ? Theme.colors.accentSoft : "#F9F7F1" }
        GradientStop { position: 1.0; color: dropArea.containsDrag ? Theme.colors.accentSoft : "#F1ECE1" }
    }

    // Qt renders a dashed border via a border.width + this trick is limited,
    // so a thin dashed overlay is drawn with a small Canvas for fidelity.
    Canvas {
        anchors.fill: parent
        onPaint: {
            const ctx = getContext("2d")
            ctx.reset()
            ctx.strokeStyle = dropArea.containsDrag ? Theme.colors.accent : Theme.colors.borderStrong
            ctx.lineWidth = 1.5
            ctx.setLineDash([6, 5])
            ctx.roundedRect ? ctx.roundedRect(1, 1, width - 2, height - 2, Theme.radius.lg) : ctx.rect(1, 1, width - 2, height - 2)
            ctx.stroke()
        }
    }

    DropArea {
        id: dropArea
        anchors.fill: parent
        onDropped: (drop) => { if (drop.hasUrls) root.filesDropped(drop.urls) }
    }

    MouseArea {
        anchors.fill: parent
        cursorShape: Qt.PointingHandCursor
        onClicked: root.browseRequested()
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacing.md
        spacing: Theme.spacing.sm

        Rectangle {
            Layout.preferredWidth: root.compact ? 32 : 38
            Layout.preferredHeight: root.compact ? 32 : 38
            radius: Theme.radius.sm
            color: "#EFEAE0"
            ThemedIcon {
                anchors.centerIn: parent
                source: "qrc:/DocumentApp/assets/icons/upload.svg"
                size: root.compact ? 14 : 16
                color: Theme.colors.textFaint
            }
        }

        ColumnLayout {
            spacing: 2
            Text {
                text: root.title
                color: Theme.colors.textSecondary
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeMd
                font.weight: Theme.typography.weightBold
            }
            Text {
                text: root.subtitle
                color: Theme.colors.textFainter
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeSm
            }
        }

        Item { Layout.fillWidth: true }
    }
}
