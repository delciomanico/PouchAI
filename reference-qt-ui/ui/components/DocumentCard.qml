import QtQuick
import QtQuick.Layouts
import Qt5Compat.GraphicalEffects
import DocumentApp

// One folder card in LibraryPage's grid. Mirrors the frames' card: a
// fanned 3-thumbnail stack up top (two soft, rotated "sibling" swatches
// behind a solid front tile carrying the folder icon), a divider, then
// name + meta footer. When `shared` is true the footer shows an avatar
// stack instead of the modified date.
Rectangle {
    id: root

    property string folderId: ""
    property string name: "[Nome da Pasta]"
    property int itemCount: 0
    property string modifiedDisplay: ""
    property string accentGradient: "teal"
    property bool shared: false
    property var sharedWith: []

    signal clicked()
    signal moreOptionsRequested()

    readonly property var _grad: Theme.colors.folderGradients[accentGradient] ?? Theme.colors.folderGradients.teal
    readonly property var _siblingGradients: {
        const keys = Object.keys(Theme.colors.folderGradients).filter(k => k !== accentGradient)
        return [Theme.colors.folderGradients[keys[0]], Theme.colors.folderGradients[keys[1] ?? keys[0]]]
    }

    radius: Theme.radius.xl
    color: Theme.colors.surface
    border.width: 1
    border.color: Theme.colors.border
    clip: true

    layer.enabled: true
    layer.effect: DropShadow {
        color: Theme.elevation.shadowColor
        radius: 18
        samples: 24
        verticalOffset: Theme.elevation.cardOffsetY
        spread: 0.04
    }

    MouseArea {
        anchors.fill: parent
        cursorShape: Qt.PointingHandCursor
        onClicked: root.clicked()
    }

    Column {
        anchors.fill: parent

        // --- thumbnail stack -------------------------------------------------
        Item {
            width: parent.width
            height: parent.height - footer.height - divider.height
            clip: true

            Rectangle {
                anchors.centerIn: parent
                width: 76; height: 102; radius: 11
                rotation: -13
                x: parent.width / 2 - width / 2 - 44
                opacity: 0.5
                gradient: Gradient {
                    orientation: Gradient.Vertical
                    GradientStop { position: 0.0; color: root._siblingGradients[0].start }
                    GradientStop { position: 1.0; color: root._siblingGradients[0].end }
                }
            }
            Rectangle {
                anchors.centerIn: parent
                width: 76; height: 102; radius: 11
                rotation: 13
                x: parent.width / 2 - width / 2 + 44
                opacity: 0.5
                gradient: Gradient {
                    orientation: Gradient.Vertical
                    GradientStop { position: 0.0; color: root._siblingGradients[1].start }
                    GradientStop { position: 1.0; color: root._siblingGradients[1].end }
                }
            }

            Rectangle {
                id: frontTile
                anchors.centerIn: parent
                width: 84; height: 112; radius: 11
                gradient: Gradient {
                    orientation: Gradient.Vertical
                    GradientStop { position: 0.0; color: root._grad.start }
                    GradientStop { position: 1.0; color: root._grad.end }
                }
                layer.enabled: true
                layer.effect: DropShadow {
                    color: "#661A1A16"
                    radius: 14
                    samples: 20
                    verticalOffset: 10
                }

                ThemedIcon {
                    anchors.centerIn: parent
                    source: "qrc:/DocumentApp/assets/icons/folder.svg"
                    size: 26
                    color: Qt.lighter(root._grad.start, 1.5)
                }

                // share badge overlay
                Rectangle {
                    visible: root.shared
                    width: 24; height: 24; radius: 12
                    anchors.right: parent.right
                    anchors.bottom: parent.bottom
                    anchors.rightMargin: -5
                    anchors.bottomMargin: -5
                    color: Theme.colors.surface
                    Rectangle {
                        anchors.centerIn: parent
                        width: 18; height: 18; radius: 9
                        color: Theme.colors.accent
                        ThemedIcon {
                            anchors.centerIn: parent
                            source: "qrc:/DocumentApp/assets/icons/user.svg"
                            size: 10
                            color: "#FFFFFF"
                        }
                    }
                }
            }
        }

        Rectangle { id: divider; width: parent.width; height: 1; color: Theme.colors.border }

        // --- footer ---------------------------------------------------------
        Column {
            id: footer
            width: parent.width
            padding: Theme.spacing.md
            spacing: Theme.spacing.sm

            Text {
                text: root.name
                color: Theme.colors.textPrimary
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeLg
                font.weight: Theme.typography.weightBold
                elide: Text.ElideRight
                width: parent.width - parent.padding * 2
            }

            RowLayout {
                width: parent.width - parent.padding * 2
                spacing: Theme.spacing.xs

                Row {
                    visible: !root.shared
                    spacing: 4
                    Text {
                        text: root.itemCount
                        color: Theme.colors.textPrimary
                        font.family: Theme.typography.monoFamily
                        font.pixelSize: Theme.typography.sizeSm
                        font.weight: Theme.typography.weightBold
                    }
                    Text {
                        text: "itens"
                        color: Theme.colors.textFaint
                        font.family: Theme.typography.monoFamily
                        font.pixelSize: Theme.typography.sizeSm
                    }
                }

                Row {
                    visible: root.shared
                    spacing: 4
                    Text {
                        text: root.sharedWith.length
                        color: Theme.colors.textPrimary
                        font.family: Theme.typography.monoFamily
                        font.pixelSize: Theme.typography.sizeSm
                        font.weight: Theme.typography.weightBold
                    }
                    Text {
                        text: "pessoas"
                        color: Theme.colors.textFaint
                        font.family: Theme.typography.monoFamily
                        font.pixelSize: Theme.typography.sizeSm
                    }
                }

                Item { Layout.fillWidth: true }

                Row {
                    visible: root.shared
                    spacing: -6
                    Repeater {
                        model: root.sharedWith
                        delegate: Rectangle {
                            width: 18; height: 18; radius: 9
                            color: ["#C9A98A", "#8FAFC9", "#9CC98F", "#C9A98A"][index % 4]
                            border.width: 2
                            border.color: Theme.colors.surface
                        }
                    }
                }

                Text {
                    visible: !root.shared
                    text: root.modifiedDisplay
                    color: Theme.colors.textFaint
                    font.family: Theme.typography.monoFamily
                    font.pixelSize: Theme.typography.sizeSm
                }

                Rectangle {
                    width: 22; height: 22; radius: 11
                    color: "transparent"
                    ThemedIcon {
                        anchors.centerIn: parent
                        source: "qrc:/DocumentApp/assets/icons/kebab.svg"
                        size: 15
                        color: Theme.colors.textFaint
                    }
                    MouseArea { anchors.fill: parent; onClicked: root.moreOptionsRequested() }
                }
            }
        }
    }
}
