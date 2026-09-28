// search-service: processo único, sempre ativo, que mantém em memória um
// índice de embeddings dos documentos já processados pelo ai-worker, e
// responde a pesquisas síncronas do Go por HTTP loopback. Nunca toca no
// SQLite directamente — o Go envia o texto a indexar via
// /internal/index/upsert, mantendo o SQLite como única fonte de verdade
// (ver internal/searchservice no lado Go).
//
// RECALIBRADO em 2026-09-28: já não calcula os seus próprios embeddings
// por hashing (feature hashing / bag-of-words, ver histórico git) — essa
// técnica não tinha noção nenhuma de significado, só de sobreposição
// exacta de palavras, e o Go também descobriu nesta ronda que estava a
// ignorar o texto real extraído dos documentos ao indexar (ver
// internal/documents.BuildIndexText). Com as duas coisas corrigidas ao
// mesmo tempo, faz mais sentido usar o MESMO modelo de embeddings
// semânticos que já corre no llm-service (paraphrase-multilingual-
// mpnet-base-v2) para pesquisa, em vez de um segundo motor mais fraco
// aqui. Este processo deixa de calcular embeddings — só guarda vectores
// já calculados pelo Go (que os pede ao llm-service) e faz a
// comparação de cosseno, que é a parte que precisa de estar sempre viva
// e responder depressa.
//
// Protocolo (texto simples, não JSON — só dois consumidores, ambos
// controlados por nós, por isso um parser JSON só acrescentava trabalho):
//   GET  /internal/health       -> "ok"
//   POST /internal/index/upsert -> cabeçalho X-Document-Id, corpo =
//        vector normalizado, floats separados por vírgula (ver
//        internal/llmservice, endpoint /embed, que calcula este vector)
//   POST /search                -> corpo = vector da pergunta/busca,
//        mesmo formato; resposta: até 20 linhas "<document_id>\t<score>",
//        por ordem decrescente de pontuação (cosseno). Decidir se isto é
//        "resposta única", "lista" ou "nada" já não é responsabilidade
//        deste processo — o Go é quem tem o texto original da pergunta
//        para essa heurística (ver classifyResultType em
//        internal/searchservice/searchservice.go); aqui simplifica para
//        só devolver a lista ordenada.
//
// Ver ROADMAP.md, Fase 5, e internal/searchservice/searchservice.go para
// o outro lado.

#define CPPHTTPLIB_THREAD_POOL_COUNT 4
#include <httplib.h>

#include <algorithm>
#include <cmath>
#include <cstdint>
#include <iostream>
#include <mutex>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>

namespace {

using Vector = std::vector<float>;

// parseVector descodifica o formato "f1,f2,f3,..." — o mesmo que
// llm-service/src/main.cpp (encodeVector) produz e que
// internal/searchservice/searchservice.go usa nos dois sentidos. Valores
// que não parseiam viram 0.0f em vez de abortar o pedido inteiro — um
// vector parcialmente corrompido ainda é mais útil que rejeitar tudo, e
// isto nunca devia acontecer de qualquer forma (só nós escrevemos este
// formato, dos dois lados).
Vector parseVector(const std::string &s) {
    Vector v;
    size_t start = 0;
    while (start <= s.size()) {
        size_t comma = s.find(',', start);
        std::string tok = (comma == std::string::npos) ? s.substr(start) : s.substr(start, comma - start);
        if (!tok.empty()) {
            try {
                v.push_back(std::stof(tok));
            } catch (...) {
                v.push_back(0.0f);
            }
        }
        if (comma == std::string::npos) break;
        start = comma + 1;
    }
    return v;
}

float cosine(const Vector &a, const Vector &b) {
    float dot = 0.0f;
    size_t n = std::min(a.size(), b.size());
    for (size_t i = 0; i < n; ++i) dot += a[i] * b[i];
    return dot;
}

// Índice em memória: id do documento -> vector (já normalizado pelo
// llm-service, ver embed() lá). O texto original não precisa de ficar
// guardado aqui — a fonte de verdade do conteúdo continua a ser o SQLite
// do lado Go, que é quem monta a resposta final.
class Index {
public:
    void upsert(const std::string &id, const Vector &vec) {
        std::lock_guard<std::mutex> lock(mu_);
        entries_[id] = vec;
    }

    struct Hit {
        std::string id;
        float score;
    };

    std::vector<Hit> rank(const Vector &query) const {
        std::lock_guard<std::mutex> lock(mu_);
        std::vector<Hit> hits;
        hits.reserve(entries_.size());
        for (const auto &[id, vec] : entries_) {
            hits.push_back({id, cosine(query, vec)});
        }
        std::sort(hits.begin(), hits.end(), [](const Hit &a, const Hit &b) { return a.score > b.score; });
        return hits;
    }

    size_t size() const {
        std::lock_guard<std::mutex> lock(mu_);
        return entries_.size();
    }

private:
    mutable std::mutex mu_;
    std::unordered_map<std::string, Vector> entries_;
};

struct ParsedListen {
    std::string host;
    int port;
};

ParsedListen parseListen(const std::string &listen) {
    auto colon = listen.find_last_of(':');
    if (colon == std::string::npos) return {listen, 0};
    return {listen.substr(0, colon), std::stoi(listen.substr(colon + 1))};
}

} // namespace

int main(int argc, char **argv) {
    std::string listen = "127.0.0.1:0";
    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--listen" && i + 1 < argc) listen = argv[++i];
    }
    ParsedListen addr = parseListen(listen);

    Index index;
    httplib::Server svr;

    svr.Get("/internal/health", [](const httplib::Request &, httplib::Response &res) {
        res.set_content("ok", "text/plain");
    });

    svr.Post("/internal/index/upsert", [&](const httplib::Request &req, httplib::Response &res) {
        auto it = req.headers.find("X-Document-Id");
        if (it == req.headers.end() || it->second.empty()) {
            res.status = 400;
            res.set_content("falta o cabecalho X-Document-Id", "text/plain");
            return;
        }
        index.upsert(it->second, parseVector(req.body));
        res.status = 204;
    });

    svr.Post("/search", [&](const httplib::Request &req, httplib::Response &res) {
        Vector query = parseVector(req.body);
        auto hits = index.rank(query);

        std::ostringstream out;
        constexpr size_t kMaxResults = 20;
        size_t n = std::min(hits.size(), kMaxResults);
        for (size_t i = 0; i < n; ++i) {
            out << hits[i].id << '\t' << hits[i].score << '\n';
        }

        res.set_content(out.str(), "text/plain; charset=utf-8");
    });

    std::cout << "search-service a arrancar em " << addr.host << ":" << addr.port << "\n";
    if (!svr.listen(addr.host, addr.port)) {
        std::cerr << "search-service: falha a ligar em " << listen << "\n";
        return 1;
    }
    return 0;
}
