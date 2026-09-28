#pragma once

#include <QAbstractListModel>
#include <QVector>
#include <QDateTime>

// ---------------------------------------------------------------------------
// FolderModel
//
// Backs LibraryPage's folder grid. One row per folder; itemCount and
// sharedWith are denormalized here (rather than computed by counting
// DocumentModel rows on every paint) so the grid stays cheap to render
// even for large libraries — see the "Performance" requirements.
// ---------------------------------------------------------------------------
struct FolderItem
{
    QString id;
    QString name;
    int itemCount = 0;
    QDateTime modifiedAt;
    QString accentGradient;   // key into Theme.colors.folderGradients
    bool isShared = false;
    QStringList sharedWithInitials; // e.g. ["AK", "JM", "MT"] for the avatar stack
};

class FolderModel : public QAbstractListModel
{
    Q_OBJECT
    Q_PROPERTY(int count READ count NOTIFY countChanged)

public:
    enum Role {
        IdRole = Qt::UserRole + 1,
        NameRole,
        ItemCountRole,
        ModifiedAtRole,
        ModifiedDisplayRole,
        AccentGradientRole,
        IsSharedRole,
        SharedWithRole
    };
    Q_ENUM(Role)

    explicit FolderModel(QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    int count() const { return rowCount(); }
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

    void setItems(const QVector<FolderItem> &items);

    Q_INVOKABLE QVariantMap get(int row) const;
    Q_INVOKABLE QString nameForId(const QString &folderId) const;

signals:
    void countChanged();

private:
    QVector<FolderItem> m_items;
};
