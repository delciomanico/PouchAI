// ai-worker: consome a fila `ai_jobs` (ver shared/schema.sql).
//
// Cada instância corre num loop: reclama o job mais antigo em fila
// (claimNext, transação atómica que também marca o documento como
// "processing"), processa (extrai texto real de PDFs com camada de texto
// via pdfio — ver processDocument/extractPdfText; imagens e PDFs sem
// camada de texto ficam sem excerto, não é OCR de imagem ainda, ver
// ROADMAP.md do plano de IA local), grava o resultado (finishJob, marca
// "pending_review"/"failed"), e avisa o Go
// por HTTP em cada transição para que a UI actualize sem recarregar. Se
// ficar sem trabalho durante --idle-timeout-ms, termina sozinho — é
// assim que o pool elástico do lado Go "encerra workers ociosos" (não
// precisa de matar o processo, só de reparar que ele já saiu).
//
// Ver ROADMAP.md, Fase 3.

#include <sqlite3.h>
#define CPPHTTPLIB_THREAD_POOL_COUNT 1
#include <httplib.h>
#include <pdfio.h>

#include <atomic>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <ctime>
#include <iostream>
#include <random>
#include <sstream>
#include <string>
#include <thread>

namespace {

struct Args {
    std::string dbPath;
    std::string callbackUrl;
    std::string heartbeatUrl;
    std::string workerId;
    int idleTimeoutMs = 10000;
    int pollIntervalMs = 300;
    int heartbeatIntervalMs = 5000;
    int workMinMs = 500;  // duração do placeholder de "processamento" —
    int workMaxMs = 1500; // só configurável para tornar testável a Fase 4 (matar a meio de um job).
};

Args parseArgs(int argc, char **argv) {
    Args a;
    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        auto next = [&]() -> std::string {
            return (i + 1 < argc) ? std::string(argv[++i]) : std::string();
        };
        if (arg == "--db") a.dbPath = next();
        else if (arg == "--callback") a.callbackUrl = next();
        else if (arg == "--heartbeat-url") a.heartbeatUrl = next();
        else if (arg == "--worker-id") a.workerId = next();
        else if (arg == "--idle-timeout-ms") a.idleTimeoutMs = std::stoi(next());
        else if (arg == "--poll-interval-ms") a.pollIntervalMs = std::stoi(next());
        else if (arg == "--heartbeat-interval-ms") a.heartbeatIntervalMs = std::stoi(next());
        else if (arg == "--work-min-ms") a.workMinMs = std::stoi(next());
        else if (arg == "--work-max-ms") a.workMaxMs = std::stoi(next());
    }
    return a;
}

std::string nowIso8601() {
    auto now = std::chrono::system_clock::now();
    std::time_t t = std::chrono::system_clock::to_time_t(now);
    std::tm tm{};
#if defined(_WIN32)
    gmtime_s(&tm, &t);
#else
    gmtime_r(&t, &tm);
#endif
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    return std::string(buf);
}

std::string escapeJson(const std::string &s) {
    std::string out;
    out.reserve(s.size());
    for (char c : s) {
        switch (c) {
            case '"': out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            default: out += c;
        }
    }
    return out;
}

std::string inferDocumentType(const std::string &fileName) {
    auto dot = fileName.find_last_of('.');
    if (dot == std::string::npos) return "Documento";
    std::string ext = fileName.substr(dot + 1);
    for (auto &c : ext) c = static_cast<char>(::tolower(static_cast<unsigned char>(c)));
    if (ext == "pdf") return "PDF";
    if (ext == "png" || ext == "jpg" || ext == "jpeg") return "Imagem";
    if (ext == "doc" || ext == "docx") return "Documento Word";
    return "Documento";
}

struct ParsedUrl {
    std::string base; // scheme://host:port
    std::string path;
};

ParsedUrl parseUrl(const std::string &url) {
    auto schemeEnd = url.find("://");
    if (schemeEnd == std::string::npos) return {url, "/"};
    auto pathStart = url.find('/', schemeEnd + 3);
    if (pathStart == std::string::npos) return {url, "/"};
    return {url.substr(0, pathStart), url.substr(pathStart)};
}

bool execSimple(sqlite3 *db, const std::string &sql) {
    char *errMsg = nullptr;
    int rc = sqlite3_exec(db, sql.c_str(), nullptr, nullptr, &errMsg);
    if (rc != SQLITE_OK) {
        std::cerr << "sqlite3_exec falhou: " << (errMsg ? errMsg : "erro desconhecido")
                   << " (" << sql << ")\n";
        sqlite3_free(errMsg);
        return false;
    }
    return true;
}

