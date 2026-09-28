// Package searchservice gere o processo C++ search-service (Fase 5):
// arranca-o uma vez (processo único, sempre ativo, ao contrário do pool
// elástico do ai-worker), e fala com ele por HTTP loopback, síncrono.
//
// RECALIBRADO em 2026-09-28: o search-service já não calcula os seus
// próprios embeddings (era feature hashing sobre bag-of-words — via
// nenhuma noção de significado, só sobreposição exacta de palavras).
// Passa a guardar e comparar vectores já calculados pelo llm-service
// (mesmo modelo usado para classificar documentos, paraphrase-
// multilingual-mpnet-base-v2) — ver Embedder abaixo. O ganho de qualidade
// só se torna real em conjunto com a correcção em
// internal/documents.BuildIndexText, que até agora ignorava o texto
// extraído dos documentos ao indexar.
//
// Protocolo interno em texto simples, não JSON — só o Go e o
// search-service falam este protocolo, os dois escritos por nós, por
// isso um parser JSON só acrescentaria trabalho sem benefício real:
//
//	GET  /internal/health       -> "ok" quando está pronto para servir
//	POST /internal/index/upsert -> cabeçalho X-Document-Id, corpo = vector
//	POST /search                -> corpo = vector da pergunta/busca;
//	     resposta: até 20 linhas "<document_id>\t<score>", por ordem
//	     decrescente. Decidir se isto é resposta única/lista/nada é feito
//	     aqui em Go (ver classifyResultType), não no C++.
//
// Ver ROADMAP.md, Fase 5, e search-service/src/main.cpp para o outro lado.
package searchservice

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Hit é um resultado devolvido pelo índice em memória do search-service —
// só o ID do documento e a pontuação de relevância. O Go é que junta os
// restantes campos (nome, pasta, resumo) a partir do SQLite, que continua
// a ser a única fonte de verdade.
type Hit struct {
	DocumentID string
	Score      float64
}

// Result é a resposta interpretada de uma pesquisa.
type Result struct {
	Type string // "answer" | "list" | "none"
	Hits []Hit
}

// Embedder abstrai o llm-service (internal/llmservice.Manager) — quem
// calcula de facto os vectores semânticos. Definido aqui como interface
// (em vez de importar llmservice directamente) para não criar uma
// dependência entre os dois pacotes: quem os liga é internal/backend, que
// já conhece os dois.
type Embedder interface {
	Embed(text string) ([]float32, error)
}

type Manager struct {
	binPath string
	baseURL string
	cmd     *exec.Cmd
	client  *http.Client

	// embedder pode ficar nil durante algum tempo depois de Start(): o
	// llm-service carrega os modelos GGUF em segundo plano (até ~150s em
	// CPU fria, ver internal/llmservice) e só fica pronto bem depois deste
	// Manager. Enquanto isso, IndexUpsert/Search degradam com um erro
	// claro em vez de mandar texto para um índice que não sabe o que
	// fazer com ele — mesmo padrão de degradação usada em todo o backend
	// quando falta um binário/modelo (ver internal/backend.LocalBackend).
	embedderMu sync.RWMutex
	embedder   Embedder
}

