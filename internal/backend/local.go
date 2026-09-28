package backend

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"DocumentApp/internal/callbackserver"
	"DocumentApp/internal/db"
	"DocumentApp/internal/documents"
	"DocumentApp/internal/filestore"
	"DocumentApp/internal/folders"
	"DocumentApp/internal/heartbeats"
	"DocumentApp/internal/jobs"
	"DocumentApp/internal/llmservice"
	"DocumentApp/internal/searchservice"
	"DocumentApp/internal/workerpool"
)

// Constantes de fiabilidade do pool e contrapressão — ver ROADMAP.md,
// Fase 4 (movidas de app.go para aqui na Fase 6, sem mudar de valor).
const (
	leaseDuration          = 90 * time.Second
	reapInterval           = 15 * time.Second
	heartbeatDeadThreshold = 20 * time.Second
	maxPendingJobs         = 20

	// maxAssistantSources limita quantos documentos recuperados pelo
	// search-service entram no contexto de AskAssistant — mais que isto
	// só dilui o excerto de cada um sem ganhar precisão, e aproxima-se
	// mais depressa do limite de contexto do modelo de chat.
	maxAssistantSources = 5

	// maxCorpusOverviewDocs limita quantos documentos entram na linha de
	// visão geral da colecção (ver corpusOverview) — mesma razão de
	// maxAssistantExcerptChars, não repetir o problema de latência com
	// uma lista enorme.
	maxCorpusOverviewDocs = 50

	// maxAssistantExcerptChars limita quanto de CADA documento entra no
	// contexto, antes de juntar os até maxAssistantSources excertos.
	// Sem isto, um único documento real longo (ex. um PDF com ~4000
	// caracteres de texto extraído, ver Marco 2 do plano de IA local)
	// pode dominar o pedido a ponto de a geração no llm-service demorar
	// vários minutos em vez de segundos — medido em primeira mão: uma
	// pergunta contra a biblioteca real do utilizador (2 pastas, 6
	// documentos) excedeu um timeout de 120s antes desta correção.
	maxAssistantExcerptChars = 800
)

// Config junta tudo o que é preciso para arrancar um LocalBackend — quem
// chama (app.go para o build desktop, cmd/server para o headless)
// resolve os caminhos por omissão consoante o seu próprio ambiente; um
// caminho de binário vazio ("") significa "indisponível", degradando
// graciosamente em vez de falhar o arranque (mesmo padrão desde a
// Fase 3).
type Config struct {
	DBPath               string
	FilesRoot            string
	AIWorkerBinPath      string
	SearchServiceBinPath string
	LLMServiceBinPath    string
	AIChatModelPath      string
	AIEmbedModelPath     string
	MaxWorkers           int
	Emit                 Emitter // nil -> NoopEmitter
}

type LocalBackend struct {
	dbConn      *sql.DB
	folders     *folders.Service
	docs        *documents.Service
	jobs        *jobs.Service
	heartbeats  *heartbeats.Tracker
	files       *filestore.Store
	callbackSrv *callbackserver.Server
	pool        *workerpool.Pool       // nil se o binário do ai-worker não foi encontrado
	search      *searchservice.Manager // nil se o binário do search-service não foi encontrado ou não arrancou
	reapStop    chan struct{}

	// llm e llmMu: o llm-service arranca em segundo plano (ver NewLocal)
	// porque carregar os modelos GGUF pode demorar até ~150s em CPU —
	// bloquear NewLocal (e portanto o arranque de toda a app, incluindo
	// os primeiros ListFolders/ListDocuments do frontend) por esse tempo
	// causava um nil pointer dereference real (a.backend continuava nil
	// no app.go enquanto NewLocal não retornava), visto em primeira mão:
	// a lista de pastas ficava presa em "a carregar" até qualquer outra
	// chamada (ex. CreateFolder) por acaso acontecer depois de
	// NewLocal já ter terminado. Enquanto llm continuar nil, Summarize/
	// Classify/AskAssistant degradam como "ainda não disponível", igual
	// ao que já acontece quando o binário/modelo falta de todo.
	llmMu sync.RWMutex
	llm   *llmservice.Manager
	emit        Emitter

	// ingestMu serializa o "contar pendentes, depois decidir" em
	// IngestFile — sem isto, pedidos verdadeiramente concorrentes podem
	// todos ver a contagem antiga e passar todos, ultrapassando
	// maxPendingJobs (visto em primeira mão com 25 chamadas em
	// simultâneo, ver ROADMAP.md Fase 4). O SQLite já serializa as
	// escritas de qualquer forma (MaxOpenConns=1 em internal/db), por
	// isso isto não acrescenta contenção nova.
	ingestMu sync.Mutex
}

