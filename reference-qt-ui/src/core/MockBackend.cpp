#include "MockBackend.h"

#include <QDateTime>

MockBackend::MockBackend(QObject *parent)
    : BackendInterface(parent)
    , m_documents(new DocumentModel(this))
    , m_folders(new FolderModel(this))
    , m_team(new TeamModel(this))
{
    seed();
}

void MockBackend::seed()
{
    // --- folders --------------------------------------------------------
    QVector<FolderItem> folders;
    folders.append({ "financeiro",  "Financeiro",         12, QDateTime::currentDateTime().addDays(-2),  "teal",       false, {} });
    folders.append({ "juridico",    "Jurídico",            7, QDateTime::currentDateTime().addDays(-5),  "terracotta", false, {} });
    folders.append({ "rh",          "Recursos Humanos",   21, QDateTime::currentDateTime().addDays(-4),  "blue",       false, {} });
    folders.append({ "projectos",   "Projectos",           9, QDateTime::currentDateTime().addSecs(-3600),"green",     true,  { "AK", "JM", "MT" } });
    folders.append({ "compartilhado","Compartilhado",       4, QDateTime::currentDateTime().addDays(-7),  "purple",     true,  { "AK", "JM" } });
    m_folders->setItems(folders);

    // --- documents --------------------------------------------------------
    QVector<DocumentItem> docs;

    auto make = [this](const QString &id, const QString &file, const QString &folderId,
                        const QString &folderName, const QString &type, const QString &author,
                        qint64 size, int daysAgo, int confidence, const QString &gradient,
                        const QString &excerpt, QStringList tags) {
        DocumentItem d;
        d.id = id;
        d.fileName = file;
        d.folderId = folderId;
        d.folderName = folderName;
        d.documentType = type;
        d.author = author;
        d.sizeBytes = size;
        d.modifiedAt = QDateTime::currentDateTime().addDays(-daysAgo);
        d.status = DocumentItem::Status::Ready;
        d.ocrConfidence = confidence;
        d.accentGradient = gradient;
        d.ocrExcerpt = excerpt;
        d.tags = tags;
        return d;
    };

    docs.append(make("doc-1", "Contrato_Fornecedor_2026.pdf", "financeiro", "Financeiro", "Contrato",
                      "Délcio Manico", 2516582, 2, 98, "blue",
                      "…prestação de serviços técnicos entre ACME Lda e WEMOF Group, assinado em Luanda a 12 de Setembro de 2026…",
                      { "fornecedor", "2026", "assinado" }));

    docs.append(make("doc-2", "Aditamento_Contrato_ACME_v2.pdf", "financeiro", "Financeiro", "Contrato",
                      "Délcio Manico", 1153433, 6, 92, "terracotta",
                      "…aditamento ao contrato original com a ACME, revisão de valores para o exercício de 2026…",
                      { "aditamento", "acme" }));

    docs.append(make("doc-3", "Fatura_ENDE_Set2026.pdf", "financeiro", "Financeiro", "Fatura",
                      "Ana Kissanga", 655360, 8, 94, "green",
                      "…fatura referente ao consumo de energia eléctrica do mês de Setembro de 2026…",
                      { "fatura", "ende" }));

    docs.append(make("doc-4", "Procuracao_Assinatura_ACME.pdf", "juridico", "Jurídico", "Procuração",
                      "Délcio Manico", 389120, 20, 71, "purple",
                      "…procuração que autoriza a assinatura de contratos em nome da ACME Lda…",
                      { "procuração", "jurídico" }));

    docs.append(make("doc-5", "Email_Confirmacao_ACME.pdf", "financeiro", "Financeiro", "Email",
                      "Ana Kissanga", 225280, 8, 78, "blue",
                      "…confirmação de recepção do contrato assinado, aguardando cópia física para arquivo…",
                      { "email" }));

    docs.append(make("doc-6", "Relatorio_Cassfrei_Ago.pdf", "projectos", "Projectos", "Relatório",
                      "João Manuel", 3355443, 12, 99, "green",
                      "…relatório de progresso do projecto CASSFREI referente ao mês de Agosto de 2026…",
                      { "relatório", "cassfrei" }));

    docs.append(make("doc-7", "Digitalizacao_0231.jpg", "rh", "Recursos Humanos", "CV",
                      "Mário Tavares", 812440, 3, 74, "purple",
                      "…curriculum vitae digitalizado, candidatura para o departamento técnico…",
                      { "rh", "candidatura" }));

    m_documents->setItems(docs);
    m_nextDocSeq += docs.size();

    // --- team ---------------------------------------------------------------
    QVector<TeamMember> team;
    team.append({ "u-1", "Délcio Manico", "delcio@wemof.tech", "DM", "Admin", false });
    team.append({ "u-2", "Ana Kissanga",  "ana.kissanga@wemof.tech", "AK", "Editor", true });
    team.append({ "u-3", "João Manuel",   "joao.manuel@wemof.tech", "JM", "Membro", true });
    team.append({ "u-4", "Mário Tavares", "mario.tavares@wemof.tech", "MT", "Admin", true });
    m_team->setItems(team);
}

void MockBackend::createLibrary(const QString &name, const QString & /*location*/)
{
    m_libraryName = name.isEmpty() ? m_libraryName : name;
    emit libraryChanged();
}

void MockBackend::startSharing()
{
    m_sharing = true;
    m_connectedClients = 0;
    emit sharingChanged();
}

void MockBackend::stopSharing()
{
    m_sharing = false;
    m_connectedClients = 0;
    emit sharingChanged();
}

void MockBackend::connectToLibrary(const QString & /*hostAddress*/)
{
    m_mode = BackendInterface::Mode::Client;
    emit modeChanged();
}

void MockBackend::ingestFile(const QString &localFilePath, const QString &folderId)
{
    const QString id = QStringLiteral("doc-%1").arg(m_nextDocSeq++);

    DocumentItem item;
    item.id = id;
    item.fileName = localFilePath.section('/', -1);
    item.folderId = folderId;
    item.folderName = m_folders->nameForId(folderId);
    item.modifiedAt = QDateTime::currentDateTime();
    item.status = DocumentItem::Status::Processing;
    item.accentGradient = QStringLiteral("blue");
    m_documents->upsertItem(item);

    // Simulate the OCR/AI pipeline completing after a short delay so
    // UploadPage can genuinely show Processing -> PendingReview.
    QTimer::singleShot(1400, this, [this, id]() {
        m_documents->setStatus(id, DocumentItem::Status::PendingReview, 91);
        emit documentClassified(id);
    });
}

void MockBackend::approveDocument(const QString &documentId)
{
    m_documents->setStatus(documentId, DocumentItem::Status::Ready);
}

void MockBackend::rejectDocument(const QString &documentId)
{
    const int row = m_documents->indexOfId(documentId);
    if (row < 0)
        return;
    m_documents->setStatus(documentId, DocumentItem::Status::Failed);
}

void MockBackend::search(const QString & /*query*/)
{
    // Real semantic/keyword search happens in the C++ core. SearchPage's
    // mock proxy filters documentModel client-side against ocrExcerpt/
    // fileName/tags in the meantime — see ui/pages/SearchPage.qml.
}
