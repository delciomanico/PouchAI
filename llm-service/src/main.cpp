// llm-service — Marco 1: protocolo HTTP real sobre os dois modelos GGUF.
// Ver plano em unified-popping-sunset.md e shared/vendor/README.md para o
// porquê do vendoring/flags de build; ver search-service/src/main.cpp para
// o mesmo molde de processo único + protocolo texto simples.
//
// Dois modelos, dois papéis (decisão explícita do utilizador, ver plano):
//   - modelo de chat/instruct: só tarefas genuinamente geradoras
//     (resumo, resposta de chat).
//   - modelo de embeddings: classificação de tipo/tags por comparação de
//     cosseno contra listas fixas de candidatos, embebidos uma vez no
//     arranque (zero-shot, determinístico, não depende do modelo pequeno
//     seguir um formato de saída à risca).
//
// Protocolo (texto simples, não JSON — só nós falamos os dois lados):
//   GET  /internal/health -> "ok" (só depois dos dois modelos carregados)
//   POST /summarize -> corpo: "<nome-do-ficheiro>\n---\n<texto extraído>"
//                       resposta: resumo gerado em 1-2 frases, texto simples
//   POST /classify  -> corpo: "<nome-do-ficheiro>\n---\n<texto extraído>"
//                       resposta: linha 1 = tipo de documento
//                                 linha 2 = tags separadas por vírgula
//   POST /chat      -> corpo: "<pergunta>\n---CONTEXT---\n<excertos>"
//                       resposta: texto gerado
//   POST /embed     -> corpo: texto livre
//                       resposta: vector normalizado, floats separados
//                       por vírgula — ver comentário em embed() acima e
//                       search-service/src/main.cpp, que já não calcula
//                       os seus próprios embeddings (Fase de indexação
//                       semântica): pede-os aqui, ao mesmo modelo já
//                       usado para classificar, em vez de ter um segundo
//                       "motor de embeddings" tosco (hashing) a viver
//                       noutro processo.
//
// Inferência serializada por mutex, um por modelo — nunca dois pedidos em
// simultâneo no mesmo llama_context (o httplib por omissão usa várias
// threads; ver CPPHTTPLIB_THREAD_POOL_COUNT abaixo).

#define CPPHTTPLIB_THREAD_POOL_COUNT 2
#include <httplib.h>

#include "llama.h"

#include <algorithm>
#include <chrono>
#include <cmath>
#include <cstdio>
#include <cstring>
#include <iostream>
#include <mutex>
#include <sstream>
#include <string>
#include <thread>
#include <utility>
#include <vector>

