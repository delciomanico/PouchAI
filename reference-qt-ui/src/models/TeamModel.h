#pragma once

#include <QAbstractListModel>
#include <QVector>

struct TeamMember
{
    QString id;
    QString name;
    QString email;
    QString initials;
    QString role;       // "Admin" | "Editor" | "Membro"
    bool pending = false; // invited, not yet accepted
};

class TeamModel : public QAbstractListModel
{
    Q_OBJECT
    Q_PROPERTY(int count READ count NOTIFY countChanged)

public:
    enum Role {
        IdRole = Qt::UserRole + 1,
        NameRole,
        EmailRole,
        InitialsRole,
        RoleNameRole,
        PendingRole
    };
    Q_ENUM(Role)

    explicit TeamModel(QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    int count() const { return rowCount(); }
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

    void setItems(const QVector<TeamMember> &items);
    Q_INVOKABLE void invite(const QString &email, const QString &role);
    Q_INVOKABLE void remove(const QString &memberId);

signals:
    void countChanged();

private:
    QVector<TeamMember> m_items;
};
