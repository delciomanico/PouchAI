import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

Item {
    id: root

    property string activeTab: "geral"

    readonly property var tabs: [
        { id: "geral",        label: "Geral",        icon: "qrc:/DocumentApp/assets/icons/grid.svg" },
        { id: "equipa",       label: "Equipa",        icon: "qrc:/DocumentApp/assets/icons/user.svg" },
        { id: "seguranca",    label: "Segurança",     icon: "qrc:/DocumentApp/assets/icons/check.svg" },
        { id: "rede",         label: "Rede",          icon: "qrc:/DocumentApp/assets/icons/share.svg" },
        { id: "integracoes",  label: "Integrações",   icon: "qrc:/DocumentApp/assets/icons/grid.svg" },
    ]

    CreateLibraryDialog {
        id: createLibraryDialog
        onCreateRequested: (name, location, shareOnLan) => {
            backend.createLibrary(name, location)
            if (shareOnLan) backend.startSharing()
        }
    }
    ConnectLibraryDialog {
        id: connectLibraryDialog
        onConnectRequested: (address) => backend.connectToLibrary(address)
    }

    RowLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacing.xl
        spacing: Theme.spacing.lg

        ColumnLayout {
            Layout.fillHeight: true
            spacing: Theme.spacing.md

            ColumnLayout {
                spacing: 4
                Text { text: "ÁREA DE TRABALHO / DEFINIÇÕES"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                Text { text: "Definições do Grupo"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXxl; font.weight: Theme.typography.weightExtrabold }
            }

            SettingsNav {
                Layout.preferredWidth: 220
                Layout.topMargin: Theme.spacing.sm
                tabs: root.tabs
                currentTab: root.activeTab
                onTabSelected: (id) => root.activeTab = id
            }

            Item { Layout.fillHeight: true }
        }

        StackLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.topMargin: 54 // align with header baseline like the frames
            currentIndex: root.tabs.findIndex(t => t.id === root.activeTab)

            // --- Geral ----------------------------------------------------------
            ColumnLayout {
                spacing: Theme.spacing.md

                Rectangle {
                    Layout.fillWidth: true
                    radius: Theme.radius.lg
                    color: Theme.colors.surface
                    border.width: 1; border.color: Theme.colors.border
                    implicitHeight: geralCol.implicitHeight + Theme.spacing.lg * 2

                    ColumnLayout {
                        id: geralCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.lg
                        spacing: Theme.spacing.md

                        Text { text: "Identidade do grupo"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }

                        RowLayout {
                            spacing: Theme.spacing.md
                            Rectangle {
                                Layout.preferredWidth: 56; Layout.preferredHeight: 56; radius: 14
                                color: Theme.colors.accent
                                Text { anchors.centerIn: parent; text: "GN"; color: "#fff"; font.family: Theme.typography.uiFamily; font.weight: Theme.typography.weightExtrabold; font.pixelSize: 18 }
                            }
                            OutlineButton { text: "Alterar logótipo"; implicitHeight: 36 }
                        }

                        RowLayout {
                            spacing: Theme.spacing.md
                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 6
                                Text { text: "Nome do grupo"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                                TextField {
                                    Layout.fillWidth: true; Layout.preferredHeight: 42
                                    text: backend.libraryName
                                    background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                                    leftPadding: Theme.spacing.md
                                    font.family: Theme.typography.uiFamily
                                    font.pixelSize: Theme.typography.sizeMd
                                }
                            }
                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 6
                                Text { text: "Tipo de organização"; color: Theme.colors.textSecondary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                                Rectangle {
                                    Layout.fillWidth: true; Layout.preferredHeight: 42; radius: Theme.radius.md; color: Theme.colors.surfaceSunken
                                    RowLayout { anchors.fill: parent; anchors.margins: Theme.spacing.sm
                                        Text { Layout.fillWidth: true; text: "Holding / Investimento"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd }
                                        ThemedIcon { source: "qrc:/DocumentApp/assets/icons/chevron-down.svg"; size: 13; color: Theme.colors.textFaint }
                                    }
                                }
                            }
                        }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    radius: Theme.radius.lg
                    color: Theme.colors.surface
                    border.width: 1; border.color: Theme.colors.border
                    implicitHeight: 90

                    RowLayout {
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.lg
                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 3
                            Text { text: "Classificação automática por IA"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }
                            Text { text: "Sugerir pasta, tipo e etiquetas ao carregar novos documentos."; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                        }
                        ToggleSwitch { checked: true }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    radius: Theme.radius.lg
                    color: Theme.colors.surface
                    border.width: 1; border.color: Theme.colors.border
                    implicitHeight: 90

                    RowLayout {
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.lg
                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 3
                            Text { text: "Aprovação obrigatória antes de arquivar"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }
                            Text { text: "Documentos com confiança de OCR abaixo de 80% ficam pendentes de revisão."; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                        }
                        ToggleSwitch { checked: true }
                    }
                }

                RowLayout {
                    Layout.alignment: Qt.AlignRight
                    PrimaryButton { text: "Guardar alterações" }
                }
                Item { Layout.fillHeight: true }
            }

            // --- Equipa -----------------------------------------------------------
            ColumnLayout {
                spacing: Theme.spacing.md

                RowLayout {
                    Layout.fillWidth: true
                    spacing: Theme.spacing.xs
                    TextField {
                        id: inviteEmailField
                        Layout.fillWidth: true
                        Layout.preferredHeight: 44
                        placeholderText: "email@empresa.co.ao"
                        background: Rectangle { radius: Theme.radius.md; color: Theme.colors.surfaceSunken }
                        leftPadding: Theme.spacing.md
                        font.family: Theme.typography.uiFamily
                        font.pixelSize: Theme.typography.sizeMd
                    }
                    PrimaryButton {
                        text: "Convidar"
                        onClicked: { if (inviteEmailField.text.length > 0) { teamModel.invite(inviteEmailField.text, "Membro"); inviteEmailField.text = "" } }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: Theme.radius.lg
                    color: Theme.colors.surface
                    border.width: 1; border.color: Theme.colors.border
                    clip: true

                    ListView {
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.sm
                        spacing: Theme.spacing.xs
                        model: teamModel

                        delegate: Rectangle {
                            width: ListView.view.width
                            height: 56
                            radius: Theme.radius.md
                            color: Theme.colors.surfaceSunken

                            RowLayout {
                                anchors.fill: parent
                                anchors.margins: Theme.spacing.sm
                                spacing: Theme.spacing.sm

                                Rectangle {
                                    Layout.preferredWidth: 32; Layout.preferredHeight: 32; radius: 16
                                    color: Theme.colors.accentSoft
                                    Text { anchors.centerIn: parent; text: model.initials; color: Theme.colors.accent; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm; font.weight: Theme.typography.weightBold }
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true
                                    spacing: 1
                                    Text { text: model.name; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                                    Text { text: model.email; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
                                }
                                IconChip { visible: model.pending; text: "Pendente"; tint: Theme.colors.warning; tintSoft: Theme.colors.warningSoft; mono: false }
                                IconChip { text: model.roleName; tint: Theme.colors.textMuted; tintSoft: "#EDEAE0"; mono: false }
                                ThemedIcon {
                                    source: "qrc:/DocumentApp/assets/icons/x.svg"; size: 15; color: Theme.colors.textFaintest
                                    MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: teamModel.remove(model.id) }
                                }
                            }
                        }
                    }
                }
            }

            // --- Segurança ---------------------------------------------------------
            EmptyState {
                iconSource: "qrc:/DocumentApp/assets/icons/check.svg"
                title: "Segurança"
                message: "Autenticação de dois factores, palavra-passe e sessões — por desenhar em detalhe."
            }

            // --- Rede: host / discover, per the product spec ------------------------
            ColumnLayout {
                spacing: Theme.spacing.md

                Rectangle {
                    Layout.fillWidth: true
                    radius: Theme.radius.lg
                    color: Theme.colors.surface
                    border.width: 1; border.color: Theme.colors.border
                    implicitHeight: netCol.implicitHeight + Theme.spacing.lg * 2

                    ColumnLayout {
                        id: netCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacing.lg
                        spacing: Theme.spacing.md

                        RowLayout {
                            Layout.fillWidth: true
                            ColumnLayout {
                                spacing: 3
                                Text { text: "Rede Local"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeLg; font.weight: Theme.typography.weightBold }
                                RowLayout {
                                    spacing: 6
                                    Rectangle { width: 8; height: 8; radius: 4; color: backend.isSharing ? Theme.colors.success : Theme.colors.textFaintest }
                                    Text {
                                        text: backend.isSharing ? "A partilhar esta biblioteca" : "Partilha desativada"
                                        color: Theme.colors.textFaint
                                        font.family: Theme.typography.uiFamily
                                        font.pixelSize: Theme.typography.sizeSm
                                    }
                                }
                            }
                            Item { Layout.fillWidth: true }
                            OutlineButton { text: "Ligar a biblioteca…"; onClicked: connectLibraryDialog.open() }
                            PrimaryButton {
                                text: backend.isSharing ? "Parar partilha" : "Partilhar biblioteca"
                                onClicked: backend.isSharing ? backend.stopSharing() : backend.startSharing()
                            }
                        }

                        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border; visible: backend.isSharing }

                        RowLayout {
                            visible: backend.isSharing
                            spacing: Theme.spacing.xl
                            ColumnLayout {
                                spacing: 2
                                Text { text: "BIBLIOTECA"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                                Text { text: backend.libraryName; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                            }
                            ColumnLayout {
                                spacing: 2
                                Text { text: "CLIENTES LIGADOS"; color: Theme.colors.textFaint; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeXs; font.weight: Theme.typography.weightBold }
                                Text { text: backend.connectedClients; color: Theme.colors.textPrimary; font.family: Theme.typography.monoFamily; font.pixelSize: Theme.typography.sizeMd; font.weight: Theme.typography.weightBold }
                            }
                        }
                    }
                }

                RowLayout {
                    Layout.alignment: Qt.AlignRight
                    OutlineButton { text: "Criar nova biblioteca…"; onClicked: createLibraryDialog.open() }
                }
                Item { Layout.fillHeight: true }
            }

            // --- Integrações ---------------------------------------------------------
            EmptyState {
                iconSource: "qrc:/DocumentApp/assets/icons/grid.svg"
                title: "Integrações"
                message: "Ligações a outros sistemas (email, ERP, armazenamento externo) ficam aqui quando existirem."
            }
        }
    }
}
