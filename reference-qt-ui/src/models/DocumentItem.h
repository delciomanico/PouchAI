#pragma once

#include <QString>
#include <QDateTime>

// ---------------------------------------------------------------------------
// DocumentItem
//
// Plain data for one document/file. Deliberately not a QObject: it is only
// ever handled in bulk by DocumentModel, so a lightweight value type keeps
// large libraries cheap to hold and sort. Individual documents are exposed
// to QML as model roles (see DocumentModel::roleNames), not as QObject*.
// ---------------------------------------------------------------------------
struct DocumentItem
{
    enum class Status {
        Uploading,
        Processing,   // OCR + AI classification in flight
        PendingReview,// classified, waiting for user approval
        Ready,
        Failed
    };

    QString id;
    QString fileName;
    QString folderId;
    QString folderName;
    QString documentType;      // e.g. "Contrato", "Fatura"
    QString author;
    qint64 sizeBytes = 0;
    QDateTime modifiedAt;
    Status status = Status::Ready;
    int ocrConfidence = 0;     // 0-100, meaningless while Uploading/Processing
    QStringList tags;
    QString ocrExcerpt;        // short extracted-text preview
    QString accentGradient;    // key into Theme.colors.folderGradients, set per-folder
};