// NewLocal abre a base de dados, arranca o callback server interno, o
// reaper, e — se os binários existirem — o pool de ai-worker e o
// search-service. Tudo o que faltar degrada de forma não fatal: a app
// funciona sempre, só com menos funcionalidade (ver logs).
func NewLocal(cfg Config) (*LocalBackend, error) {
	emit := cfg.Emit
	if emit == nil {
		emit = NoopEmitter{}
	}

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("backend: open db: %w", err)
	}

	b := &LocalBackend{
		dbConn:     conn,
		folders:    folders.New(conn),
		docs:       documents.New(conn),
		jobs:       jobs.New(conn),
		heartbeats: heartbeats.NewTracker(),
		files:      filestore.New(cfg.FilesRoot),
		emit:       emit,
	}

	if cfg.SearchServiceBinPath != "" {
		mgr := searchservice.New(cfg.SearchServiceBinPath)
		if err := mgr.Start(); err != nil {
			log.Printf("search-service falhou a arrancar, pesquisa não vai funcionar: %v", err)
		} else {
			b.search = mgr
			b.reindexSearchService()
		}
	}

	if cfg.LLMServiceBinPath != "" && cfg.AIChatModelPath != "" && cfg.AIEmbedModelPath != "" {
		go func() {
			mgr := llmservice.New(cfg.LLMServiceBinPath, cfg.AIChatModelPath, cfg.AIEmbedModelPath)
			if err := mgr.Start(); err != nil {
				log.Printf("llm-service falhou a arrancar, resumo/classificação/chat reais não vão funcionar: %v", err)
				return
			}
			b.llmMu.Lock()
			b.llm = mgr
			b.llmMu.Unlock()
			log.Printf("llm-service pronto")

			// O search-service já não calcula os seus próprios
			// embeddings (ver internal/searchservice) — precisa do
			// llm-service para isso, que só fica pronto agora, bem
			// depois de search.Start()/reindexSearchService() terem
			// corrido lá em cima (o llm-service pode demorar até ~150s a
			// carregar os modelos GGUF, ver internal/llmservice). Qualquer
			// documento processado nesse intervalo falhou a indexar
			// silenciosamente (log só) — reindexar tudo agora corrige
			// isso, em vez de deixar a pesquisa incompleta até o próximo
			// reinício da app.
			if b.search != nil {
				b.search.SetEmbedder(mgr)
				b.reindexSearchService()
			}
		}()
	}

	var indexer callbackserver.Indexer
	if b.search != nil {
		indexer = b.search
	}

	// b (não b.llm) é passado como Enricher — b.llm ainda pode estar nil
	// neste momento (o goroutine acima pode não ter terminado), mas os
	// métodos Summarize/Classify de LocalBackend leem b.llm com o mutex
	// sempre que forem chamados, por isso o callback server passa a ver
	// o llm-service assim que ele ficar pronto, sem precisar de saber
	// disso à partida.
	callbackSrv, err := callbackserver.Start(b.docs, emit, b.heartbeats, indexer, b)
	if err != nil {
		conn.Close()
		if b.search != nil {
			b.search.Stop()
		}
		return nil, fmt.Errorf("backend: start callback server: %w", err)
	}
	b.callbackSrv = callbackSrv

	b.reapStop = make(chan struct{})
	go b.reapLoop()

	if cfg.AIWorkerBinPath == "" {
		return b, nil
	}
	maxWorkers := cfg.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	b.pool = workerpool.New(cfg.AIWorkerBinPath, cfg.DBPath, callbackSrv.CallbackURL(), callbackSrv.HeartbeatURL(), maxWorkers)
	b.pool.Start()

	return b, nil
}

func (b *LocalBackend) Close() error {
	if b.reapStop != nil {
		close(b.reapStop)
	}
	if b.callbackSrv != nil {
		b.callbackSrv.Close()
	}
	if b.search != nil {
		b.search.Stop()
	}
	b.llmMu.RLock()
	llm := b.llm
	b.llmMu.RUnlock()
	if llm != nil {
		llm.Stop()
	}
	if b.dbConn != nil {
		return b.dbConn.Close()
	}
	return nil
}

