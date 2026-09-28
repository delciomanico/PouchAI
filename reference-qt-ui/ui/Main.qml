import QtQuick
import QtQuick.Controls.Basic
import QtQuick.Layouts
import DocumentApp

// ---------------------------------------------------------------------------
// Main.qml — the App Shell
//
//   App Shell
//   ├── AppTopBar        (hidden during Welcome/Onboarding — no app chrome
//   │                      yet, matching those frames' full-bleed layout)
//   └── content area     (Loader, swapped by `currentPage`)
//
// The content Loader resolves pages by URL (Loader.source), one at a time,
// rather than pre-declaring all 9 pages as sibling Component{} blocks: with
// this many composite types in one custom QML module, declaring them all
// eagerly at parse time can race in Qt's type loader (seen firsthand while
// testing on Qt 6.4 — a different page would randomly fail to resolve on
// each launch). Loading by URL keeps exactly one page's compilation unit in
// flight at a time, avoiding that.
//
// `currentPage` is the single source of navigation truth. Every page only
// emits intent signals (folderOpened, documentOpened, backRequested, ...);
// Main.qml is the only place that decides what those intents mean for
// navigation, which keeps individual pages reusable and testable in
// isolation. Signals are wired dynamically (Connections { target: pageLoader.item })
// since Loader.source doesn't allow inline `onSomeSignal:` handlers.
// ---------------------------------------------------------------------------
ApplicationWindow {
    id: window

    visible: true
    width: 1440
    height: 900
    minimumWidth: 1040
    minimumHeight: 680
    title: "Gestão Documental"
    color: Theme.colors.background

    property string currentPage: "welcome"
    // library | documents | upload | search | document | settings | profile

    property string selectedFolderId: ""
    property string selectedFolderName: ""
    property string selectedDocumentId: ""

    readonly property bool showsChrome: currentPage !== "welcome" && currentPage !== "onboarding"

    readonly property string pageUrl: {
        const base = "qrc:/DocumentApp/ui/pages/"
        switch (currentPage) {
        case "welcome":    return base + "WelcomePage.qml"
        case "onboarding": return base + "OnboardingPage.qml"
        case "library":    return base + "LibraryPage.qml"
        case "documents":  return base + "DocumentsPage.qml"
        case "upload":     return base + "UploadPage.qml"
        case "search":     return base + "SearchPage.qml"
        case "document":   return base + "DocumentPage.qml"
        case "settings":   return base + "SettingsPage.qml"
        case "profile":    return base + "ProfilePage.qml"
        default:           return base + "LibraryPage.qml"
        }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        AppTopBar {
            id: topBar
            Layout.fillWidth: true
            visible: window.showsChrome
            currentPage: {
                switch (window.currentPage) {
                case "library":
                case "documents":
                case "search":
                case "document":  return "search"
                case "upload":    return "upload"
                case "settings":  return "settings"
                case "profile":   return "profile"
                default:          return ""
                }
            }
            groupName: backend.libraryName

            onNavigate: (pageId) => {
                switch (pageId) {
                case "search":   window.currentPage = "search"; break
                case "upload":   window.currentPage = "upload"; break
                case "settings": window.currentPage = "settings"; break
                case "profile":  window.currentPage = "profile"; break
                }
            }
            onNewDocumentRequested: window.currentPage = "upload"
        }

        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            Loader {
                id: pageLoader
                anchors.fill: parent
                source: window.pageUrl
                asynchronous: false

                onLoaded: {
                    if (window.currentPage === "documents") {
                        item.folderId = window.selectedFolderId
                        item.folderName = window.selectedFolderName
                    } else if (window.currentPage === "document") {
                        item.documentId = window.selectedDocumentId
                    }
                }
            }

            Connections {
                target: pageLoader.item
                ignoreUnknownSignals: true

                // WelcomePage
                function onLoginRequested() { window.currentPage = "library" }
                function onSetupRequested() { window.currentPage = "onboarding" }

                // OnboardingPage
                function onFinished() { window.currentPage = "library" }
                function onSkipped() { window.currentPage = "library" }

                // LibraryPage
                function onFolderOpened(folderId, folderName) {
                    window.selectedFolderId = folderId
                    window.selectedFolderName = folderName
                    window.currentPage = "documents"
                }

                // DocumentsPage / DocumentPage share "backRequested" —
                // resolve it by where we currently are.
                function onBackRequested() {
                    window.currentPage = (window.currentPage === "document") ? "search" : "library"
                }
                function onUploadRequested() { window.currentPage = "upload" }

                // SearchPage
                function onDocumentOpened(documentId) {
                    window.selectedDocumentId = documentId
                    window.currentPage = "document"
                }

                // UploadPage (no navigation change on approve/reject today)
                function onApproved(documentId) {}
                function onRejected(documentId) {}
            }
        }
    }
}