namespace {

constexpr const char *kFileTextSep = "\n---\n";
constexpr const char *kChatContextSep = "\n---CONTEXT---\n";
constexpr size_t kMaxInputChars = 6000;
// Medido em primeira mão com um documento real (fatura, 801 caracteres
// de OCR): com 180 tokens de orcamento e um pedido de "1-2 frases", o
// modelo nao parou sozinho (sem EOS) e excedeu um timeout de 180s no
// lado Go. 80 tokens + pedir explicitamente UMA frase (ver system prompt
// abaixo) limita bem melhor o pior caso.
constexpr int kMaxNewTokensSummary = 80;
// 350 permitia respostas longas de sobra, mas sem o modelo emitir EOS
// cedo (comum em perguntas fora do formato habitual, ex. perguntas sobre
// a colecção em si), a geração corre até ao fim do orçamento sempre —
// medido em CPU sem GPU (Qwen2.5-1.5B-Instruct): ultrapassou 4 minutos
// numa pergunta real. 120 tokens chega para 2-4 frases (o que se pede no
// prompt de sistema, ver buildPrompt) e limita o pior caso a algo
// razoável.
constexpr int kMaxNewTokensChat = 120;
// Limiares de classificação zero-shot: um limiar absoluto de cosseno NÃO
// chega sozinho — modelos de embeddings de frase têm anisotropia
// conhecida, em que a nuvem de vetores fica concentrada numa direcção
// dominante, fazendo com que a similaridade de cosseno entre um texto
// SEM sinal nenhum e qualquer candidato fixo já venha alta por si só. Por
// isso exige-se também uma MARGEM entre o melhor candidato e o resto.
//
// RECALIBRADO em 2026-09-28, segunda vez no mesmo dia: modelo de
// embeddings trocado de multilingual-e5-small para
// paraphrase-multilingual-mpnet-base-v2 (ver comentário em embed() acima)
// — a primeira calibração (e5, valores 0.85/0.003/0.85/0.004) fica só no
// histórico da sessão, não se aplica a este modelo.
//
// Com o mpnet, o sinal é claramente mais forte: um currículo real
// corretamente classificado teve margem 0.184 entre 1º e 2º candidato
// (contra 0.0032 com o e5 — quase 60x maior). MAS a distinção continua
// imperfeita para casos menos óbvios: um cartão de visita corretamente
// classificado ("Cartão de visita") teve margem só 0.026, e texto sem
// sentido nenhum atingiu uma margem parecida (0.026) para uma categoria
// errada, e uma fatura real teve margem 0.028 para uma categoria ERRADA
// (perdeu "Fatura" para "Documento de identificação"). Ou seja: só os
// casos MUITO claros se distinguem de forma inequívoca do ruído; casos
// ambíguos continuam indistinguíveis por margem sozinha.
//
// Decisão explícita (consistente com a política de nunca inventar já
// usada no resto do serviço): margem alta o suficiente para só aceitar
// os casos muito claros, mesmo que isso signifique cair em "Outro" nos
// ambíguos (incluindo o cartão de visita, que teria acertado com uma
// margem mais baixa) — preferível a arriscar etiquetas erradas com
// confiança. O floor absoluto deixa de ser útil como filtro de conteúdo
// (a fatura real teve o score absoluto mais BAIXO dos 4 testes, mais
// baixo até que o texto sem sentido) — serve só de rede de segurança
// contra um vector degenerado.
constexpr float kCategoryAbsoluteFloor = 0.20f;
constexpr float kCategoryMargin = 0.05f;
constexpr float kTagThreshold = 0.25f;
constexpr float kTagMarginAboveMean = 0.15f;
constexpr size_t kMaxTags = 5;

// kGenerationDeadline: limite de tempo de PAREDE (não de tokens) para uma
// geração — sem isto, um documento real cujo texto (ex. OCR com muitos
// números/boilerplate repetido) faça o modelo nunca emitir EOS pode
// correr o orçamento de tokens inteiro a uma velocidade imprevisível,
// segurando o mutex do modelo indefinidamente (medido em primeira mão:
// um /summarize de uma factura real ultrapassou os 10 minutos e nunca
// devolveu nada, porque só o timeout do CLIENTE Go desistia — o processo
// C++ continuava a gerar sem saber que ninguém estava à espera). Ao
// atingir o limite, devolve-se o que já foi gerado até ali (pode ser
// texto vazio) em vez de nunca responder — degradação, não erro fatal.
constexpr auto kGenerationDeadline = std::chrono::seconds(90);

// ---------- utilidades de texto ----------

std::string trim(const std::string &s) {
    size_t start = s.find_first_not_of(" \t\r\n");
    if (start == std::string::npos) return "";
    size_t end = s.find_last_not_of(" \t\r\n");
    return s.substr(start, end - start + 1);
}

std::string truncateChars(const std::string &s, size_t maxChars) {
    if (s.size() <= maxChars) return s;
    return s.substr(0, maxChars) + " [...texto truncado...]";
}

// separa "cabecalho<sep>resto" — usado por /summarize, /classify (nome do
// ficheiro + texto) e /chat (pergunta + contexto).
std::pair<std::string, std::string> splitOnSep(const std::string &raw, const std::string &sep) {
    auto pos = raw.find(sep);
    if (pos == std::string::npos) return {trim(raw), ""};
    return {trim(raw.substr(0, pos)), trim(raw.substr(pos + sep.size()))};
}

struct ParsedListen {
    std::string host;
    int port;
};

ParsedListen parseListen(const std::string &listen) {
    auto colon = listen.find_last_of(':');
    if (colon == std::string::npos) return {listen, 0};
    return {listen.substr(0, colon), std::stoi(listen.substr(colon + 1))};
}

// ---------- wrapper de um modelo llama.cpp ----------

class LlamaModel {
public:
    bool load(const std::string &path, bool wantEmbeddings, uint32_t nCtx) {
        llama_model_params mp = llama_model_default_params();
        mp.n_gpu_layers = 0;  // só CPU, ver CMakeLists.txt (GGML_CUDA/VULKAN/METAL=OFF)

        model_ = llama_model_load_from_file(path.c_str(), mp);
        if (!model_) return false;
        vocab_ = llama_model_get_vocab(model_);

        unsigned threads = std::max(1u, std::thread::hardware_concurrency());

        llama_context_params cp = llama_context_default_params();
        cp.n_ctx = nCtx;
        cp.n_batch = nCtx;
        cp.n_ubatch = nCtx;
        cp.n_threads = (int32_t)threads;
        cp.n_threads_batch = (int32_t)threads;
        cp.embeddings = wantEmbeddings;
        if (wantEmbeddings) cp.pooling_type = LLAMA_POOLING_TYPE_MEAN;

        ctx_ = llama_init_from_model(model_, cp);
        if (!ctx_) return false;

        nCtx_ = nCtx;
        return true;
    }

