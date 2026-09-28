#include "FolderModel.h"

FolderModel::FolderModel(QObject *parent)
    : QAbstractListModel(parent)
{
}

int FolderModel::rowCount(const QModelIndex &parent) const
{
    if (parent.isValid())
        return 0;
    return m_items.size();
}

QVariant FolderModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() < 0 || index.row() >= m_items.size())
        return {};

    const FolderItem &item = m_items.at(index.row());
    switch (role) {
    case IdRole:              return item.id;
    case NameRole:            return item.name;
    case ItemCountRole:       return item.itemCount;
    case ModifiedAtRole:      return item.modifiedAt;
    case ModifiedDisplayRole: return item.modifiedAt.toString("d MMM · HH:mm");
    case AccentGradientRole:  return item.accentGradient;
    case IsSharedRole:        return item.isShared;
    case SharedWithRole:      return item.sharedWithInitials;
    default:                  return {};
    }
}

QHash<int, QByteArray> FolderModel::roleNames() const
{
    return {
        { IdRole,              "id" },
        { NameRole,            "name" },
        { ItemCountRole,       "itemCount" },
        { ModifiedAtRole,      "modifiedAt" },
        { ModifiedDisplayRole, "modifiedDisplay" },
        { AccentGradientRole,  "accentGradient" },
        { IsSharedRole,        "isShared" },
        { SharedWithRole,      "sharedWith" },
    };
}

void FolderModel::setItems(const QVector<FolderItem> &items)
{
    beginResetModel();
    m_items = items;
    endResetModel();
    emit countChanged();
}

QVariantMap FolderModel::get(int row) const
{
    QVariantMap map;
    if (row < 0 || row >= m_items.size())
        return map;
    const QHash<int, QByteArray> roles = roleNames();
    for (auto it = roles.constBegin(); it != roles.constEnd(); ++it)
        map.insert(QString::fromUtf8(it.value()), data(index(row), it.key()));
    return map;
}

QString FolderModel::nameForId(const QString &folderId) const
{
    for (const auto &item : m_items) {
        if (item.id == folderId)
            return item.name;
    }
    return {};
}
