import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// Docked AI assistant panel from the Library page. Messages are a plain
// ListModel here (UI-local conversation state); asking a question calls
// out through `askRequested(text)`, which LibraryPage forwards to
// `backend` once the real AI pipeline exists — no chat logic lives here.
Rectangle {
    id: root

    signal askRequested(string text)

    property ListModel messages: ListModel {
        ListElement { fromUser: false; text: "Olá! Posso ajudar-te a encontrar ficheiros, resumir pastas ou organizar o teu arquivo. O que precisas?" }
    }

    function addReply(text) {
        messages.append({ fromUser: false, text: text })
        list.positionViewAtEnd()
    }

    color: Theme.colors.surface
    border.width: 0

    Rectangle { anchors.left: parent.left; width: 1; height: parent.height; color: Theme.colors.border }

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        // header
        RowLayout {
            Layout.fillWidth: true
            Layout.margins: Theme.spacing.lg
            spacing: Theme.spacing.sm

            Rectangle {
                Layout.preferredWidth: 32; Layout.preferredHeight: 32
                radius: Theme.radius.sm
                color: Theme.colors.accentSoft
                ThemedIcon {
                    anchors.centerIn: parent
                    source: "qrc:/DocumentApp/assets/icons/sparkles.svg"
                    size: 17
                    color: Theme.colors.accent
                }
            }
            ColumnLayout {
                spacing: 1
                Text { text: "Assistente IA"; color: Theme.colors.textPrimary; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeBase; font.weight: Theme.typography.weightBold }
                Text { text: "Pergunta sobre os teus documentos"; color: Theme.colors.textFainter; font.family: Theme.typography.uiFamily; font.pixelSize: Theme.typography.sizeSm }
            }
        }
        Rectangle { Layout.fillWidth: true; height: 1; color: Theme.colors.border }

        // messages
        ListView {
            id: list
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.margins: Theme.spacing.lg
            spacing: Theme.spacing.sm
            model: root.messages
            clip: true

            delegate: Rectangle {
                height: bubbleText.implicitHeight + Theme.spacing.md * 2
                radius: Theme.radius.lg
                color: fromUser ? Theme.colors.accent : Theme.colors.surfaceSunken
                property real bubbleWidth: Math.min(list.width * 0.88, bubbleText.implicitWidth + Theme.spacing.md * 2)
                width: bubbleWidth
                x: fromUser ? list.width - width : 0

                Text {
                    id: bubbleText
                    anchors.fill: parent
                    anchors.margins: Theme.spacing.md
                    text: model.text
                    wrapMode: Text.WordWrap
                    color: fromUser ? "#FFFFFF" : Theme.colors.textSecondary
                    font.family: Theme.typography.uiFamily
                    font.pixelSize: Theme.typography.sizeMd
                    lineHeight: 1.35
                }
            }
        }

        // input
        RowLayout {
            Layout.fillWidth: true
            Layout.margins: Theme.spacing.md
            spacing: Theme.spacing.xs

            Rectangle {
                Layout.fillWidth: true
                Layout.preferredHeight: 40
                radius: Theme.radius.pill
                color: Theme.colors.surfaceSunken
                border.width: 1
                border.color: Theme.colors.border

                TextField {
                    id: input
                    anchors.fill: parent
                    anchors.leftMargin: Theme.spacing.md
                    anchors.rightMargin: Theme.spacing.xxs
                    verticalAlignment: TextInput.AlignVCenter
                    placeholderText: "Pergunta alguma coisa…"
                    background: null
                    font.family: Theme.typography.uiFamily
                    font.pixelSize: Theme.typography.sizeMd
                    color: Theme.colors.textPrimary
                    onAccepted: sendButton.clicked()
                }
            }

            Rectangle {
                id: sendButton
                Layout.preferredWidth: 34; Layout.preferredHeight: 34
                radius: 17
                color: Theme.colors.accent
                ThemedIcon {
                    anchors.centerIn: parent
                    source: "qrc:/DocumentApp/assets/icons/send.svg"
                    size: 14
                    color: "#FFFFFF"
                }
                function clicked() {
                    if (input.text.trim().length === 0)
                        return
                    root.messages.append({ fromUser: true, text: input.text })
                    root.askRequested(input.text)
                    input.text = ""
                    list.positionViewAtEnd()
                }
                MouseArea { anchors.fill: parent; cursorShape: Qt.PointingHandCursor; onClicked: sendButton.clicked() }
            }
        }
    }
}