    ~LlamaModel() {
        if (ctx_) llama_free(ctx_);
        if (model_) llama_model_free(model_);
    }

    llama_model *model() const { return model_; }
    const llama_vocab *vocab() const { return vocab_; }
    llama_context *ctx() const { return ctx_; }
    uint32_t nCtx() const { return nCtx_; }
    std::mutex &mutex() { return mu_; }

private:
    llama_model *model_ = nullptr;
    const llama_vocab *vocab_ = nullptr;
    llama_context *ctx_ = nullptr;
    uint32_t nCtx_ = 0;
    std::mutex mu_;
};

std::vector<llama_token> tokenize(const llama_vocab *vocab, const std::string &text,
                                   bool addSpecial, bool parseSpecial) {
    int need = -llama_tokenize(vocab, text.c_str(), (int)text.size(), nullptr, 0, addSpecial, parseSpecial);
    if (need <= 0) return {};
    std::vector<llama_token> tokens(need);
    llama_tokenize(vocab, text.c_str(), (int)text.size(), tokens.data(), need, addSpecial, parseSpecial);
    return tokens;
}

std::string tokenToPiece(const llama_vocab *vocab, llama_token tok) {
    char buf[256];
    int n = llama_token_to_piece(vocab, tok, buf, sizeof(buf), 0, true);
    if (n < 0) return "";
    return std::string(buf, n);
}

// Aplica o chat template embutido no GGUF (llama_model_chat_template); cai
// para um formato simples se o modelo não trouxer nenhum (não devia
// acontecer com os modelos instruct recomendados no plano, mas mantém o
// serviço a degradar em vez de crashar).
std::string buildPrompt(LlamaModel &m, const std::string &system, const std::string &user) {
    const char *tmpl = llama_model_chat_template(m.model(), nullptr);
    if (!tmpl) {
        return "System: " + system + "\nUser: " + user + "\nAssistant:";
    }

    llama_chat_message msgs[2] = {
        {"system", system.c_str()},
        {"user", user.c_str()},
    };

    std::vector<char> buf(system.size() + user.size() + 512);
    int n = llama_chat_apply_template(tmpl, msgs, 2, true, buf.data(), (int)buf.size());
    if (n < 0) return "";
    if ((size_t)n > buf.size()) {
        buf.resize(n);
        n = llama_chat_apply_template(tmpl, msgs, 2, true, buf.data(), (int)buf.size());
        if (n < 0) return "";
    }
    return std::string(buf.data(), n);
}

// Gera texto a partir de um prompt já formatado, token a token, até
// encontrar um token de fim-de-geração ou atingir maxNewTokens. Serializado
// por mutex — nunca duas gerações em simultâneo no mesmo llama_context.
std::string generate(LlamaModel &m, const std::string &prompt, int maxNewTokens) {
    std::lock_guard<std::mutex> lock(m.mutex());
    llama_memory_clear(llama_get_memory(m.ctx()), true);

    auto tokens = tokenize(m.vocab(), prompt, /*addSpecial=*/true, /*parseSpecial=*/true);
    if (tokens.empty()) return "";

    // margem de segurança: garantir espaço para a geração dentro de n_ctx,
    // cortando o INÍCIO do prompt se for preciso (mantém o fim, mais
    // próximo da pergunta/instrução).
    size_t budget = m.nCtx() > (size_t)maxNewTokens + 16 ? m.nCtx() - maxNewTokens - 16 : m.nCtx() / 2;
    if (tokens.size() > budget) {
        tokens.erase(tokens.begin(), tokens.begin() + (tokens.size() - budget));
    }

    llama_sampler_chain_params sp = llama_sampler_chain_default_params();
    llama_sampler *smpl = llama_sampler_chain_init(sp);
    llama_sampler_chain_add(smpl, llama_sampler_init_top_k(40));
    llama_sampler_chain_add(smpl, llama_sampler_init_top_p(0.9f, 1));
    llama_sampler_chain_add(smpl, llama_sampler_init_temp(0.7f));
    llama_sampler_chain_add(smpl, llama_sampler_init_dist(1234));

    std::string out;
    llama_batch batch = llama_batch_get_one(tokens.data(), (int32_t)tokens.size());
    llama_token curTok = -1;
    auto deadline = std::chrono::steady_clock::now() + kGenerationDeadline;

    for (int i = 0; i < maxNewTokens; ++i) {
        if (std::chrono::steady_clock::now() >= deadline) break;
        if (llama_decode(m.ctx(), batch) != 0) break;

        curTok = llama_sampler_sample(smpl, m.ctx(), -1);
        if (llama_vocab_is_eog(m.vocab(), curTok)) break;

        out += tokenToPiece(m.vocab(), curTok);
        llama_sampler_accept(smpl, curTok);

        batch = llama_batch_get_one(&curTok, 1);
    }

    llama_sampler_free(smpl);
    return trim(out);
}

// Embebe um texto e devolve o vector normalizado (norma 1) — comparar por
// cosseno depois passa a ser só um produto escalar.
//
// prefix: TROCADO em 2026-09-28 de multilingual-e5-small para
// paraphrase-multilingual-mpnet-base-v2 (sentence-transformers, base
// XLM-RoBERTa, pooling por média — mesma configuração já usada aqui).
// Ao contrário do e5 (modelo de RECUPERAÇÃO, treinado com prefixos
// "query: "/"passage: " para tarefas assimétricas pergunta→documento),
// este é um modelo de SIMILARIDADE SEMÂNTICA DIRETA — treinado para
// que frases parecidas fiquem perto no espaço de embeddings, sem
// nenhum prefixo, o que encaixa muito melhor na tarefa real aqui
// (comparar um documento contra descrições fixas de categorias por
// cosseno). O parâmetro fica para não mudar a assinatura da função,
// mas chama-se sempre com "" agora — ver chamadas abaixo.
std::vector<float> embed(LlamaModel &m, const std::string &prefix, const std::string &text) {
    std::lock_guard<std::mutex> lock(m.mutex());
    int nEmbd = llama_model_n_embd(m.model());

    std::string prefixed = prefix + text;
    auto tokens = tokenize(m.vocab(), prefixed, /*addSpecial=*/true, /*parseSpecial=*/false);
    if (tokens.empty()) return std::vector<float>(nEmbd, 0.0f);
    if (tokens.size() > m.nCtx()) tokens.resize(m.nCtx());

    // llama_encode (não llama_decode) é a API correta para um contexto de
    // embeddings: não usa KV cache, cada chamada é independente por
    // definição (ver llama.h). Usar llama_decode aqui fazia o llama.cpp
    // redirecionar silenciosamente para encode() por baixo (mensagem
    // "cannot decode batches with this context"), mas esse caminho de
    // compatibilidade devolvia embeddings praticamente idênticos para
    // textos completamente diferentes — medido em 2026-09-28: 18 de 20
    // tags candidatas davam a MESMA pontuação de cosseno ao milésimo,
    // fosse qual fosse o documento. Chamar llama_encode diretamente
    // resolveu.
    if (llama_encode(m.ctx(), llama_batch_get_one(tokens.data(), (int32_t)tokens.size())) != 0) {
        return std::vector<float>(nEmbd, 0.0f);
    }

    float *raw = llama_get_embeddings_seq(m.ctx(), 0);
    std::vector<float> vec(nEmbd, 0.0f);
    if (raw) std::copy(raw, raw + nEmbd, vec.begin());

    float norm = 0.0f;
    for (float x : vec) norm += x * x;
    norm = std::sqrt(norm);
    if (norm > 1e-9f) {
        for (float &x : vec) x /= norm;
    }
    return vec;
}

float dot(const std::vector<float> &a, const std::vector<float> &b) {
    float s = 0.0f;
    for (size_t i = 0; i < a.size() && i < b.size(); ++i) s += a[i] * b[i];
    return s;
}

// encodeVector serializa um vector como floats separados por vírgula —
// 9 dígitos significativos chegam para um float32 dar a volta (round-trip)
// sem perdas, sem gastar espaço com dígitos que não fazem diferença
// nenhuma (ver internal/searchservice/searchservice.go, que descodifica
// exactamente este formato).
std::string encodeVector(const std::vector<float> &v) {
    std::ostringstream out;
    out.precision(9);
    for (size_t i = 0; i < v.size(); ++i) {
        if (i) out << ",";
        out << v[i];
    }
    return out.str();
}

// ---------- listas fixas de candidatos para classificação zero-shot ----------

struct Candidate {
    std::string label;
    std::vector<float> vec;
};

std::vector<Candidate> g_categories;  // "Outro" NÃO entra aqui — é o
                                       // fallback quando nada bate o limiar.
std::vector<Candidate> g_tags;

// TENTATIVA REJEITADA (2026-09-28): centrar cada vector (candidato e
// documento) subtraindo o vector médio do grupo, para remover a direção
// dominante partilhada por qualquer texto (anisotropia conhecida deste
// modelo — mesmo um texto sem sentido nenhum vinha com cosseno 0.88-0.94
// contra TODAS as categorias). Testado com documentos reais: piorou casos
// que já funcionavam (o currículo real deixou de ganhar para "Curriculo"
// e passou a ganhar para "Cartão de visita" por um triz). Uma correção
// de 1ª ordem (subtrair só a média) não chega quando o sinal real é tão
// pequeno — precisaria de whitening completo (PCA) para separar bem as
// direções, o que não compensa fazer artesanalmente aqui. Revertido a
// favor de comparar cosseno bruto com limiares mais realistas (ver
// kCategoryAbsoluteFloor/kCategoryMargin abaixo, recalibrados por medição
// direta com este modelo e documentos reais em vez do modelo de teste
// original).

void precomputeCandidates(LlamaModel &embedModel) {
    static const std::vector<std::pair<std::string, std::string>> categories = {
        {"Fatura", "fatura, nota fiscal, documento comercial que pede o pagamento de um valor por bens ou servicos prestados"},
        {"Recibo", "recibo, comprovativo de um pagamento ja efetuado"},
        {"Contrato", "contrato, acordo formal entre partes, com clausulas e assinaturas"},
        {"Cartao de visita", "cartao de visita, business card, com nome, cargo e contactos de uma pessoa ou empresa"},
        {"Curriculo", "curriculo, CV, curriculum vitae, com percurso profissional e academico de uma pessoa"},
        {"Carta", "carta, correspondencia formal ou pessoal dirigida a alguem"},
        {"Relatorio", "relatorio, documento com analise, resultados ou apresentacao de uma empresa ou projeto"},
        {"Documento de identificacao", "documento de identificacao, bilhete de identidade, cartao de cidadao, passaporte, numero de identificacao"},
    };
    for (const auto &c : categories) {
        g_categories.push_back({c.first, embed(embedModel, "", c.second)});
    }

    // Frases curtas, não palavras soltas: com o modelo e5 (que precisa do
    // prefixo "passage: "), uma palavra isolada tem só 1-2 tokens de
    // conteúdo contra 3 do prefixo — a média (mean pooling) fica dominada
    // pelo prefixo, e quase todas as tags colapsam para o mesmo vector
    // (medido em 2026-09-28: 18 de 20 tags davam a mesma pontuação de
    // cosseno ao milésimo, para qualquer documento). Uma frase mais longa
    // dilui o peso do prefixo o suficiente para o conteúdo discriminar.
    static const std::vector<std::pair<std::string, std::string>> tags = {
        {"financeiro", "documento financeiro, dinheiro, pagamentos, contas"},
        {"pessoal", "documento de uso pessoal, nao relacionado com trabalho"},
        {"trabalho", "documento relacionado com emprego ou atividade profissional"},
        {"identificacao", "documento que identifica uma pessoa, com nome e numero de identificacao"},
        {"contrato", "contrato, acordo com clausulas entre partes"},
        {"fatura", "fatura ou nota fiscal, pedido de pagamento por bens ou servicos"},
        {"design", "trabalho de design grafico ou visual, layout, arte"},
        {"saude", "documento relacionado com saude, medicina, hospital ou clinica"},
        {"educacao", "documento relacionado com educacao, escola ou universidade"},
        {"juridico", "documento juridico, legal, tribunal ou advogado"},
        {"imobiliario", "documento relacionado com imoveis, casas ou terrenos"},
        {"viagem", "documento relacionado com viagem, bilhete ou reserva"},
        {"seguro", "apolice ou documento de seguro"},
        {"orcamento", "orcamento, proposta de preco para um trabalho ou servico"},
        {"proposta", "proposta comercial ou de negocio"},
        {"apresentacao", "apresentacao de slides sobre uma empresa ou projeto"},
        {"curriculo", "curriculo profissional com percurso academico e experiencia"},
        {"cartao de visita", "cartao de visita com nome, cargo e contactos"},
        {"recibo", "recibo, comprovativo de pagamento ja efetuado"},
        {"relatorio", "relatorio com analise ou resultados de uma empresa ou projeto"},
    };
    for (const auto &t : tags) {
        g_tags.push_back({t.first, embed(embedModel, "", t.second)});
    }
}

std::string classifyType(const std::vector<float> &docVec) {
    float best = -1.0f, second = -1.0f;
    std::string bestLabel = "Outro";
    for (const auto &c : g_categories) {
        float score = dot(docVec, c.vec);
        if (score > best) {
            second = best;
            best = score;
            bestLabel = c.label;
        } else if (score > second) {
            second = score;
        }
    }
    if (best < kCategoryAbsoluteFloor || (best - second) < kCategoryMargin) return "Outro";
    return bestLabel;
}

std::vector<std::string> classifyTags(const std::vector<float> &docVec) {
    std::vector<std::pair<float, std::string>> scored;
    float sum = 0.0f;
    for (const auto &t : g_tags) {
        float score = dot(docVec, t.vec);
        scored.push_back({score, t.label});
        sum += score;
    }
    float mean = scored.empty() ? 0.0f : sum / (float)scored.size();
    std::sort(scored.begin(), scored.end(), [](const auto &a, const auto &b) { return a.first > b.first; });

    std::vector<std::string> out;
    for (size_t i = 0; i < scored.size() && i < kMaxTags; ++i) {
        if (scored[i].first < kTagThreshold) break;
        if (scored[i].first < mean + kTagMarginAboveMean) break;
        out.push_back(scored[i].second);
    }
    return out;
}

// ---------- argumentos da linha de comandos ----------

struct Args {
    std::string listen = "127.0.0.1:0";
    std::string chatModelPath;
    std::string embedModelPath;
    uint32_t chatCtx = 4096;
    uint32_t embedCtx = 512;
};

bool parseArgs(int argc, char **argv, Args &out) {
    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--listen" && i + 1 < argc) {
            out.listen = argv[++i];
        } else if (arg == "--chat-model" && i + 1 < argc) {
            out.chatModelPath = argv[++i];
        } else if (arg == "--embed-model" && i + 1 < argc) {
            out.embedModelPath = argv[++i];
        } else if (arg == "--chat-ctx" && i + 1 < argc) {
            out.chatCtx = (uint32_t)std::stoul(argv[++i]);
        } else if (arg == "--embed-ctx" && i + 1 < argc) {
            out.embedCtx = (uint32_t)std::stoul(argv[++i]);
        }
    }
    return !out.chatModelPath.empty() && !out.embedModelPath.empty();
}

}  // namespace