// reindexSearchService povoa o índice em memória do search-service a
// partir do SQLite — necessário porque esse índice não sobrevive a um
// reinício do processo (ver internal/searchservice), e o SQLite continua
// a ser a única fonte de verdade. Falhas são só logadas.
func (b *LocalBackend) reindexSearchService() {
	docs, err := b.docs.ListIndexable()
	if err != nil {
		log.Printf("search-service: falha a listar documentos indexáveis: %v", err)
		return
	}
	for _, d := range docs {
		if err := b.search.IndexUpsert(d.ID, documents.BuildIndexText(d)); err != nil {
			log.Printf("search-service: falha a reindexar documento %s: %v", d.ID, err)
		}
	}
}

// reapLoop corre em segundo plano durante toda a vida do backend: liberta
// leases expiradas e jobs de workers sem heartbeat, de volta para a
// fila, e avisa quem estiver a ouvir. Continua a funcionar mesmo que
// todos os ai-worker tenham morrido — é este processo, não os workers,
// quem garante que nada fica preso para sempre.
func (b *LocalBackend) reapLoop() {
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()
	for {
		select {
		case <-b.reapStop:
			return
		case <-ticker.C:
			b.reapOnce()
		}
	}
}

func (b *LocalBackend) reapOnce() {
	var releasedDocs []string

	if ids, err := b.jobs.ReleaseStale(leaseDuration); err != nil {
		log.Printf("reap: ReleaseStale: %v", err)
	} else if len(ids) > 0 {
		log.Printf("reap: %d job(s) com lease expirada, de volta à fila", len(ids))
		releasedDocs = append(releasedDocs, ids...)
	}

	for _, workerID := range b.heartbeats.Dead(heartbeatDeadThreshold) {
		ids, err := b.jobs.ReleaseByWorker(workerID)
		if err != nil {
			log.Printf("reap: ReleaseByWorker(%s): %v", workerID, err)
			continue
		}
		if len(ids) > 0 {
			log.Printf("reap: worker %s sem heartbeat, %d job(s) libertado(s)", workerID, len(ids))
		}
		releasedDocs = append(releasedDocs, ids...)
	}

	if len(releasedDocs) == 0 {
		return
	}

	for _, docID := range releasedDocs {
		if doc, err := b.docs.Get(docID); err == nil {
			b.emit.Emit("document:updated", doc)
		}
	}

	if b.pool != nil {
		if n, err := b.jobs.CountQueued(); err == nil {
			b.pool.EnsureCapacity(n)
		}
	}
}

func (b *LocalBackend) ListFolders() ([]folders.Folder, error) {
	return b.folders.List()
}

func (b *LocalBackend) CreateFolder(name, accentGradient string, isShared bool) (folders.Folder, error) {
	return b.folders.Create(name, accentGradient, isShared)
}

func (b *LocalBackend) ListDocuments(folderID string) ([]documents.Document, error) {
	return b.docs.ListByFolder(folderID)
}

// DeleteFolder apaga a pasta e todos os documentos dentro dela (jobs,
// linha, ficheiros geridos) — não há confirmação nem "soft delete" a
// este nível; a UI é que pede confirmação antes de chamar isto.
func (b *LocalBackend) DeleteFolder(id string) error {
	docs, err := b.docs.ListByFolder(id)
	if err != nil {
		return err
	}
	for _, d := range docs {
		if err := b.DeleteDocument(d.ID); err != nil {
			return fmt.Errorf("apagar documento %q da pasta: %w", d.ID, err)
		}
	}
	return b.folders.Delete(id)
}

func (b *LocalBackend) ApproveDocument(id string) error {
	return b.docs.Approve(id)
}

func (b *LocalBackend) RejectDocument(id string) error {
	return b.docs.Reject(id)
}

// UpdateDocumentClassification aplica uma correção manual do utilizador
// ao tipo de documento e às tags sugeridas pelo modelo (ver
// documents.UpdateManualClassification) — reindexa (o tipo de documento
// entra em BuildIndexText) e avisa o frontend, para uma janela de revisão
// aberta noutra vista ver a correção sem recarregar.
func (b *LocalBackend) UpdateDocumentClassification(id, documentType string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("codificar tags: %w", err)
	}
	if err := b.docs.UpdateManualClassification(id, documentType, string(tagsJSON)); err != nil {
		return err
	}

	doc, err := b.docs.Get(id)
	if err != nil {
		return err
	}
	if b.search != nil {
		if err := b.search.IndexUpsert(doc.ID, documents.BuildIndexText(doc)); err != nil {
			log.Printf("backend: reindexar %s depois da correção manual: %v", doc.ID, err)
		}
	}
	b.emit.Emit("document:updated", doc)
	return nil
}

