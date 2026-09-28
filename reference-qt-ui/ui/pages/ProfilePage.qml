import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

Item {
    id: root

    RowLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacing.xl
        spacing: Theme.spacing.lg

        // --- identity card ----------------------------------------------------
        Rectangle {
            Layout.preferredWidth: 300
            Layout.fillHeight: true
            radius: Theme.radius.lg
            color: Theme.colors.surface
            border.width: 1
            border.color: Theme.colors.border

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: Theme.spacing.lg
                spacing: Theme.spacing.md

                Rectangle {
                    Layout.alignment: Qt.AlignHCenter
                    Layout.preferredWidth: 84; Layout.preferredHeight: 84; radius: 22
                    color: Theme.colors.accent
                    Text { anchors.centerIn: parent; text: "DM"; color: "#fff"; font.family: Theme.typography.uiFamily; font.weight: Theme.typography.weightExtrabold; font.pixelSize: 28 }
                }

                ColumnLayout {
                    Layout.alignment: Qt.AlignHCenter
                    spacing: 2
                    Text { Layout.alignment: Qt.AlignHCenter; text: "Délcio Manico"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg + 1; font.weight: Theme.typography.weightExtrabold }
                    Text { Layout.alignment: Qt.AlignHCenter; text: "delcio@wemof.tech"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd - 1 }
                }

                IconChip { Layout.alignment: Qt.AlignHCenter; text: "Admin"; tint: Theme.colors.accent; tintSoft: Theme.colors.accentSoft }

                OutlineButton { Layout.fillWidth: true; Layout.topMargin: Theme.spacing.xs; implicitHeight: 38; text: "Alterar fotografia" }

                Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border; Layout.topMargin: Theme.spacing.xs; Layout.bottomMargin: Theme.spacing.xs }

                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacing.xs
                    RowLayout { Layout.fillWidth: true
                        Text { text: "Grupo"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                        Item { Layout.fillWidth: true }
                        Text { text: backend.libraryName; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                    }
                    RowLayout { Layout.fillWidth: true
                        Text { text: "Membro desde"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                        Item { Layout.fillWidth: true }
                        Text { text: "Set 2026"; color: Theme.colors.textSecondary; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                    }
                }

                Item { Layout.fillHeight: true }
            }
        }

        // --- forms ---------------------------------------------------------------
        ColumnLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: Theme.spacing.md

            Rectangle {
                Layout.fillWidth: true
                radius: Theme.radius.lg
                color: Theme.colors.surface
                border.width: 1; border.color: Theme.colors.border
                implicitHeight: infoCol.implicitHeight + Theme.spacing.lg * 2

                ColumnLayout {
                    id: infoCol
                    anchors.fill: parent
                    anchors.margins: Theme.spacing.lg
                    spacing: Theme.spacing.md

                    Text { text: "Informação pessoal"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }

                    RowLayout {
                        spacing: Theme.spacing.md
                        ColumnLayout {
                            Layout.fillWidth: true; spacing: 6
                            Text { text: "Nome completo"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            TextField {
                                Layout.fillWidth: true; Layout.preferredHeight: 42
                                text: "Délcio Manico"
                                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                leftPadding: Theme.spacing.md
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeMd
                            }
                        }
                        ColumnLayout {
                            Layout.fillWidth: true; spacing: 6
                            Text { text: "Email"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            TextField {
                                Layout.fillWidth: true; Layout.preferredHeight: 42
                                text: "delcio@wemof.tech"
                                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                leftPadding: Theme.spacing.md
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeMd
                            }
                        }
                    }

                    RowLayout { Layout.alignment: Qt.AlignRight; PrimaryButton { text: "Guardar"; implicitHeight: 38 } }
                }
            }

            Rectangle {
                Layout.fillWidth: true
                radius: Theme.radius.lg
                color: Theme.colors.surface
                border.width: 1; border.color: Theme.colors.border
                implicitHeight: pwCol.implicitHeight + Theme.spacing.lg * 2

                ColumnLayout {
                    id: pwCol
                    anchors.fill: parent
                    anchors.margins: Theme.spacing.lg
                    spacing: Theme.spacing.md

                    Text { text: "Palavra-passe"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }

                    RowLayout {
                        spacing: Theme.spacing.md
                        ColumnLayout {
                            Layout.fillWidth: true; spacing: 6
                            Text { text: "Nova palavra-passe"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            TextField {
                                Layout.fillWidth: true; Layout.preferredHeight: 42
                                echoMode: TextInput.Password
                                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                leftPadding: Theme.spacing.md
                            }
                        }
                        ColumnLayout {
                            Layout.fillWidth: true; spacing: 6
                            Text { text: "Confirmar"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            TextField {
                                Layout.fillWidth: true; Layout.preferredHeight: 42
                                echoMode: TextInput.Password
                                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                leftPadding: Theme.spacing.md
                            }
                        }
                    }
                    RowLayout { Layout.alignment: Qt.AlignRight; PrimaryButton { text: "Actualizar"; implicitHeight: 38 } }
                }
            }

            Rectangle {
                Layout.fillWidth: true
                Layout.fillHeight: true
                radius: Theme.radius.lg
                color: Theme.colors.surface
                border.width: 1; border.color: Theme.colors.border

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: Theme.spacing.lg
                    spacing: Theme.spacing.sm

                    Text { text: "Sessões activas"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }

                    Rectangle {
                        Layout.fillWidth: true; implicitHeight: 54; radius: Theme.radius.md
                        color: Theme.colors.surfaceSunken; border.width: 1; border.color: Theme.colors.border
                        RowLayout {
                            anchors.fill: parent; anchors.margins: Theme.spacing.sm; spacing: Theme.spacing.sm
                            ThemedIcon { source: "qrc:/DocumentApp/assets/icons/grid.svg"; size: 17; color: Theme.colors.accent }
                            ColumnLayout {
                                Layout.fillWidth: true; spacing: 1
                                Text { text: "Este computador · Luanda"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                                Text { text: "activo agora"; color: Theme.colors.textFaint; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeXs }
                            }
                            IconChip { text: "Actual"; tint: Theme.colors.success; tintSoft: Theme.colors.successSoft; mono: false }
                        }
                    }

                    Rectangle {
                        Layout.fillWidth: true; implicitHeight: 54; radius: Theme.radius.md
                        color: Theme.colors.surfaceSunken; border.width: 1; border.color: Theme.colors.border
                        RowLayout {
                            anchors.fill: parent; anchors.margins: Theme.spacing.sm; spacing: Theme.spacing.sm
                            ThemedIcon { source: "qrc:/DocumentApp/assets/icons/user.svg"; size: 17; color: Theme.colors.textFaint }
                            ColumnLayout {
                                Layout.fillWidth: true; spacing: 1
                                Text { text: "iPhone · Luanda"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                                Text { text: "última actividade há 2h"; color: Theme.colors.textFaint; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeXs }
                            }
                            OutlineButton { text: "Terminar"; tone: OutlineButton.Tone.Danger; implicitHeight: 30 }
                        }
                    }

                    Item { Layout.fillHeight: true }
                }
            }
        }
    }
}
