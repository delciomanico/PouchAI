import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// Combines the frames' 3 onboarding screens (Grupo / Equipa / Pronto) into
// one page with internal step state, rather than 3 separate top-level
// pages — StackView back/forward would otherwise fight with the wizard's
// own linear Voltar/Continuar flow.
Item {
    id: root

    signal finished()
    signal skipped()

    property int step: 1 // 1 Grupo, 2 Equipa, 3 Pronto
    readonly property var stepLabels: ["Grupo", "Equipa", "Pronto"]

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // header: logo + skip
        RowLayout {
            Layout.fillWidth: true
            Layout.preferredHeight: 78
            Layout.leftMargin: Theme.spacing.xxl
            Layout.rightMargin: Theme.spacing.xxl

            RowLayout {
                spacing: Theme.spacing.xs
                Item {
                    Layout.preferredWidth: 32; Layout.preferredHeight: 32
                    Rectangle { anchors.fill: parent; radius: 9; color: Theme.colors.accent
                        Text { anchors.centerIn: parent; text: "D"; color: "#fff"; font.family: Theme.typography.uiFamily; font.weight: Theme.typography.weightExtrabold; font.pixelSize: 14 }
                    }
                }
                Text { text: "Gestão Documental"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightExtrabold }
            }
            Item { Layout.fillWidth: true }
            Text {
                visible: root.step < 3
                text: "Ignorar por agora"
                color: Theme.colors.textFaint
                font.family: Theme.typography.uiFamily
                font.pixelSize: Theme.typography.sizeMd
                font.weight: Theme.typography.weightBold
                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: root.skipped() }
            }
        }

        // stepper
        RowLayout {
            Layout.alignment: Qt.AlignHCenter
            Layout.bottomMargin: Theme.spacing.lg
            spacing: 0

            Repeater {
                model: 3
                delegate: RowLayout {
                    readonly property int stepNum: index + 1
                    readonly property bool done: stepNum < root.step
                    readonly property bool active: stepNum === root.step

                    ColumnLayout {
                        spacing: Theme.spacing.xs
                        Rectangle {
                            Layout.alignment: Qt.AlignHCenter
                            width: 30; height: 30; radius: 15
                            color: done || active ? Theme.colors.accent : Theme.colors.surfaceSunken
                            border.width: 0
                            ThemedIcon {
                                anchors.centerIn: parent
                                visible: done
                                source: "qrc:/DocumentApp/assets/icons/check.svg"
                                size: 14; color: "#fff"
                            }
                            Text {
                                anchors.centerIn: parent
                                visible: !done
                                text: stepNum
                                color: active ? "#fff" : Theme.colors.textFaintest
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeMd
                                font.weight: Theme.typography.weightBold
                            }
                        }
                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: root.stepLabels[index]
                            color: active ? Theme.colors.accent : Theme.colors.textMuted
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeSm
                            font.weight: active ? Theme.typography.weightBold : Theme.typography.weightMedium
                        }
                    }

                    Rectangle {
                        visible: index < 2
                        Layout.preferredWidth: 96; Layout.preferredHeight: 2
                        Layout.bottomMargin: 22
                        color: done ? Theme.colors.accent : Theme.colors.border
                    }
                }
            }
        }

        // content
        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            StackLayout {
                anchors.centerIn: parent
                currentIndex: root.step - 1
                width: 480

                // --- step 1: Grupo ------------------------------------------------
                Rectangle {
                    implicitHeight: groupCol.implicitHeight + Theme.spacing.xxl * 2
                    radius: Theme.radius.xl
                    color: Theme.colors.surface
                    border.width: 1
                    border.color: Theme.colors.border

                    ColumnLayout {
                        id: groupCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.xxl
                        spacing: Theme.spacing.lg

                        ColumnLayout {
                            spacing: 6
                            Text { text: "Cria o teu grupo"; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXxl; font.weight: Theme.typography.weightExtrabold; color: Theme.colors.textPrimary }
                            Text { text: "Isto ajuda-nos a organizar os documentos por equipa ou organização."; wrapMode: Text.WordWrap; Layout.fillWidth: true; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                        }

                        RowLayout {
                            spacing: Theme.spacing.md
                            Rectangle {
                                Layout.preferredWidth: 62; Layout.preferredHeight: 62; radius: Theme.radius.lg
                                color: Theme.colors.surfaceSunken
                                border.width: 1.5; border.color: Theme.colors.borderStrong
                                ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/upload.svg"; size: 22; color: Theme.colors.textFaintest }
                            }
                            ColumnLayout {
                                spacing: 2
                                Text { text: "Logótipo do grupo"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                                Text { text: "PNG ou SVG, opcional"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                            }
                        }

                        ColumnLayout {
                            spacing: 6
                            Text { text: "Nome do grupo"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                            TextField {
                                id: groupNameField
                                Layout.fillWidth: true; Layout.preferredHeight: 46
                                placeholderText: "ex. WEMOF Group"
                                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                leftPadding: Theme.spacing.md
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeBase
                            }
                        }

                        PrimaryButton { Layout.fillWidth: true; implicitHeight: 48; text: "Continuar"; onClicked: root.step = 2 }
                    }
                }

                // --- step 2: Equipa ----------------------------------------------
                Rectangle {
                    implicitHeight: teamCol.implicitHeight + Theme.spacing.xxl * 2
                    radius: Theme.radius.xl
                    color: Theme.colors.surface
                    border.width: 1
                    border.color: Theme.colors.border

                    ColumnLayout {
                        id: teamCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.xxl
                        spacing: Theme.spacing.lg

                        ColumnLayout {
                            spacing: 6
                            Text { text: "Convida a tua equipa"; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXxl; font.weight: Theme.typography.weightExtrabold; color: Theme.colors.textPrimary }
                            Text { text: "Adiciona os colegas que vão trabalhar convosco. Podes sempre convidar mais tarde."; wrapMode: Text.WordWrap; Layout.fillWidth: true; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                        }

                        RowLayout {
                            spacing: Theme.spacing.xs
                            TextField {
                                id: inviteEmail
                                Layout.fillWidth: true; Layout.preferredHeight: 46
                                placeholderText: "email@empresa.co.ao"
                                background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                leftPadding: Theme.spacing.md
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeBase
                            }
                            Rectangle {
                                Layout.preferredWidth: 46; Layout.preferredHeight: 46; radius: Theme.radius.md
                                color: Theme.colors.accent
                                ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/plus.svg"; size: 17; color: "#fff" }
                                MouseArea {
                                    anchors.fill: parent
                                    cursorShape: Qt.PointingHandCursor
                                    onClicked: { if (inviteEmail.text.length > 0) { teamModel.invite(inviteEmail.text, "Membro"); inviteEmail.text = "" } }
                                }
                            }
                        }

                        ColumnLayout {
                            spacing: Theme.spacing.xs
                            Text { text: "Convites por enviar · " + teamModel.count; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }

                            Repeater {
                                model: teamModel
                                delegate: Rectangle {
                                    Layout.fillWidth: true
                                    Layout.preferredHeight: 46
                                    radius: Theme.radius.md
                                    color: Theme.colors.surfaceSunken
                                    border.width: 1; border.color: Theme.colors.border

                                    RowLayout {
                                        anchors.fill: parent
                                        anchors.margins: Theme.spacing.sm
                                        spacing: Theme.spacing.sm
                                        Rectangle {
                                            Layout.preferredWidth: 28; Layout.preferredHeight: 28; radius: 14
                                            color: Theme.colors.accentSoft
                                            Text { anchors.centerIn: parent; text: model.initials; color: Theme.colors.accent; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                                        }
                                        Text { Layout.fillWidth: true; text: model.email; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; elide: Text.ElideRight }
                                        IconChip { text: model.roleName; tint: Theme.colors.textMuted; tintSoft: "#EDEAE0"; mono: false }
                                        ThemedIcon {
                                            source: "qrc:/DocumentApp/assets/icons/x.svg"; size: 14; color: Theme.colors.textFaintest
                                            MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: teamModel.remove(model.id) }
                                        }
                                    }
                                }
                            }
                        }

                        RowLayout {
                            spacing: Theme.spacing.sm
                            OutlineButton { text: "Voltar"; implicitHeight: 48; onClicked: root.step = 1 }
                            PrimaryButton { Layout.fillWidth: true; implicitHeight: 48; text: "Continuar"; onClicked: root.step = 3 }
                        }
                    }
                }

                // --- step 3: Pronto -----------------------------------------------
                Rectangle {
                    implicitHeight: doneCol.implicitHeight + Theme.spacing.xxl * 2
                    radius: Theme.radius.xl
                    color: Theme.colors.surface
                    border.width: 1
                    border.color: Theme.colors.border

                    ColumnLayout {
                        id: doneCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.xxl
                        spacing: Theme.spacing.lg

                        Rectangle {
                            Layout.alignment: Qt.AlignHCenter
                            width: 64; height: 64; radius: 32
                            color: Theme.colors.accent
                            ThemedIcon { anchors.centerIn: parent; source: "qrc:/DocumentApp/assets/icons/check.svg"; size: 28; color: "#fff" }
                        }

                        ColumnLayout {
                            spacing: 6
                            Layout.alignment: Qt.AlignHCenter
                            Text { Layout.alignment: Qt.AlignHCenter; text: "Tudo pronto!"; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXxl; font.weight: Theme.typography.weightExtrabold; color: Theme.colors.textPrimary }
                            Text {
                                Layout.alignment: Qt.AlignHCenter
                                horizontalAlignment: Text.AlignHCenter
                                width: 340
                                wrapMode: Text.WordWrap
                                text: "O teu espaço está configurado e já podes começar a organizar os documentos do grupo."
                                color: Theme.colors.textFaint
                                font.family: Theme.typography.uiFamily
                                font.pixelSize: Theme.typography.sizeMd
                            }
                        }

                        ColumnLayout {
                            spacing: Theme.spacing.xs
                            Rectangle {
                                Layout.fillWidth: true; Layout.preferredHeight: 44; radius: Theme.radius.md
                                color: Theme.colors.surfaceSunken; border.width: 1; border.color: Theme.colors.border
                                RowLayout {
                                    anchors.fill: parent; anchors.margins: Theme.spacing.sm; spacing: Theme.spacing.sm
                                    ThemedIcon { source: "qrc:/DocumentApp/assets/icons/check.svg"; size: 15; color: Theme.colors.accent }
                                    Text { text: "Grupo " + (groupNameField.text.length > 0 ? groupNameField.text : "Nome do Grupo") + " criado"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                                }
                            }
                            Rectangle {
                                Layout.fillWidth: true; Layout.preferredHeight: 44; radius: Theme.radius.md
                                color: Theme.colors.surfaceSunken; border.width: 1; border.color: Theme.colors.border
                                RowLayout {
                                    anchors.fill: parent; anchors.margins: Theme.spacing.sm; spacing: Theme.spacing.sm
                                    ThemedIcon { source: "qrc:/DocumentApp/assets/icons/check.svg"; size: 15; color: Theme.colors.accent }
                                    Text { text: teamModel.count + " pessoas convidadas para a equipa"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                                }
                            }
                        }

                        PrimaryButton { Layout.fillWidth: true; implicitHeight: 48; text: "Ir para o Painel"; onClicked: root.finished() }

                        Text {
                            Layout.alignment: Qt.AlignHCenter
                            text: "Podes convidar mais pessoas em qualquer altura, em Definições."
                            color: Theme.colors.textFainter
                            font.family: Theme.typography.uiFamily
                            font.pixelSize: Theme.typography.sizeSm
                        }
                    }
                }
            }
        }
    }
}
