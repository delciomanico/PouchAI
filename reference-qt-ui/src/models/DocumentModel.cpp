#include "DocumentModel.h"

#include <QLocale>

namespace {

QString statusToString(DocumentItem::Status s)
{
    switch (s) {
    case DocumentItem::Status::Uploading:     return QStringLiteral("uploading");
    case DocumentItem::Status::Processing:    return QStringLiteral("processing");
    case DocumentItem::Status::PendingReview: return QStringLiteral("pendingReview");
    case DocumentItem::Status::Ready:         return QStringLiteral("ready");
    case DocumentItem::Status::Failed:        return QStringLiteral("failed");
    }
    return {};
}

QString formatSize(qint64 bytes)
{
    QLocale locale;
    return locale.formattedDataSize(bytes, 1);
}

}

DocumentModel::DocumentModel(QObject *parent)
    : QAbstractListModel(parent)
{
}

int DocumentModel::rowCount(const QModelIndex &parent) const
{
    if (parent.isValid())
        return 0;
    return m_items.size();
}

QVariant DocumentModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() < 0 || index.row() >= m_items.size())
        return {};

    const DocumentItem &item = m_items.at(index.row());
    switch (role) {
    case IdRole:              return item.id;
    case FileNameRole:        return item.fileName;
    case FolderIdRole:        return item.folderId;
    case FolderNameRole:      return item.folderName;
    case DocumentTypeRole:    return item.documentType;
    case AuthorRole:          return item.author;
    case SizeBytesRole:       return item.sizeBytes;
    case SizeDisplayRole:     return formatSize(item.sizeBytes);
    case ModifiedAtRole:      return item.modifiedAt;
    case ModifiedDisplayRole: return item.modifiedAt.toString("d MMM yyyy · HH:mm");
    case StatusRole:          return statusToString(item.status);
    case OcrConfidenceRole:   return item.ocrConfidence;
    case TagsRole:            return item.tags;
    case OcrExcerptRole:      return item.ocrExcerpt;
    case AccentGradientRole:  return item.accentGradient;
    default:                  return {};
    }
}

QHash<int, QByteArray> DocumentModel::roleNames() const
{
    return {
        { IdRole,              "id" },
        { FileNameRole,        "fileName" },
        { FolderIdRole,        "folderId" },
        { FolderNameRole,      "folderName" },
        { DocumentTypeRole,    "documentType" },
        { AuthorRole,          "author" },
        { SizeBytesRole,       "sizeBytes" },
        { SizeDisplayRole,     "sizeDisplay" },
        { ModifiedAtRole,      "modifiedAt" },
        { ModifiedDisplayRole, "modifiedDisplay" },
        { StatusRole,          "status" },
        { OcrConfidenceRole,   "ocrConfidence" },
        { TagsRole,            "tags" },
        { OcrExcerptRole,      "ocrExcerpt" },
        { AccentGradientRole,  "accentGradient" },
    };
}

void DocumentModel::setItems(const QVector<DocumentItem> &items)
{
    beginResetModel();
    m_items = items;
    endResetModel();
    emit countChanged();
}

void DocumentModel::upsertItem(const DocumentItem &item)
{
    const int row = indexOfId(item.id);
    if (row >= 0) {
        m_items[row] = item;
        const QModelIndex idx = index(row);
        emit dataChanged(idx, idx);
    } else {
        beginInsertRows(QModelIndex(), m_items.size(), m_items.size());
        m_items.append(item);
        endInsertRows();
        emit countChanged();
    }
}

void DocumentModel::setStatus(const QString &documentId, DocumentItem::Status status, int confidence)
{
    const int row = indexOfId(documentId);
    if (row < 0)
        return;
    m_items[row].status = status;
    if (confidence >= 0)
        m_items[row].ocrConfidence = confidence;
    const QModelIndex idx = index(row);
    emit dataChanged(idx, idx, { StatusRole, OcrConfidenceRole });
}

QVariantMap DocumentModel::get(int row) const
{
    QVariantMap map;
    if (row < 0 || row >= m_items.size())
        return map;
    const QHash<int, QByteArray> roles = roleNames();
    for (auto it = roles.constBegin(); it != roles.constEnd(); ++it)
        map.insert(QString::fromUtf8(it.value()), data(index(row), it.key()));
    return map;
}

int DocumentModel::indexOfId(const QString &documentId) const
{
    for (int i = 0; i < m_items.size(); ++i) {
        if (m_items.at(i).id == documentId)
            return i;
    }
    return -1;
}
