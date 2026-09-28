import QtQuick
import QtQuick.Layouts
import DocumentApp

// Generic "nothing here yet" state — an empty folder, no search results,
// no team members. Keeps every page's empty case visually consistent
// instead of each page inventing its own.
Column {
    id: root

    property string iconSource: "qrc:/DocumentApp/assets/icons/folder.svg"
    property string title: "Nada por aqui ainda"
    property string message: ""
    property string actionLabel: ""
    signal actionTriggered()

    anchors.centerIn: parent
    spacing: Theme.spacing.md
    width: 320

    Rectangle {
        anchors.horizontalCenter: parent.horizontalCenter
        width: 64; height: 64; radius: Theme.radius.lg
        color: Theme.colors.surfaceSunken
        ThemedIcon {
            anchors.centerIn: parent
            source: root.iconSource
            size: 28
            color: Theme.colors.textFaintest
        }
    }

    Text {
        anchors.horizontalCenter: parent.horizontalCenter
        text: root.title
        color: Theme.colors.textPrimary
        font.family: Theme.typography.uiFamily
        font.pixelSize: Theme.typography.sizeXl
        font.weight: Theme.typography.weightBold
    }

    Text {
        visible: root.message !== ""
        anchors.horizontalCenter: parent.horizontalCenter
        text: root.message
        width: parent.width
        horizontalAlignment: Text.AlignHCenter
        wrapMode: Text.WordWrap
        color: Theme.colors.textFaint
        font.family: Theme.typography.uiFamily
        font.pixelSize: Theme.typography.sizeMd
    }

    PrimaryButton {
        visible: root.actionLabel !== ""
        anchors.horizontalCenter: parent.horizontalCenter
        text: root.actionLabel
        onClicked: root.actionTriggered()
    }
}