// DeleteDocument apaga definitivamente um documento: primeiro os
// ai_jobs (senão a foreign key documents<-ai_jobs impede o DELETE),
// depois a linha do documento, e só depois — best-effort, uma falha
// aqui não desfaz o que já foi apagado da base de dados — os ficheiros
// geridos (internal/filestore). Não remove a entrada correspondente do
// índice em memória do search-service: LocalBackend.Search já ignora
// resultados cujo documento já não existe (ver o "continue" nesse
// método), por isso uma entrada parada lá não tem efeito visível.
func (b *LocalBackend) DeleteDocument(id string) error {
	if err := b.jobs.DeleteByDocument(id); err != nil {
		return err
	}
	if err := b.docs.Delete(id); err != nil {
		return err
	}
	if err := b.files.Delete(id); err != nil {
		log.Printf("DeleteDocument(%s): falha a apagar ficheiros geridos: %v", id, err)
	}
	return nil
}

// IngestFile grava o conteúdo de r na pasta gerida (internal/filestore)
// e só depois entra na fila de IA — nunca confia num caminho que já
// exista no disco deste processo, porque em modo host o pedido pode vir
// de um cliente remoto cujo disco é outro (ver ROADMAP.md, Fase 6).
// Recusa novas entradas se a fila já tiver demasiado trabalho pendente
// (contrapressão, ver ROADMAP.md Fase 4).
func (b *LocalBackend) IngestFile(folderID, fileName string, size int64, r io.Reader) (documents.Document, error) {
	b.ingestMu.Lock()
	defer b.ingestMu.Unlock()

	if n, err := b.jobs.CountPending(); err == nil && n >= maxPendingJobs {
		return documents.Document{}, fmt.Errorf(
			"fila de IA cheia (%d jobs pendentes) — espera que alguns terminem antes de adicionar mais", n,
		)
	}

	id := uuid.NewString()
	path, written, err := b.files.Save(id, fileName, r)
	if err != nil {
		return documents.Document{}, fmt.Errorf("guardar ficheiro: %w", err)
	}

	doc, err := b.jobs.Enqueue(id, folderID, fileName, path, written)
	if err != nil {
		return documents.Document{}, err
	}
	if b.pool != nil {
		if n, err := b.jobs.CountQueued(); err == nil {
			b.pool.EnsureCapacity(n)
		}
	}
	return doc, nil
}

func (b *LocalBackend) DocumentFile(id string) (io.ReadCloser, string, error) {
	doc, err := b.docs.Get(id)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(doc.LocalPath)
	if err != nil {
		return nil, "", fmt.Errorf("abrir ficheiro do documento %q: %w", id, err)
	}
	return f, doc.FileName, nil
}

// Search pesquisa nos documentos já indexados (pending_review ou ready)
// por texto/pergunta. Chamada síncrona: bloqueia até o search-service
// responder, propositadamente (ver ROADMAP.md — "síncrono, nunca passa
// pela fila").
func (b *LocalBackend) Search(query string) (SearchResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResponse{Type: "none"}, nil
	}
	if b.search == nil {
		return SearchResponse{}, fmt.Errorf("serviço de pesquisa indisponível (search-service não arrancou)")
	}

	result, err := b.search.Search(query)
	if err != nil {
		return SearchResponse{}, err
	}

	hits := make([]SearchHit, 0, len(result.Hits))
	for _, h := range result.Hits {
		doc, err := b.docs.Get(h.DocumentID)
		if err != nil {
			continue
		}
		if doc.Status != documents.StatusPendingReview && doc.Status != documents.StatusReady {
			continue // rejeitado/falhado depois de indexado — não mostrar
		}
		hits = append(hits, SearchHit{
			DocumentID:   doc.ID,
			FolderID:     doc.FolderID,
			FileName:     doc.FileName,
			DocumentType: doc.DocumentType,
			Summary:      doc.Summary,
			Score:        h.Score,
		})
	}

	respType := result.Type
	if len(hits) == 0 {
		respType = "none"
	}
	return SearchResponse{Type: respType, Results: hits}, nil
}

