#include "TeamModel.h"
#include <QUuid>

TeamModel::TeamModel(QObject *parent)
    : QAbstractListModel(parent)
{
}

int TeamModel::rowCount(const QModelIndex &parent) const
{
    if (parent.isValid())
        return 0;
    return m_items.size();
}

QVariant TeamModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() < 0 || index.row() >= m_items.size())
        return {};

    const TeamMember &m = m_items.at(index.row());
    switch (role) {
    case IdRole:       return m.id;
    case NameRole:     return m.name;
    case EmailRole:    return m.email;
    case InitialsRole: return m.initials;
    case RoleNameRole: return m.role;
    case PendingRole:  return m.pending;
    default:           return {};
    }
}

QHash<int, QByteArray> TeamModel::roleNames() const
{
    return {
        { IdRole,       "id" },
        { NameRole,     "name" },
        { EmailRole,    "email" },
        { InitialsRole, "initials" },
        { RoleNameRole, "roleName" },
        { PendingRole,  "pending" },
    };
}

void TeamModel::setItems(const QVector<TeamMember> &items)
{
    beginResetModel();
    m_items = items;
    endResetModel();
    emit countChanged();
}

void TeamModel::invite(const QString &email, const QString &role)
{
    TeamMember m;
    m.id = QUuid::createUuid().toString(QUuid::WithoutBraces);
    m.email = email;
    m.name = email.section('@', 0, 0);
    m.initials = m.name.left(2).toUpper();
    m.role = role.isEmpty() ? QStringLiteral("Membro") : role;
    m.pending = true;

    beginInsertRows(QModelIndex(), m_items.size(), m_items.size());
    m_items.append(m);
    endInsertRows();
    emit countChanged();
}

void TeamModel::remove(const QString &memberId)
{
    for (int i = 0; i < m_items.size(); ++i) {
        if (m_items.at(i).id == memberId) {
            beginRemoveRows(QModelIndex(), i, i);
            m_items.remove(i);
            endRemoveRows();
            emit countChanged();
            return;
        }
    }
}
