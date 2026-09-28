import QtQuick
import QtQuick.Layouts
import DocumentApp

Column {
    id: root

    property var tabs: []          // [{ id, label, icon }]
    property string currentTab: tabs.length > 0 ? tabs[0].id : ""
    signal tabSelected(string tabId)

    spacing: 4

    Repeater {
        model: root.tabs
        delegate: Rectangle {
            width: root.width
            height: 40
            radius: Theme.radius.md
            readonly property bool active: modelData.id === root.currentTab
            color: active ? Theme.colors.accentSoft : "transparent"

            RowLayout {
                anchors.fill: parent
                anchors.leftMargin: Theme.spacing.md
                anchors.rightMargin: Theme.spacing.md
                spacing: Theme.spacing.sm

                ThemedIcon {
                    source: modelData.icon
                    size: 16
                    color: active ? Theme.colors.accent : Theme.colors.textMuted
                }
                Text {
                    text: modelData.label
                    color: active ? Theme.colors.accent : Theme.colors.textMuted
                    font.family: Theme.typography.uiFamily
                    font.pixelSize: Theme.typography.sizeMd
                    font.weight: active ? Theme.typography.weightBold : Theme.typography.weightSemibold
                }
            }

            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.tabSelected(modelData.id) }
        }
    }
}
