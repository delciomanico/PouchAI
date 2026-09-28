#pragma once

#include <QAbstractListModel>
#include <QVector>

#include "DocumentItem.h"

// ---------------------------------------------------------------------------
// DocumentModel
//
// Backs every document list/grid/table in the UI: LibraryPage's folder
// cards summarize counts from this model, DocumentsPage filters it by
// folderId, UploadPage's bulk table filters it by Status::PendingReview,
// SearchPage filters/ranks it by a query.
//
// Kept intentionally "dumb": it holds rows and answers role queries.
// Filtering/sorting for a given page is done with a QSortFilterProxyModel
// in that page's C++ view-model (not shown yet) or, for the mocked pages
// below, with a lightweight in-QML proxy — never by mutating this model's
// row order for one page's needs.
// ---------------------------------------------------------------------------
class DocumentModel : public QAbstractListModel
{
    Q_OBJECT
    Q_PROPERTY(int count READ count NOTIFY countChanged)

public:
    enum Role {
        IdRole = Qt::UserRole + 1,
        FileNameRole,
        FolderIdRole,
        FolderNameRole,
        DocumentTypeRole,
        AuthorRole,
        SizeBytesRole,
        SizeDisplayRole,
        ModifiedAtRole,
        ModifiedDisplayRole,
        StatusRole,
        OcrConfidenceRole,
        TagsRole,
        OcrExcerptRole,
        AccentGradientRole
    };
    Q_ENUM(Role)

    explicit DocumentModel(QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    int count() const { return rowCount(); }
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

    // Called by MockBackend (and, later, LocalBackend) to populate/update rows.
    void setItems(const QVector<DocumentItem> &items);
    void upsertItem(const DocumentItem &item);
    void setStatus(const QString &documentId, DocumentItem::Status status, int confidence = -1);

    Q_INVOKABLE QVariantMap get(int row) const;
    Q_INVOKABLE int indexOfId(const QString &documentId) const;

signals:
    void countChanged();

private:
    QVector<DocumentItem> m_items;
};
