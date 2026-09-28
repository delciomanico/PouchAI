// Package callbackserver expõe os dois endpoints HTTP internos que o
// ai-worker chama:
//   - /internal/jobs/callback ao reclamar ou terminar um job — nunca
//     escreve no SQLite (o C++ já o fez), só relê o documento e emite
//     um evento para o React actualizar a UI sem recarregar.
//   - /internal/workers/heartbeat periodicamente enquanto vivo — só
//     regista o instante em heartbeats.Tracker, para app.go detectar
//     workers mortos mais depressa que a lease de 90s (ver Fase 4).
//
// Ver ROADMAP.md, Fases 3 e 4.
package callbackserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"DocumentApp/internal/documents"
	"DocumentApp/internal/heartbeats"
)

// Emitter abstrai o wailsruntime.EventsEmit, para este package ser
// testável sem um contexto Wails real.
type Emitter interface {
	Emit(eventName string, data ...interface{})
}

// Indexer abstrai o searchservice.Manager (Fase 5) — pode ser nil
// (interface, não valor concreto) se o search-service não estiver
// disponível, nesse caso o callback simplesmente não indexa nada.
type Indexer interface {
	IndexUpsert(documentID, text string) error
}

// Enricher abstrai o llmservice.Manager (motor de IA local real — ver
// plano em unified-popping-sunset.md): pode ser nil (interface, não valor
// concreto) se o llm-service não estiver disponível, nesse caso o callback
// mantém exatamente o comportamento anterior (BuildSummary/BuildIndexText,
// sem document_type/tags reais). Duas chamadas separadas — um endpoint
// gerativo (Summarize), um de embeddings (Classify) — em vez de um único
// pedido que devolvesse tudo num formato que o Go teria de parsear, ver
// "Dois modelos, dois papéis diferentes" no plano.
type Enricher interface {
	Summarize(fileName, text string) (string, error)
	Classify(fileName, text string) (documentType string, tags []string, err error)
}

type Server struct {
	docs       *documents.Service
	emit       Emitter
	heartbeats *heartbeats.Tracker
	indexer    Indexer
	enricher   Enricher
	listener   net.Listener
	httpSrv    *http.Server
}

// Start liga a um porto livre em 127.0.0.1 (nunca exposto à rede) e
// começa a servir em segundo plano. indexer e enricher podem ser nil.
func Start(docs *documents.Service, emit Emitter, hb *heartbeats.Tracker, indexer Indexer, enricher Enricher) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("callbackserver: listen: %w", err)
	}

	s := &Server{docs: docs, emit: emit, heartbeats: hb, indexer: indexer, enricher: enricher, listener: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/jobs/callback", s.handleCallback)
	mux.HandleFunc("/internal/workers/heartbeat", s.handleHeartbeat)
	s.httpSrv = &http.Server{Handler: mux}

	go s.httpSrv.Serve(ln)
	return s, nil
}

// CallbackURL devolve o endpoint a passar ao ai-worker via --callback.
func (s *Server) CallbackURL() string {
	return fmt.Sprintf("http://%s/internal/jobs/callback", s.listener.Addr().String())
}

// HeartbeatURL devolve o endpoint a passar ao ai-worker via
// --heartbeat-url.
func (s *Server) HeartbeatURL() string {
	return fmt.Sprintf("http://%s/internal/workers/heartbeat", s.listener.Addr().String())
}

func (s *Server) Close() error {
	return s.httpSrv.Close()
}