struct ClaimedJob {
    std::string jobId;
    std::string documentId;
    std::string localPath;
    std::string fileName;
};

// claimNext: transação que reclama o job mais antigo em fila e marca o
// documento associado como "processing". Devolve false se a fila
// estava vazia (ou se algo correu mal a reclamar).
bool claimNext(sqlite3 *db, const std::string &workerId, ClaimedJob &out) {
    if (!execSimple(db, "BEGIN IMMEDIATE")) return false;

    sqlite3_stmt *stmt = nullptr;
    std::string jobId, documentId;
    {
        const char *sql = "SELECT id, document_id FROM ai_jobs WHERE status='queued' "
                           "ORDER BY created_at ASC LIMIT 1";
        if (sqlite3_prepare_v2(db, sql, -1, &stmt, nullptr) != SQLITE_OK) {
            execSimple(db, "ROLLBACK");
            return false;
        }
        if (sqlite3_step(stmt) != SQLITE_ROW) {
            sqlite3_finalize(stmt);
            execSimple(db, "ROLLBACK");
            return false; // fila vazia
        }
        jobId = reinterpret_cast<const char *>(sqlite3_column_text(stmt, 0));
        documentId = reinterpret_cast<const char *>(sqlite3_column_text(stmt, 1));
        sqlite3_finalize(stmt);
    }

    const std::string now = nowIso8601();

    {
        const char *sql = "UPDATE ai_jobs SET status='claimed', claimed_by=?, updated_at=? WHERE id=?";
        sqlite3_prepare_v2(db, sql, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, workerId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 2, now.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 3, jobId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_step(stmt);
        sqlite3_finalize(stmt);
    }
    {
        const char *sql = "UPDATE documents SET status='processing', updated_at=? WHERE id=?";
        sqlite3_prepare_v2(db, sql, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, now.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 2, documentId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_step(stmt);
        sqlite3_finalize(stmt);
    }

    std::string localPath, fileName;
    {
        const char *sql = "SELECT local_path, file_name FROM documents WHERE id=?";
        sqlite3_prepare_v2(db, sql, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, documentId.c_str(), -1, SQLITE_TRANSIENT);
        if (sqlite3_step(stmt) == SQLITE_ROW) {
            localPath = reinterpret_cast<const char *>(sqlite3_column_text(stmt, 0));
            fileName = reinterpret_cast<const char *>(sqlite3_column_text(stmt, 1));
        }
        sqlite3_finalize(stmt);
    }

    if (!execSimple(db, "COMMIT")) return false;

    out = {jobId, documentId, localPath, fileName};
    return true;
}

struct ProcessResult {
    bool ok = true;
    std::string documentType;
    std::string ocrExcerpt;
    double confidence = 0.0;
    std::string errorMessage;
};

namespace {
constexpr size_t kMaxExcerptChars = 4000;
}

// extractPdfTextFromStream: percorre os tokens de um content stream já
// descodificado à procura de texto real — não tenta perceber os
// operadores PDF um a um (Tj, TJ, ', "), porque na sintaxe de content
// stream só esses operadores de mostrar texto é que alguma vez levam uma
// string literal "(...)" como operando; por isso basta apanhar toda a
// string literal encontrada entre um BT e o ET correspondente. O pdfio
// devolve strings literais já com os escapes resolvidos, com o "(" inicial
// incluído e o ")" final removido (ver pdfio-token.c) — daí o token+1.
void extractPdfTextFromStream(pdfio_stream_t *st, std::string &out) {
    char token[8192];
    bool inText = false;

    while (pdfioStreamGetToken(st, token, sizeof(token))) {
        if (out.size() >= kMaxExcerptChars) continue;

        if (std::strcmp(token, "BT") == 0) {
            inText = true;
        } else if (std::strcmp(token, "ET") == 0) {
            inText = false;
            if (!out.empty() && out.back() != '\n') out += '\n';
        } else if (inText && token[0] == '(') {
            if (!out.empty() && out.back() != '\n' && out.back() != ' ') out += ' ';
            out += (token + 1);
        }
    }
}

// extractPdfText: texto REAL extraído da camada de texto do PDF, quando
// existir — nunca inventado. Devolve string vazia (não uma frase de
// substituição) quando o PDF não tem camada de texto (ex.: um design
// "achatado" tipo Canva, só gráficos/imagens vectorizadas — visto em
// primeira mão na biblioteca real de teste, "Blue Black Creative Modern
// Business Card.pdf") ou quando não é sequer possível abrir o ficheiro
// (encriptado sem password, corrompido, etc.) — ver ROADMAP.md do plano
// de IA local, "Tier 1" deliberadamente não tenta OCR de imagens/páginas
// sem texto embutido, isso fica para uma fase futura separada.
std::string extractPdfText(const std::string &path) {
    auto ignoreError = [](pdfio_file_t *, const char *, void *) -> bool { return false; };

    pdfio_file_t *pdf = pdfioFileOpen(path.c_str(), nullptr, nullptr, ignoreError, nullptr);
    if (pdf == nullptr) return "";

    std::string text;
    size_t numPages = pdfioFileGetNumPages(pdf);
    for (size_t p = 0; p < numPages && text.size() < kMaxExcerptChars; ++p) {
        pdfio_obj_t *page = pdfioFileGetPage(pdf, p);
        if (page == nullptr) continue;

        size_t numStreams = pdfioPageGetNumStreams(page);
        for (size_t s = 0; s < numStreams && text.size() < kMaxExcerptChars; ++s) {
            pdfio_stream_t *st = pdfioPageOpenStream(page, s, /*decode=*/true);
            if (st == nullptr) continue;
            extractPdfTextFromStream(st, text);
            pdfioStreamClose(st);
        }
    }

    pdfioFileClose(pdf);

    if (text.size() > kMaxExcerptChars) text.resize(kMaxExcerptChars);
    return text;
}

// processDocument: tipo de documento continua a vir da extensão do
// ficheiro (inferDocumentType) — só o excerto de texto passou a ser real
// (extractPdfText), para PDFs com camada de texto. A pequena espera
// aleatória fica tal e qual estava (ver ROADMAP.md, Fase 4 — é o que
// torna "processing" visível na UI e testável ao matar um worker a
// meio), independente do tempo que a extração real leve.
ProcessResult processDocument(const ClaimedJob &job, std::mt19937 &rng, int workMinMs, int workMaxMs) {
    std::uniform_int_distribution<int> workMs(workMinMs, workMaxMs);
    std::this_thread::sleep_for(std::chrono::milliseconds(workMs(rng)));

    ProcessResult r;
    r.documentType = inferDocumentType(job.fileName);

    if (r.documentType == "PDF") {
        r.ocrExcerpt = extractPdfText(job.localPath);
    }

    // Texto extraído a sério (não uma probabilidade de leitura de
    // imagem) vale confiança máxima; sem texto nenhum, 0 — nunca um
    // número inventado no meio, isso enganaria quem olhasse para a
    // coluna ocr_confidence a pensar que é OCR de imagem.
    r.confidence = r.ocrExcerpt.empty() ? 0.0 : 1.0;
    return r;
}

void finishJob(sqlite3 *db, const ClaimedJob &job, const ProcessResult &result) {
    const std::string now = nowIso8601();
    sqlite3_stmt *stmt = nullptr;

    execSimple(db, "BEGIN IMMEDIATE");

    if (result.ok) {
        std::ostringstream resultJson;
        resultJson << "{\"document_type\":\"" << escapeJson(result.documentType)
                    << "\",\"ocr_excerpt\":\"" << escapeJson(result.ocrExcerpt)
                    << "\",\"ocr_confidence\":" << result.confidence << "}";
        const std::string rj = resultJson.str();

        const char *sql1 = "UPDATE ai_jobs SET status='done', result_json=?, updated_at=? WHERE id=?";
        sqlite3_prepare_v2(db, sql1, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, rj.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 2, now.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 3, job.jobId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_step(stmt);
        sqlite3_finalize(stmt);

        const char *sql2 = "UPDATE documents SET status='pending_review', document_type=?, "
                            "ocr_excerpt=?, ocr_confidence=?, updated_at=? WHERE id=?";
        sqlite3_prepare_v2(db, sql2, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, result.documentType.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 2, result.ocrExcerpt.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_double(stmt, 3, result.confidence);
        sqlite3_bind_text(stmt, 4, now.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 5, job.documentId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_step(stmt);
        sqlite3_finalize(stmt);
    } else {
        const char *sql1 = "UPDATE ai_jobs SET status='failed', error=?, updated_at=? WHERE id=?";
        sqlite3_prepare_v2(db, sql1, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, result.errorMessage.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 2, now.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 3, job.jobId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_step(stmt);
        sqlite3_finalize(stmt);

        const char *sql2 = "UPDATE documents SET status='failed', updated_at=? WHERE id=?";
        sqlite3_prepare_v2(db, sql2, -1, &stmt, nullptr);
        sqlite3_bind_text(stmt, 1, now.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_bind_text(stmt, 2, job.documentId.c_str(), -1, SQLITE_TRANSIENT);
        sqlite3_step(stmt);
        sqlite3_finalize(stmt);
    }

    execSimple(db, "COMMIT");
}

// notifyCallback avisa o Go de que algo mudou para este documento —
// nunca escreve no SQLite, é só o sinal para o Go reler a linha e
// emitir o evento Wails. Falhas de rede são só logadas: a base de
// dados já reflecte o estado real, o pior que acontece é a UI só
// actualizar no próximo refresh manual.
void notifyCallback(const std::string &callbackUrl, const std::string &documentId) {
    ParsedUrl parsed = parseUrl(callbackUrl);
    httplib::Client cli(parsed.base);
    cli.set_connection_timeout(2, 0);
    cli.set_read_timeout(2, 0);

    std::string body = "{\"document_id\":\"" + escapeJson(documentId) + "\"}";
    auto res = cli.Post(parsed.path, body, "application/json");
    if (!res) {
        std::cerr << "callback POST falhou: " << httplib::to_string(res.error()) << "\n";
    } else if (res->status >= 400) {
        std::cerr << "callback POST devolveu estado " << res->status << "\n";
    }
}

// sendHeartbeat avisa o Go que este worker ainda está vivo — permite
// libertar os jobs dele mais depressa que a lease de 90s se parar de
// chegar (ver internal/heartbeats no lado Go). Tal como notifyCallback,
// falhas de rede são só logadas.
void sendHeartbeat(const std::string &heartbeatUrl, const std::string &workerId) {
    ParsedUrl parsed = parseUrl(heartbeatUrl);
    httplib::Client cli(parsed.base);
    cli.set_connection_timeout(2, 0);
    cli.set_read_timeout(2, 0);

    std::string body = "{\"worker_id\":\"" + escapeJson(workerId) + "\"}";
    auto res = cli.Post(parsed.path, body, "application/json");
    if (!res) {
        std::cerr << "heartbeat POST falhou: " << httplib::to_string(res.error()) << "\n";
    } else if (res->status >= 400) {
        std::cerr << "heartbeat POST devolveu estado " << res->status << "\n";
    }
}

} // namespace

int main(int argc, char **argv) {
    Args args = parseArgs(argc, argv);
    if (args.dbPath.empty() || args.callbackUrl.empty() || args.heartbeatUrl.empty() ||
        args.workerId.empty()) {
        std::cerr << "uso: ai-worker --db <path> --callback <url> --heartbeat-url <url> "
                     "--worker-id <id> [--idle-timeout-ms N] [--poll-interval-ms N] "
                     "[--heartbeat-interval-ms N] [--work-min-ms N] [--work-max-ms N]\n";
        return 2;
    }

    sqlite3 *db = nullptr;
    if (sqlite3_open(args.dbPath.c_str(), &db) != SQLITE_OK) {
        std::cerr << "falha a abrir a base de dados: " << sqlite3_errmsg(db) << "\n";
        return 1;
    }
    sqlite3_busy_timeout(db, 5000);

    std::random_device rd;
    std::mt19937 rng(rd());

    std::cout << "ai-worker " << args.workerId << " a arrancar (db=" << args.dbPath << ")\n";

    // Thread de heartbeat: corre independentemente do loop principal
    // para continuar a provar que o processo está vivo mesmo durante um
    // job longo (a simulação actual é curta, mas o OCR real não vai
    // ser).
    std::atomic<bool> running{true};
    std::thread heartbeatThread([&]() {
        while (running.load()) {
            sendHeartbeat(args.heartbeatUrl, args.workerId);
            std::this_thread::sleep_for(std::chrono::milliseconds(args.heartbeatIntervalMs));
        }
    });

    auto idleSince = std::chrono::steady_clock::now();
    while (true) {
        ClaimedJob job;
        if (!claimNext(db, args.workerId, job)) {
            auto idleMs = std::chrono::duration_cast<std::chrono::milliseconds>(
                              std::chrono::steady_clock::now() - idleSince)
                              .count();
            if (idleMs > args.idleTimeoutMs) {
                std::cout << "ai-worker " << args.workerId << " ocioso, a terminar\n";
                break;
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(args.pollIntervalMs));
            continue;
        }

        idleSince = std::chrono::steady_clock::now();
        std::cout << "ai-worker " << args.workerId << " a processar job " << job.jobId
                   << " (documento " << job.documentId << ")\n";
        notifyCallback(args.callbackUrl, job.documentId); // avisa transição para "processing"

        ProcessResult result = processDocument(job, rng, args.workMinMs, args.workMaxMs);
        finishJob(db, job, result);
        notifyCallback(args.callbackUrl, job.documentId); // avisa resultado final
    }

    running = false;
    heartbeatThread.join();
    sqlite3_close(db);
    return 0;
}