// AskAssistant faz RAG simples: reaproveita o mesmo search-service de
// Search para encontrar os documentos mais relevantes, monta o contexto a
// partir do excerto (ou, na falta dele, do resumo) de cada um, e pede ao
// llm-service uma resposta gerada. Chamada síncrona, tal como Search —
// bloqueia até o modelo terminar de gerar.
func (b *LocalBackend) AskAssistant(question string) (AssistantAnswer, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return AssistantAnswer{}, fmt.Errorf("pergunta vazia")
	}
	if b.search == nil {
		return AssistantAnswer{}, fmt.Errorf("assistente indisponível (search-service não arrancou)")
	}
	b.llmMu.RLock()
	llm := b.llm
	b.llmMu.RUnlock()
	if llm == nil {
		return AssistantAnswer{}, fmt.Errorf("assistente indisponível (llm-service ainda não arrancou ou não está instalado)")
	}

	result, err := b.search.Search(question)
	if err != nil {
		return AssistantAnswer{}, err
	}

	sources := make([]SearchHit, 0, maxAssistantSources)
	contextParts := make([]string, 0, maxAssistantSources+1)
	if overview := b.corpusOverview(); overview != "" {
		contextParts = append(contextParts, overview)
	}
	for _, h := range result.Hits {
		if len(sources) >= maxAssistantSources {
			break
		}
		doc, err := b.docs.Get(h.DocumentID)
		if err != nil {
			continue
		}
		if doc.Status != documents.StatusPendingReview && doc.Status != documents.StatusReady {
			continue // rejeitado/falhado depois de indexado — não mostrar nem usar como fonte
		}

		excerpt := doc.OCRExcerpt
		if excerpt == "" {
			excerpt = doc.Summary
		}
		if len(excerpt) > maxAssistantExcerptChars {
			excerpt = excerpt[:maxAssistantExcerptChars] + " […]"
		}
		contextParts = append(contextParts, fmt.Sprintf("[%s] %s", doc.FileName, excerpt))
		sources = append(sources, SearchHit{
			DocumentID:   doc.ID,
			FolderID:     doc.FolderID,
			FileName:     doc.FileName,
			DocumentType: doc.DocumentType,
			Summary:      doc.Summary,
			Score:        h.Score,
		})
	}

	answer, err := llm.Chat(question, strings.Join(contextParts, "\n\n"))
	if err != nil {
		return AssistantAnswer{}, err
	}
	return AssistantAnswer{Answer: answer, Sources: sources}, nil
}

// Summarize e Classify implementam callbackserver.Enricher — LocalBackend
// em si (não llmservice.Manager directamente) é o que é passado a
// callbackserver.Start, precisamente para poder continuar nil por baixo
// enquanto o llm-service ainda está a carregar os modelos em segundo
// plano (ver NewLocal) sem o callback server precisar de saber disso.
func (b *LocalBackend) Summarize(fileName, text string) (string, error) {
	b.llmMu.RLock()
	llm := b.llm
	b.llmMu.RUnlock()
	if llm == nil {
		return "", fmt.Errorf("llm-service ainda não arrancou ou não está instalado")
	}
	return llm.Summarize(fileName, text)
}

func (b *LocalBackend) Classify(fileName, text string) (documentType string, tags []string, err error) {
	b.llmMu.RLock()
	llm := b.llm
	b.llmMu.RUnlock()
	if llm == nil {
		return "", nil, fmt.Errorf("llm-service ainda não arrancou ou não está instalado")
	}
	return llm.Classify(fileName, text)
}

// corpusOverview monta uma linha com a contagem e a lista (nome + tipo)
// dos documentos indexados. Sem isto, perguntas sobre a COLEÇÃO em si
// ("quantos documentos tens", "o que está na base de dados") nunca
// encontravam nenhum "match" de conteúdo na pesquisa semântica — não há
// nenhum documento cujo TEXTO fale sobre quantos documentos existem — por
// isso o contexto ficava sempre vazio e a resposta era sempre "não sei",
// visto em primeira mão a testar com o utilizador real. Devolve "" (sem
// entrar no contexto) se não houver documentos ou a listagem falhar —
// nunca bloqueia a pergunta por causa disto.
func (b *LocalBackend) corpusOverview() string {
	docs, err := b.docs.ListIndexable()
	if err != nil || len(docs) == 0 {
		return ""
	}

	total := len(docs)
	listed := docs
	truncated := false
	if len(listed) > maxCorpusOverviewDocs {
		listed = listed[:maxCorpusOverviewDocs]
		truncated = true
	}

	items := make([]string, 0, len(listed))
	for _, d := range listed {
		docType := d.DocumentType
		if docType == "" {
			docType = "tipo desconhecido"
		}
		items = append(items, fmt.Sprintf("%s (%s)", d.FileName, docType))
	}

	overview := fmt.Sprintf("A base de dados tem %d documento(s) indexado(s): %s.", total, strings.Join(items, ", "))
	if truncated {
		overview += fmt.Sprintf(" (lista limitada aos primeiros %d)", maxCorpusOverviewDocs)
	}
	return overview
}
