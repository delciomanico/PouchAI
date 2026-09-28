#pragma once

#include <QObject>
#include <QString>

class DocumentModel;
class FolderModel;
class TeamModel;

// ---------------------------------------------------------------------------
// BackendInterface
//
// The single seam between UI and data. Pages and C++ view-models call only
// this interface — never SQLite, never the filesystem, never HTTP directly.
//
// Two concrete implementations are expected long-term:
//
//   - LocalBackend  (standalone or "host" mode): talks to SQLite + local
//     filesystem + the in-process AI/OCR pipeline directly.
//   - RemoteBackend (LAN client mode): talks to a DocumentApp host over the
//     embedded HTTP API; never touches the host's SQLite or filesystem.
//
// MockBackend (see MockBackend.h) is the third, temporary implementation:
// it satisfies this same interface with realistic in-memory sample data so
// every page in ui/ can be built and demoed today.
//
// Because every page binds to `backend` (see src/main.cpp) and to the
// QAbstractListModel*s it exposes, swapping MockBackend for LocalBackend or
// RemoteBackend later is a one-line change in main.cpp — no QML changes.
// ---------------------------------------------------------------------------
class BackendInterface : public QObject
{
    Q_OBJECT
    Q_PROPERTY(BackendInterface::Mode mode READ mode NOTIFY modeChanged)
    Q_PROPERTY(QString libraryName READ libraryName NOTIFY libraryChanged)
    Q_PROPERTY(bool isSharing READ isSharing NOTIFY sharingChanged)
    Q_PROPERTY(int connectedClients READ connectedClients NOTIFY sharingChanged)

public:
    enum class Mode { Standalone, Host, Client };
    Q_ENUM(Mode)

    explicit BackendInterface(QObject *parent = nullptr) : QObject(parent) {}
    ~BackendInterface() override = default;

    virtual BackendInterface::Mode mode() const = 0;
    virtual QString libraryName() const = 0;
    virtual bool isSharing() const = 0;
    virtual int connectedClients() const = 0;

    virtual DocumentModel *documentModel() const = 0;
    virtual FolderModel *folderModel() const = 0;
    virtual TeamModel *teamModel() const = 0;

    // Library / network lifecycle -------------------------------------
    Q_INVOKABLE virtual void createLibrary(const QString &name, const QString &location) = 0;
    Q_INVOKABLE virtual void startSharing() = 0;
    Q_INVOKABLE virtual void stopSharing() = 0;
    Q_INVOKABLE virtual void connectToLibrary(const QString &hostAddress) = 0;

    // Document lifecycle -------------------------------------------------
    // folderId may be empty for "no destination chosen yet" (e.g. during
    // the upload/review flow, before the user approves the AI suggestion).
    Q_INVOKABLE virtual void ingestFile(const QString &localFilePath, const QString &folderId) = 0;
    Q_INVOKABLE virtual void approveDocument(const QString &documentId) = 0;
    Q_INVOKABLE virtual void rejectDocument(const QString &documentId) = 0;

    // Search ---------------------------------------------------------------
    Q_INVOKABLE virtual void search(const QString &query) = 0;

signals:
    void modeChanged();
    void libraryChanged();
    void sharingChanged();

    // Emitted once a document's AI classification/OCR result is ready for
    // the user to review — this is what UploadPage listens to in order to
    // fill in the "suggested classification" panel.
    void documentClassified(const QString &documentId);

    void errorOccurred(const QString &message);
};
