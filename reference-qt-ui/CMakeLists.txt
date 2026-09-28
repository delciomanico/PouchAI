cmake_minimum_required(VERSION 3.21)
project(DocumentApp VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Quick Qml Sql)

qt_standard_project_setup(REQUIRES 6.4)

# qt_add_qml_module's auto-detection of `pragma Singleton` isn't reliable
# on every Qt6 minor version (seen firsthand on 6.4.2: Theme.qml still got
# registered as a plain type, leaving every Theme.colors.* / Theme.spacing.*
# binding undefined at runtime). Marking it explicitly here is the
# documented, version-safe way to do it.
set_source_files_properties(ui/theme/Theme.qml PROPERTIES
    QT_QML_SINGLETON_TYPE TRUE
)

# ---------------------------------------------------------------------------
# QML module
#
# All ui/*.qml files are registered under the "DocumentApp" URI so pages and
# components can simply `import DocumentApp` instead of relying on relative
# file imports. C++ types (models, backend) are added to the same module
# further down with qt_add_qml_module's SOURCES/QML_FILES lists combined.
# ---------------------------------------------------------------------------
qt_add_executable(DocumentApp
    src/main.cpp
)

set_target_properties(DocumentApp PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)

qt_add_qml_module(DocumentApp
    URI DocumentApp
    VERSION 1.0
    QML_FILES
        ui/Main.qml
        ui/theme/Theme.qml
        ui/components/ThemedIcon.qml
        ui/components/AppTopBar.qml
        ui/components/NavPill.qml
        ui/components/PrimaryButton.qml
        ui/components/OutlineButton.qml
        ui/components/IconChip.qml
        ui/components/DocumentCard.qml
        ui/components/DocumentRow.qml
        ui/components/StatusBadge.qml
        ui/components/EmptyState.qml
        ui/components/LoadingState.qml
        ui/components/ProcessingIndicator.qml
        ui/components/FileDropZone.qml
        ui/components/AIChatPanel.qml
        ui/components/SmartSearchBar.qml
        ui/components/SettingsNav.qml
        ui/components/ToggleSwitch.qml
        ui/dialogs/ConfirmDialog.qml
        ui/dialogs/CreateLibraryDialog.qml
        ui/dialogs/ConnectLibraryDialog.qml
        ui/pages/WelcomePage.qml
        ui/pages/OnboardingPage.qml
        ui/pages/LibraryPage.qml
        ui/pages/DocumentsPage.qml
        ui/pages/UploadPage.qml
        ui/pages/SearchPage.qml
        ui/pages/DocumentPage.qml
        ui/pages/SettingsPage.qml
        ui/pages/ProfilePage.qml
    RESOURCES
        assets/icons/folder.svg
        assets/icons/plus.svg
        assets/icons/search.svg
        assets/icons/gear.svg
        assets/icons/user.svg
        assets/icons/chevron-down.svg
        assets/icons/kebab.svg
        assets/icons/check.svg
        assets/icons/upload.svg
        assets/icons/sparkles.svg
        assets/icons/file.svg
        assets/icons/grid.svg
        assets/icons/list.svg
        assets/icons/checklist.svg
        assets/icons/x.svg
        assets/icons/share.svg
        assets/icons/send.svg
        assets/icons/sliders.svg
    SOURCES
        src/core/BackendInterface.h
        src/core/MockBackend.h
        src/core/MockBackend.cpp
        src/models/DocumentItem.h
        src/models/DocumentModel.h
        src/models/DocumentModel.cpp
        src/models/FolderModel.h
        src/models/FolderModel.cpp
        src/models/TeamModel.h
        src/models/TeamModel.cpp
)

target_link_libraries(DocumentApp
    PRIVATE
        Qt6::Core
        Qt6::Quick
        Qt6::Qml
        Qt6::Sql
)

include(GNUInstallDirs)
install(TARGETS DocumentApp
    BUNDLE DESTINATION .
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
