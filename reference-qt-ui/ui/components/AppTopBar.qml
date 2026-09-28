import QtQuick
import QtQuick.Shapes
import QtQuick.Layouts
import Qt5Compat.GraphicalEffects
import DocumentApp

// The app's single piece of global chrome: a centered bar with obliquely
// cut side edges (full width at the top, inset by Theme.obliqueCut at the
// bottom — the frames' clip-path trapezoid), floating over the page
// background with a soft shadow. Holds the brand mark, the active-group
// selector, the primary "Novo Documento" action, and the 4 page-nav
// pills (Todos os Documentos / Definições / Perfil — "Novo Documento"
// doubles as the 2nd nav item per the frames).
Item {
    id: root

    property string currentPage: "library" // library | documents | search | upload | settings | profile
    property string groupName: "Nome do Grupo"

    signal navigate(string pageId)
    signal newDocumentRequested()
    signal groupMenuRequested()

    implicitHeight: 92

    Item {
        id: bar
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.top: parent.top
        anchors.topMargin: 14
        width: Math.min(1180, parent.width - 80)
        height: 64

        Shape {
            id: obliqueBg
            anchors.fill: parent
            layer.enabled: true
            layer.effect: DropShadow {
                color: Theme.elevation.shadowColor
                radius: 20
                samples: 32
                verticalOffset: 12
                spread: 0.05
            }
            ShapePath {
                strokeWidth: 0
                fillColor: Theme.colors.surface
                startX: 0; startY: 0
                PathLine { x: obliqueBg.width; y: 0 }
                PathLine { x: obliqueBg.width - Theme.obliqueCut; y: obliqueBg.height }
                PathLine { x: Theme.obliqueCut; y: obliqueBg.height }
                PathLine { x: 0; y: 0 }
            }
        }

        RowLayout {
            anchors.fill: parent
            anchors.leftMargin: 46
            anchors.rightMargin: 46
            spacing: Theme.spacing.lg

            // --- brand mark -------------------------------------------------
            RowLayout {
                spacing: Theme.spacing.xs

                Item {
                    Layout.preferredWidth: 34; Layout.preferredHeight: 34

                    Shape {
                        anchors.fill: parent
                        ShapePath {
                            fillColor: Theme.colors.accent
                            strokeWidth: 0
                            startX: 0; startY: 0
                            PathLine { x: 34; y: 0 }
                            PathLine { x: 34 * 0.78; y: 34 }
                            PathLine { x: 0; y: 34 }
                            PathLine { x: 0; y: 0 }
                        }
                    }
                    Text {
                        anchors.centerIn: parent
                        text: "D"
                        color: "#FFFFFF"
                        font.family: Theme.typography.uiFamily
                        font.weight: Theme.typography.weightExtrabold
                        font.pixelSize: 15
                    }
                }

                ColumnLayout {
                    spacing: 1
                    Text {
                        text: "Gestão Documental"
                        color: Theme.colors.textPrimary
                        font.family: Theme.typography.uiFamily
                        font.pixelSize: Theme.typography.sizeMd
                        font.weight: Theme.typography.weightExtrabold
                    }
                    Text {
                        text: "build 1.0.0 · nativo"
                        color: Theme.colors.textFaintest
                        font.family: Theme.typography.monoFamily
                        font.pixelSize: 10
                    }
                }
            }

            Rectangle { Layout.preferredWidth: 1; Layout.fillHeight: true; Layout.topMargin: 8; Layout.bottomMargin: 8; color: Theme.colors.border }

            // --- nav items ----------------------------------------------------
            RowLayout {
                spacing: Theme.spacing.xs

                // 1. group dropdown
                Rectangle {
                    Layout.preferredHeight: 40
                    implicitWidth: groupRow.implicitWidth + Theme.spacing.md * 2
                    radius: Theme.radius.pill
                    border.width: 1
                    border.color: Theme.colors.border
                    gradient: Gradient {
                        GradientStop { position: 0.0; color: "#FFFFFF" }
                        GradientStop { position: 1.0; color: "#F3F0E7" }
                    }

                    RowLayout {
                        id: groupRow
                        anchors.centerIn: parent
                        spacing: Theme.spacing.xs
                        Rectangle {
                            Layout.preferredWidth: 24; Layout.preferredHeight: 24
                            radius: 7
                            color: "#EDEAE0"
                            Text {
                                anchors.centerIn: parent
                                text: root.groupName.split(" ").map(w => w[0]).join("").substring(0, 2).toUpperCase()
                                font.family: Theme.typography.monoFamily
                                font.pixelSize: 10
                                font.weight: Theme.typography.weightBold
                                color: Theme.colors.textMuted
                            }
                        }
                        Text {
                            text: root.groupName
                            color: Theme.colors.textSecondary
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeMd
                            font.weight: Theme.typography.weightBold
                        }
                        ThemedIcon { source: "qrc:/DocumentApp/assets/icons/chevron-down.svg"; size: 14; color: Theme.colors.textFaint }
                    }
                    MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.groupMenuRequested() }
                }

                Rectangle { Layout.preferredWidth: 1; Layout.preferredHeight: 24; color: Theme.colors.border }

                // 2. novo documento (primary action, doubles as nav)
                PrimaryButton {
                    text: "Novo Documento"
                    implicitHeight: 40
                    onClicked: { root.newDocumentRequested(); root.navigate("upload") }
                }

                // 3. todos os documentos
                NavPill {
                    text: "Todos os Documentos"
                    iconSource: "qrc:/DocumentApp/assets/icons/folder.svg"
                    active: root.currentPage === "library" || root.currentPage === "documents" || root.currentPage === "search"
                    onClicked: root.navigate("search")
                }

                // 4. definições
                NavPill {
                    text: "Definições"
                    iconSource: "qrc:/DocumentApp/assets/icons/gear.svg"
                    active: root.currentPage === "settings"
                    onClicked: root.navigate("settings")
                }

                // 5. perfil
                NavPill {
                    text: "Perfil"
                    iconSource: "qrc:/DocumentApp/assets/icons/user.svg"
                    active: root.currentPage === "profile"
                    onClicked: root.navigate("profile")
                }
            }

            Item { Layout.fillWidth: true }
        }
    }
}
