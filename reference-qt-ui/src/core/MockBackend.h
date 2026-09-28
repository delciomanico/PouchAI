#pragma once

#include <QTimer>

#include "BackendInterface.h"
#include "../models/DocumentModel.h"
#include "../models/FolderModel.h"
#include "../models/TeamModel.h"

// ---------------------------------------------------------------------------
// MockBackend
//
// Temporary BackendInterface implementation, in-memory only. Seeds
// realistic Angolan-business sample data (contracts, invoices, HR docs —
// the same fictional ACME/ENDE/CASSFREI-style documents already used
// across the visual frames) so every page has real content to render.
//
// Simulates async behaviour where the real backend will genuinely be
// async (ingestFile -> processing -> classified), using QTimer, so pages
// that render Processing/PendingReview states can be exercised for real
// instead of only in a static mock.
// ---------------------------------------------------------------------------
class MockBackend : public BackendInterface
{
    Q_OBJECT

public:
    explicit MockBackend(QObject *parent = nullptr);

    BackendInterface::Mode mode() const override { return m_mode; }
    QString libraryName() const override { return m_libraryName; }
    bool isSharing() const override { return m_sharing; }
    int connectedClients() const override { return m_connectedClients; }

    DocumentModel *documentModel() const override { return m_documents; }
    FolderModel *folderModel() const override { return m_folders; }
    TeamModel *teamModel() const override { return m_team; }

    void createLibrary(const QString &name, const QString &location) override;
    void startSharing() override;
    void stopSharing() override;
    void connectToLibrary(const QString &hostAddress) override;

    void ingestFile(const QString &localFilePath, const QString &folderId) override;
    void approveDocument(const QString &documentId) override;
    void rejectDocument(const QString &documentId) override;

    void search(const QString &query) override;

private:
    void seed();

    DocumentModel *m_documents;
    FolderModel *m_folders;
    TeamModel *m_team;

    BackendInterface::Mode m_mode = BackendInterface::Mode::Standalone;
    QString m_libraryName = QStringLiteral("Nome do Grupo");
    bool m_sharing = false;
    int m_connectedClients = 0;

    int m_nextDocSeq = 1000;
};