int main(int argc, char **argv) {
    Args args;
    if (!parseArgs(argc, argv, args)) {
        std::fprintf(stderr,
                     "uso: llm-service --chat-model <caminho.gguf> --embed-model <caminho.gguf> "
                     "[--listen host:port] [--chat-ctx n] [--embed-ctx n]\n");
        return 1;
    }

    llama_backend_init();

    LlamaModel chatModel;
    if (!chatModel.load(args.chatModelPath, /*wantEmbeddings=*/false, args.chatCtx)) {
        std::fprintf(stderr, "llm-service: falha a carregar o modelo de chat: %s\n", args.chatModelPath.c_str());
        llama_backend_free();
        return 1;
    }

    LlamaModel embedModel;
    if (!embedModel.load(args.embedModelPath, /*wantEmbeddings=*/true, args.embedCtx)) {
        std::fprintf(stderr, "llm-service: falha a carregar o modelo de embeddings: %s\n", args.embedModelPath.c_str());
        llama_backend_free();
        return 1;
    }

    std::fprintf(stderr, "llm-service: a pre-calcular embeddings das categorias/tags candidatas...\n");
    precomputeCandidates(embedModel);

    ParsedListen addr = parseListen(args.listen);
    httplib::Server svr;

    svr.Get("/internal/health", [](const httplib::Request &, httplib::Response &res) {
        res.set_content("ok", "text/plain");
    });

    svr.Post("/summarize", [&](const httplib::Request &req, httplib::Response &res) {
        auto [filename, text] = splitOnSep(req.body, kFileTextSep);

        std::string system =
            "Es um assistente que resume documentos em portugues, em UMA UNICA frase curta e "
            "factual (max 30 palavras), sem inventar informacao que nao esteja no texto "
            "fornecido. Nunca repitas o texto original nem escrevas mais que uma frase.";
        std::string user = "Nome do ficheiro: " + filename + "\n\n" +
                            (text.empty() ? std::string("(sem texto extraido deste documento)")
                                          : "Texto extraido:\n" + truncateChars(text, kMaxInputChars));

        std::string prompt = buildPrompt(chatModel, system, user);
        std::string answer = generate(chatModel, prompt, kMaxNewTokensSummary);
        res.set_content(answer, "text/plain; charset=utf-8");
    });

    svr.Post("/classify", [&](const httplib::Request &req, httplib::Response &res) {
        auto [filename, text] = splitOnSep(req.body, kFileTextSep);

        // Sem texto extraído real, o nome do ficheiro sozinho é curto
        // demais para o modelo de embeddings distinguir sinal de ruído
        // (medido em testes: mesmo sem nenhum conteúdo, várias categorias
        // e tags batiam limiares calibrados para texto real — falsos
        // positivos). Mesma honestidade já aplicada à extracção de PDF
        // sem camada de texto: nunca inventar, cair em "Outro" sem tags.
        if (text.empty()) {
            res.set_content("Outro\n\n", "text/plain; charset=utf-8");
            return;
        }

        std::string combined = truncateChars(filename + "\n" + text, kMaxInputChars);

        auto vec = embed(embedModel, "", combined);
        std::string type = classifyType(vec);
        auto tags = classifyTags(vec);

        std::ostringstream out;
        out << type << "\n";
        for (size_t i = 0; i < tags.size(); ++i) {
            if (i) out << ",";
            out << tags[i];
        }
        out << "\n";
        res.set_content(out.str(), "text/plain; charset=utf-8");
    });

    svr.Post("/embed", [&](const httplib::Request &req, httplib::Response &res) {
        std::string text = truncateChars(trim(req.body), kMaxInputChars);
        auto vec = embed(embedModel, "", text);
        res.set_content(encodeVector(vec), "text/plain; charset=utf-8");
    });

    svr.Post("/chat", [&](const httplib::Request &req, httplib::Response &res) {
        auto [question, context] = splitOnSep(req.body, kChatContextSep);

        std::string system =
            "Es um assistente que responde a perguntas em portugues com base apenas nos "
            "excertos de documentos fornecidos como contexto. Se a resposta nao estiver no "
            "contexto, diz que nao sabe — nunca inventes. Responde sempre de forma direta e "
            "curta, no maximo 2-3 frases — nunca repitas a pergunta nem escrevas texto a mais.";
        std::string user = "Pergunta: " + question + "\n\nContexto:\n" +
                            (context.empty() ? std::string("(sem contexto disponivel)")
                                              : truncateChars(context, kMaxInputChars));

        std::string prompt = buildPrompt(chatModel, system, user);
        std::string answer = generate(chatModel, prompt, kMaxNewTokensChat);
        res.set_content(answer, "text/plain; charset=utf-8");
    });

    std::fprintf(stderr, "llm-service a arrancar em %s:%d\n", addr.host.c_str(), addr.port);
    if (!svr.listen(addr.host, addr.port)) {
        std::fprintf(stderr, "llm-service: falha a ligar em %s\n", args.listen.c_str());
        return 1;
    }
    return 0;
}