type callbackPayload struct {
	DocumentID string `json:"document_id"`
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload callbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.DocumentID == "" {
		http.Error(w, "invalid payload: expected {\"document_id\": \"...\"}", http.StatusBadRequest)
		return
	}

	doc, err := s.docs.Get(payload.DocumentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// O OCR/classificação acabou de terminar (esta é a segunda das duas
	// chamadas de callback do ai-worker por job — a primeira, ao
	// reclamar, ainda não tem ocr_excerpt). É aqui que se deriva o
	// resumo e se manda indexar para pesquisa (Fase 5) — nunca no
	// primeiro callback, porque ainda não há texto nenhum para indexar.
	if s.indexer != nil && doc.OCRExcerpt != "" &&
		(doc.Status == documents.StatusPendingReview || doc.Status == documents.StatusReady) {
		summary := documents.BuildSummary(doc)
		if err := s.docs.UpdateSummary(doc.ID, summary); err != nil {
			log.Printf("callbackserver: UpdateSummary(%s): %v", doc.ID, err)
		} else {
			doc.Summary = summary
		}
		if err := s.indexer.IndexUpsert(doc.ID, documents.BuildIndexText(doc)); err != nil {
			log.Printf("callbackserver: IndexUpsert(%s): %v", doc.ID, err)
		}

		// enrich chama o llm-service (gerar resumo + classificar), que em
		// CPU sem GPU pode demorar dezenas de segundos a alguns minutos
		// num documento real (medido em primeira mão: um /summarize
		// chegou a exceder 180s) — corre em segundo plano para não
		// bloquear esta resposta HTTP ao ai-worker nem o primeiro evento
		// "document:updated" (que já sai com o resumo/tipo derivados por
		// extensão, o mesmo fallback de sempre). Quando o enriquecimento
		// real terminar, um SEGUNDO "document:updated" avisa o frontend
		// outra vez, com os dados verdadeiros — o mesmo padrão já usado
		// para o arranque assíncrono do llm-service em internal/backend.
		if s.enricher != nil {
			if err := s.docs.MarkEnrichmentPending(doc.ID); err != nil {
				log.Printf("callbackserver: MarkEnrichmentPending(%s): %v", doc.ID, err)
			} else {
				doc.ClassificationPending = true
			}
			docCopy := doc
			go s.enrichAsync(docCopy)
		}
	}

	s.emit.Emit("document:updated", doc)
	w.WriteHeader(http.StatusNoContent)
}

// enrichAsync chama o llm-service para gerar um resumo real e classificar
// o documento (tipo + tags), substituindo o resumo derivado por
// BuildSummary só instantes antes. Corre na sua própria goroutine (ver
// handleCallback) — falhas são só logadas, o resumo derivado e o
// document_type já calculados a partir da extensão ficam como fallback,
// o documento nunca fica sem nenhum dos dois. Emite um SEGUNDO
// "document:updated" quando terminar, para o frontend actualizar a UI
// mesmo que isso só aconteça muito depois da resposta HTTP original ao
// ai-worker.
func (s *Server) enrichAsync(doc documents.Document) {
	summary, err := s.enricher.Summarize(doc.FileName, doc.OCRExcerpt)
	if err != nil {
		log.Printf("callbackserver: Summarize(%s): %v", doc.ID, err)
		summary = doc.Summary
	}

	documentType, tags, err := s.enricher.Classify(doc.FileName, doc.OCRExcerpt)
	if err != nil {
		log.Printf("callbackserver: Classify(%s): %v", doc.ID, err)
		documentType = doc.DocumentType
		tags = nil
	}

	tagsJSON := doc.TagsJSON
	if tags != nil {
		encoded, err := json.Marshal(tags)
		if err != nil {
			log.Printf("callbackserver: marshal tags(%s): %v", doc.ID, err)
		} else {
			tagsJSON = string(encoded)
		}
	}

	if err := s.docs.UpdateClassification(doc.ID, documentType, summary, tagsJSON); err != nil {
		log.Printf("callbackserver: UpdateClassification(%s): %v", doc.ID, err)
		return
	}
	doc.DocumentType = documentType
	doc.Summary = summary
	doc.TagsJSON = tagsJSON
	doc.ClassificationPending = false
	s.emit.Emit("document:updated", doc)
}

type heartbeatPayload struct {
	WorkerID string `json:"worker_id"`
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload heartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.WorkerID == "" {
		http.Error(w, "invalid payload: expected {\"worker_id\": \"...\"}", http.StatusBadRequest)
		return
	}

	s.heartbeats.Touch(payload.WorkerID)
	w.WriteHeader(http.StatusNoContent)
}