func New(binPath string) *Manager {
	return &Manager{
		binPath: binPath,
		// 3s chegava quando pesquisar era só um produto escalar sobre um
		// índice hash — agora inclui sempre chamar o llm-service para
		// embutir a query primeiro (uma passagem pelo modelo, CPU), por
		// isso o timeout tem de acompanhar esse custo novo.
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// SetEmbedder liga este Manager ao motor de embeddings real — chamado
// assim que o llm-service ficar pronto (ver internal/backend.NewLocal).
// Antes disso, e se o llm-service nunca arrancar, IndexUpsert/Search
// devolvem erro em vez de silenciosamente indexar lixo.
func (m *Manager) SetEmbedder(e Embedder) {
	m.embedderMu.Lock()
	m.embedder = e
	m.embedderMu.Unlock()
}

func (m *Manager) getEmbedder() Embedder {
	m.embedderMu.RLock()
	defer m.embedderMu.RUnlock()
	return m.embedder
}

// Start escolhe um porto livre em 127.0.0.1, lança o search-service já
// apontado para esse porto, e espera até /internal/health responder (ou
// desiste ao fim de alguns segundos). Devolve erro sem deixar processos
// pendurados se o arranque falhar.
func (m *Manager) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("searchservice: escolher porto: %w", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	m.baseURL = "http://" + addr

	cmd := exec.Command(m.binPath, "--listen", addr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("searchservice: arrancar %q: %w", m.binPath, err)
	}
	m.cmd = cmd

	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := m.client.Get(m.baseURL + "/internal/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health devolveu estado %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}

	m.Stop()
	return fmt.Errorf("searchservice: não respondeu em %s a tempo: %w", m.baseURL, lastErr)
}

// Stop termina o processo do search-service, se estiver a correr.
func (m *Manager) Stop() {
	if m.cmd == nil || m.cmd.Process == nil {
		return
	}
	_ = m.cmd.Process.Kill()
	_ = m.cmd.Wait()
}

// encodeVector serializa um vector como floats separados por vírgula —
// mesmo formato que llm-service/src/main.cpp (encodeVector) e
// search-service/src/main.cpp (parseVector) usam.
func encodeVector(v []float32) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = strconv.FormatFloat(float64(x), 'g', -1, 32)
	}
	return strings.Join(parts, ",")
}

// IndexUpsert manda o search-service (re)guardar o vector semântico de um
// documento, substituindo qualquer entrada anterior para o mesmo ID. O
// texto é embutido aqui (via Embedder) antes de seguir para o C++, que já
// não sabe nada sobre texto, só sobre vectores.
func (m *Manager) IndexUpsert(documentID, text string) error {
	embedder := m.getEmbedder()
	if embedder == nil {
		return fmt.Errorf("searchservice: motor de embeddings ainda não está pronto (llm-service a carregar ou indisponível)")
	}
	vec, err := embedder.Embed(text)
	if err != nil {
		return fmt.Errorf("searchservice: embed do texto a indexar: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, m.baseURL+"/internal/index/upsert", strings.NewReader(encodeVector(vec)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Document-Id", documentID)
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("searchservice: index upsert: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("searchservice: index upsert devolveu estado %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// kListFloor: abaixo disto, um "resultado" é ruído, não sinal — a
// anisotropia conhecida de modelos de embeddings de frase (ver
// llm-service/src/main.cpp) faz qualquer par de textos partilhar uma
// pontuação de cosseno positiva mesmo sem relação nenhuma. Sem este
// filtro, uma pesquisa sempre devolveria o corpus inteiro ordenado, só
// que os últimos resultados seriam lixo a fingir de correspondência.
//
// kAnswerAbsoluteFloor/kAnswerMargin: para o modo "resposta única" (uma
// pergunta, não uma pesquisa por palavras-chave), exige-se mais —
// confiança suficiente para mostrar UM documento como "a resposta", não
// só "o melhor de uma lista pouco convincente". Segue o mesmo raciocínio
// de kCategoryAbsoluteFloor/kCategoryMargin da classificação (margem
// entre o 1º e o 2º candidato, não só um limiar absoluto).
//
// CALIBRADO em 2026-09-28 com documentos reais da biblioteca do
// utilizador (uma fatura real com texto extraído, um horário escolar, um
// cartão de visita e um cartão de negócios só com placeholder). Medido em
// primeira mão: uma query sem sentido nenhum ("xyzzy quantum toaster")
// ainda batia 0.16-0.20 de cosseno contra documentos completamente
// não relacionados — esse é o "chão" de ruído deste modelo, não zero.
// Já um match genuinamente forte (uma query "horário escolar" contra o
// próprio documento de horário) ficou isolado a 0.30, sem mais nenhum
// documento a passar de 0.15. kListFloor fica entre as duas bandas.
//
// kAnswerAbsoluteFloor ficou propositadamente mais alto que o que os
// matches genuínos mediram (0.30-0.41): uma pergunta simulada do tipo
// "qual é a fatura?" teria dado 0.407 ao documento ERRADO (o horário
// escolar, não a fatura) com margem 0.08 acima do 2º — o suficiente para
// passar num limiar mais baixo e mostrar uma "resposta" confiante mas
// errada. Preferir nunca mostrar "resposta única" a mostrá-la errada
// (mesma política já usada na classificação, ver llm-service/src/main.cpp)
// significa aceitar que o modo "answer" só dispare raramente por agora.
//
// NOTA: parte do ruído nestas medições vem de um problema real e
// diferente, ainda não corrigido: o texto extraído de PDFs
// (ai-worker/src/main.cpp, extractPdfText) vem com caracteres acentuados
// corrompidos (ex. "Descri��o" em vez de "Descrição") — a extracção lê
// bytes da stream do PDF sem respeitar a codificação/ToUnicode da fonte.
// Isto degrada a qualidade do embedding de qualquer documento com
// acentuação (quase todos, em português) e ajuda a explicar por que a
// fatura real nem sempre ficou em 1º lugar nas pesquisas por "fatura".
// Corrigir a extracção de texto deixaria estes números ainda melhores;
// fica documentado para uma próxima ronda, não bloqueou esta.
const (
	kListFloor           = 0.22
	kAnswerAbsoluteFloor = 0.42
	kAnswerMargin        = 0.06
)

// Search pesquisa nos documentos já indexados por texto/pergunta.
// Chamada síncrona: bloqueia até o search-service responder,
// propositadamente (ver ROADMAP.md — "síncrono, nunca passa pela fila").
func (m *Manager) Search(query string) (Result, error) {
	embedder := m.getEmbedder()
	if embedder == nil {
		return Result{}, fmt.Errorf("searchservice: motor de embeddings ainda não está pronto (llm-service a carregar ou indisponível)")
	}
	vec, err := embedder.Embed(query)
	if err != nil {
		return Result{}, fmt.Errorf("searchservice: embed da pergunta: %w", err)
	}

	resp, err := m.client.Post(m.baseURL+"/search", "text/plain; charset=utf-8", strings.NewReader(encodeVector(vec)))
	if err != nil {
		return Result{}, fmt.Errorf("searchservice: search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("searchservice: search devolveu estado %d: %s", resp.StatusCode, string(body))
	}

	var hits []Hit
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		score, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}
		hits = append(hits, Hit{DocumentID: parts[0], Score: score})
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("searchservice: ler resposta: %w", err)
	}

	return classifyResult(query, hits), nil
}

// classifyResult filtra ruído (kListFloor) e decide "answer" | "list" |
// "none" — esta decisão vivia no search-service (C++) antes deste pacote
// passar a falar de vectores: como o Go já tem o texto original da
// pergunta para looksLikeQuestion, ficou mais simples decidir aqui do que
// mandar essa informação de volta para o C++ por um cabeçalho extra.
func classifyResult(query string, hits []Hit) Result {
	filtered := make([]Hit, 0, len(hits))
	for _, h := range hits {
		if h.Score >= kListFloor {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) == 0 {
		return Result{Type: "none"}
	}

	if looksLikeQuestion(query) {
		best := filtered[0].Score
		second := 0.0
		if len(filtered) > 1 {
			second = filtered[1].Score
		}
		if best >= kAnswerAbsoluteFloor && (best-second) >= kAnswerMargin {
			return Result{Type: "answer", Hits: filtered[:1]}
		}
	}

	return Result{Type: "list", Hits: filtered}
}

// looksLikeQuestion detecta se a query parece uma pergunta em vez de uma
// busca por palavras-chave — mesma heurística que vivia em
// search-service/src/main.cpp antes deste pacote passar a decidir isto.
func looksLikeQuestion(query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return false
	}
	if strings.HasSuffix(q, "?") {
		return true
	}

	lower := strings.ToLower(q)
	starters := []string{
		"quem ", "qual ", "quais ", "quando ", "onde ", "como ", "porque ",
		"porquê ", "o que ", "que ", "quanto ", "quantos ", "quantas ",
	}
	for _, s := range starters {
		if strings.HasPrefix(lower, s) {
			return true
		}
	}
	return false
}
