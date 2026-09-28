import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// Split-panel welcome/login screen (frames: Login.dc.html). Unlike the
// HTML mockup, this runs inside a real OS window — no fake title bar is
// drawn here, the window manager already provides one.
Item {
    id: root

    signal loginRequested()
    signal setupRequested()

    RowLayout {
        anchors.fill: parent
        spacing: 0

        // --- brand panel ------------------------------------------------------
        Rectangle {
            Layout.preferredWidth: Math.min(560, parent.width * 0.4)
            Layout.fillHeight: true
            clip: true
            gradient: Gradient {
                orientation: Gradient.Vertical
                GradientStop { position: 0.0; color: Theme.colors.accent }
                GradientStop { position: 1.0; color: "#0E2E27" }
            }

            Rectangle {
                width: 420; height: 420
                x: parent.width - 260; y: -160
                color: Qt.rgba(1, 1, 1, 0.06)
                radius: 40
                rotation: -18
            }

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: Theme.spacing.xxl
                spacing: Theme.spacing.xl

                RowLayout {
                    spacing: Theme.spacing.xs
                    Item {
                        Layout.preferredWidth: 34; Layout.preferredHeight: 34
                        Rectangle { anchors.fill: parent; radius: 9; color: "#FFFFFF"
                            Text { anchors.centerIn: parent; text: "D"; color: Theme.colors.accent; font.family: Theme.typography.uiFamily; font.weight: Theme.typography.weightExtrabold; font.pixelSize: 15 }
                        }
                    }
                    Text { text: "Gestão Documental"; color: "#FFFFFF"; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightExtrabold }
                }

                Item { Layout.fillHeight: true }

                ColumnLayout {
                    Layout.maximumWidth: 400
                    spacing: Theme.spacing.md
                    Text {
                        text: "Documentos organizados.\nEquipas alinhadas."
                        color: "#FFFFFF"
                        font.family: Theme.typography.uiFamily
                        font.pixelSize: 30
                        font.weight: Theme.typography.weightExtrabold
                        lineHeight: 1.25
                    }
                    Text {
                        text: "Um único espaço para o teu grupo guardar, partilhar e encontrar tudo o que importa."
                        color: Qt.rgba(1, 1, 1, 0.75)
                        wrapMode: Text.WordWrap
                        Layout.fillWidth: true
                        font.family: Theme.typography.uiFamily
                        font.pixelSize: Theme.typography.sizeBase
                    }
                }

                Item { Layout.fillHeight: true }

                Text {
                    text: "build 1.0.0 · nativo"
                    color: Qt.rgba(1, 1, 1, 0.45)
                    font.family: Theme.typography.monoFamily
                    font.pixelSize: 10
                }
            }
        }

        // --- form panel ---------------------------------------------------------
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            ColumnLayout {
                anchors.centerIn: parent
                width: Math.min(380, parent.width - 80)
                spacing: Theme.spacing.lg

                ColumnLayout {
                    spacing: 6
                    Text { text: "Bem-vindo de volta"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: 26; font.weight: Theme.typography.weightExtrabold }
                    Text { text: "Inicia sessão para aceder aos documentos do teu grupo."; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                }

                ColumnLayout {
                    spacing: Theme.spacing.sm

                    ColumnLayout {
                        spacing: 6
                        Text { text: "Email"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                        TextField {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 46
                            placeholderText: "nome@empresa.co.ao"
                            background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken; border.width: 1; border.color: Theme.colors.borderStrong }
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeBase
                            leftPadding: Theme.spacing.md
                        }
                    }
                    ColumnLayout {
                        spacing: 6
                        Text { text: "Palavra-passe"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                        TextField {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 46
                            placeholderText: "••••••••"
                            echoMode: TextInput.Password
                            background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken; border.width: 1; border.color: Theme.colors.borderStrong }
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeBase
                            leftPadding: Theme.spacing.md
                        }
                    }

                    RowLayout {
                        Layout.fillWidth: true
                        Layout.topMargin: 2
                        Text { text: "Manter sessão iniciada"; color: Theme.colors.textMuted; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                        Item { Layout.fillWidth: true }
                        Text {
                            text: "Esqueci a palavra-passe"
                            color: Theme.colors.accent
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeSm
                            font.weight: Theme.typography.weightBold
                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor }
                        }
                    }
                }

                PrimaryButton {
                    Layout.fillWidth: true
                    text: "Entrar"
                    implicitHeight: 48
                    onClicked: root.loginRequested()
                }

                RowLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacing.sm
                    Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border }
                    Text { text: "ou"; color: Theme.colors.textFaintest; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                    Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border }
                }

                OutlineButton {
                    Layout.fillWidth: true
                    text: "Continuar com SSO da organização"
                    implicitHeight: 48
                }

                RowLayout {
                    Layout.alignment: Qt.AlignHCenter
                    spacing: 6
                    Text { text: "Primeira vez aqui?"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                    Text {
                        text: "Configurar o teu espaço"
                        color: Theme.colors.accent
                        font.family: Theme.typography.uiFamily
                        font.pixelSize: Theme.typography.sizeSm
                        font.weight: Theme.typography.weightBold
                        MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.setupRequested() }
                    }
                }
            }
        }
    }
}
