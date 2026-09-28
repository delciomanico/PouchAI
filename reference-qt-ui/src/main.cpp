#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QUrl>

#include "core/MockBackend.h"

// -----------------------------------------------------------------------
// main.cpp
//
// Today this wires the UI to MockBackend, an in-memory stand-in that
// implements BackendInterface with realistic sample data so every QML
// page has something real to bind to.
//
// When the native core (SQLite storage, filesystem, OCR/AI pipeline,
// embedded HTTP server) is ready, swap the single line below for
// LocalBackend, and RemoteBackend for the LAN-client case — nothing in
// ui/ needs to change, since pages only ever talk to `backend`
// (a BackendInterface*) and to the models it exposes.
// -----------------------------------------------------------------------

int main(int argc, char *argv[])
{
    QGuiApplication app(argc, argv);
    app.setOrganizationName("WEMOF Group");
    app.setApplicationName("Gestão Documental");

    QQmlApplicationEngine engine;
    engine.addImportPath(QStringLiteral("qrc:/"));

    // Swap for LocalBackend / RemoteBackend once the C++ core lands.
    auto *backend = new MockBackend(&app);
    engine.rootContext()->setContextProperty("backend", backend);
    engine.rootContext()->setContextProperty("documentModel", backend->documentModel());
    engine.rootContext()->setContextProperty("folderModel", backend->folderModel());
    engine.rootContext()->setContextProperty("teamModel", backend->teamModel());

    QObject::connect(
        &engine, &QQmlApplicationEngine::objectCreationFailed,
        &app, [] { QCoreApplication::exit(-1); },
        Qt::QueuedConnection);

    // loadFromModule() is the documented, portable way to load the app's
    // root QML file — but it only exists from Qt 6.5 onward. The explicit
    // qrc:// path below is the equivalent for Qt 6.4's qt_add_qml_module
    // resource layout, used as a fallback so this builds cleanly on either.
#if QT_VERSION >= QT_VERSION_CHECK(6, 5, 0)
    engine.loadFromModule("DocumentApp", "Main");
#else
    engine.load(QUrl(QStringLiteral("qrc:/DocumentApp/ui/Main.qml")));
#endif
    if (engine.rootObjects().isEmpty())
        return -1;

    return app.exec();
}
