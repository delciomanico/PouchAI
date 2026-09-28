package callbackserver

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"DocumentApp/internal/db"
	"DocumentApp/internal/documents"
	"DocumentApp/internal/folders"
	"DocumentApp/internal/heartbeats"
)

type fakeEmitter struct {
	mu     sync.Mutex
	events []string
	docs   []documents.Document
}

func (f *fakeEmitter) Emit(eventName string, data ...interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, eventName)
	if len(data) == 1 {
		if doc, ok := data[0].(documents.Document); ok {
			f.docs = append(f.docs, doc)
		}
	}
}

func (f *fakeEmitter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

type fakeIndexer struct {
	mu    sync.Mutex
	calls []string // "documentID:text"
	err   error
}

func (f *fakeIndexer) IndexUpsert(documentID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, documentID+":"+text)
	return f.err
}

func (f *fakeIndexer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type fakeEnricher struct {
	mu           sync.Mutex
	calls        int
	documentType string
	tags         []string
	summary      string
	err          error
}

func (f *fakeEnricher) Summarize(fileName, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.summary, nil
}

func (f *fakeEnricher) Classify(fileName, text string) (string, []string, error) {
	if f.err != nil {
		return "", nil, f.err
	}
	return f.documentType, f.tags, nil
}

func newTestServer(t *testing.T) (*Server, *fakeEmitter, *heartbeats.Tracker, *sql.DB) {
	t.Helper()
	srv, emitter, hb, conn, _ := newTestServerWithDeps(t, nil, nil)
	return srv, emitter, hb, conn
}

func newTestServerWithIndexer(t *testing.T, indexer Indexer) (*Server, *fakeEmitter, *heartbeats.Tracker, *sql.DB, *documents.Service) {
	t.Helper()
	return newTestServerWithDeps(t, indexer, nil)
}

func newTestServerWithDeps(t *testing.T, indexer Indexer, enricher Enricher) (*Server, *fakeEmitter, *heartbeats.Tracker, *sql.DB, *documents.Service) {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	docsSvc := documents.New(conn)
	emitter := &fakeEmitter{}
	hb := heartbeats.NewTracker()
	srv, err := Start(docsSvc, emitter, hb, indexer, enricher)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { srv.Close() })

	return srv, emitter, hb, conn, docsSvc
}

func TestCallbackEmitsUpdatedDocument(t *testing.T) {
	srv, emitter, _, conn := newTestServer(t)

	folderSvc := folders.New(conn)
	folder, err := folderSvc.Create("Contratos", "grad", false)
	if err != nil {
		t.Fatalf("folders.Create: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	docID := "doc-1"
	_, err = conn.Exec(
		`INSERT INTO documents (id, file_name, folder_id, size_bytes, local_path, status, tags_json, created_at, updated_at)
		 VALUES (?, 'a.pdf', ?, 1, '/tmp/a.pdf', 'pending_review', '[]', ?, ?)`,
		docID, folder.ID, now, now,
	)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}

	body, _ := json.Marshal(callbackPayload{DocumentID: docID})
	resp, err := http.Post(srv.CallbackURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if emitter.count() != 1 {
		t.Fatalf("emitted %d events, want 1", emitter.count())
	}
	if emitter.docs[0].ID != docID {
		t.Fatalf("emitted document ID = %q, want %q", emitter.docs[0].ID, docID)
	}
	if emitter.docs[0].Status != "pending_review" {
		t.Fatalf("emitted document Status = %q, want %q", emitter.docs[0].Status, "pending_review")
	}
}

func TestCallbackIndexesWhenExcerptPresent(t *testing.T) {
	indexer := &fakeIndexer{}
	srv, _, _, conn, docsSvc := newTestServerWithIndexer(t, indexer)

	folderSvc := folders.New(conn)
	folder, err := folderSvc.Create("Contratos", "grad", false)
	if err != nil {
		t.Fatalf("folders.Create: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	docID := "doc-indexed"
	_, err = conn.Exec(
		`INSERT INTO documents (id, file_name, folder_id, document_type, size_bytes, local_path, status, ocr_excerpt, tags_json, created_at, updated_at)
		 VALUES (?, 'contrato.pdf', ?, 'PDF', 1, '/tmp/contrato.pdf', 'pending_review', 'texto extraido de contrato.pdf', '[]', ?, ?)`,
		docID, folder.ID, now, now,
	)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}

	body, _ := json.Marshal(callbackPayload{DocumentID: docID})
	resp, err := http.Post(srv.CallbackURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if indexer.count() != 1 {
		t.Fatalf("indexer called %d times, want 1", indexer.count())
	}

	doc, err := docsSvc.Get(docID)
	if err != nil {
		t.Fatalf("docs.Get: %v", err)
	}
	if doc.Summary == "" {
		t.Fatalf("summary was not persisted")
	}
	if doc.Summary != "PDF — texto extraido de contrato.pdf" {
		t.Fatalf("summary = %q, want %q", doc.Summary, "PDF — texto extraido de contrato.pdf")
	}
}

// TestCallbackEnrichesWhenPresent confirma que, quando há um Enricher
// (llm-service disponível), o resumo/document_type/tags gravados vêm dele
// em vez do fallback BuildSummary — sem ele (ver
// TestCallbackIndexesWhenExcerptPresent, que passa enricher nil), o
// comportamento fica exatamente o mesmo de antes desta funcionalidade.
func TestCallbackEnrichesWhenPresent(t *testing.T) {
	indexer := &fakeIndexer{}
	enricher := &fakeEnricher{
		documentType: "Contrato",
		tags:         []string{"juridico", "trabalho"},
		summary:      "Resumo gerado pelo modelo local.",
	}
	srv, _, _, conn, docsSvc := newTestServerWithDeps(t, indexer, enricher)

	folderSvc := folders.New(conn)
	folder, err := folderSvc.Create("Contratos", "grad", false)
	if err != nil {
		t.Fatalf("folders.Create: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	docID := "doc-enriched"
	_, err = conn.Exec(
		`INSERT INTO documents (id, file_name, folder_id, document_type, size_bytes, local_path, status, ocr_excerpt, tags_json, created_at, updated_at)
		 VALUES (?, 'contrato.pdf', ?, 'PDF', 1, '/tmp/contrato.pdf', 'pending_review', 'texto extraido de contrato.pdf', '[]', ?, ?)`,
		docID, folder.ID, now, now,
	)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}

	body, _ := json.Marshal(callbackPayload{DocumentID: docID})
	resp, err := http.Post(srv.CallbackURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// enrichAsync corre na sua própria goroutine (ver handleCallback) —
	// esperar até ao segundo evento "document:updated" ou expirar, em vez
	// de verificar logo a seguir ao POST, que corria com a goroutine
	// ainda por terminar.
	doc, err := docsSvc.Get(docID)
	if err != nil {
		t.Fatalf("docs.Get: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for doc.DocumentType != "Contrato" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		doc, err = docsSvc.Get(docID)
		if err != nil {
			t.Fatalf("docs.Get: %v", err)
		}
	}
	if doc.Summary != "Resumo gerado pelo modelo local." {
		t.Fatalf("summary = %q, want o resumo do enricher", doc.Summary)
	}
	if doc.DocumentType != "Contrato" {
		t.Fatalf("document_type = %q, want %q", doc.DocumentType, "Contrato")
	}
	if doc.TagsJSON != `["juridico","trabalho"]` {
		t.Fatalf("tags_json = %q, want %q", doc.TagsJSON, `["juridico","trabalho"]`)
	}
	if doc.ClassificationPending {
		t.Fatalf("classification_pending = true depois do enriquecimento terminar, want false")
	}
}

func TestCallbackDoesNotIndexBeforeExcerpt(t *testing.T) {
	indexer := &fakeIndexer{}
	srv, _, _, conn, _ := newTestServerWithIndexer(t, indexer)

	folderSvc := folders.New(conn)
	folder, err := folderSvc.Create("Contratos", "grad", false)
	if err != nil {
		t.Fatalf("folders.Create: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	docID := "doc-processing"
	_, err = conn.Exec(
		`INSERT INTO documents (id, file_name, folder_id, size_bytes, local_path, status, tags_json, created_at, updated_at)
		 VALUES (?, 'a.pdf', ?, 1, '/tmp/a.pdf', 'processing', '[]', ?, ?)`,
		docID, folder.ID, now, now,
	)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}

	body, _ := json.Marshal(callbackPayload{DocumentID: docID})
	resp, err := http.Post(srv.CallbackURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST callback: %v", err)
	}
	defer resp.Body.Close()

	if indexer.count() != 0 {
		t.Fatalf("indexer called %d times, want 0 (document still processing, no excerpt yet)", indexer.count())
	}
}

func TestCallbackUnknownDocument(t *testing.T) {
	srv, emitter, _, _ := newTestServer(t)

	body, _ := json.Marshal(callbackPayload{DocumentID: "does-not-exist"})
	resp, err := http.Post(srv.CallbackURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if emitter.count() != 0 {
		t.Fatalf("emitted %d events, want 0", emitter.count())
	}
}

func TestCallbackInvalidPayload(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	resp, err := http.Post(srv.CallbackURL(), "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("POST callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHeartbeatTouchesTracker(t *testing.T) {
	srv, _, hb, _ := newTestServer(t)

	body, _ := json.Marshal(heartbeatPayload{WorkerID: "worker-1"})
	resp, err := http.Post(srv.HeartbeatURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST heartbeat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	time.Sleep(20 * time.Millisecond)
	if dead := hb.Dead(10 * time.Millisecond); len(dead) != 1 || dead[0] != "worker-1" {
		t.Fatalf("tracker did not register the heartbeat: Dead(10ms) after a 20ms sleep = %v", dead)
	}
}

func TestHeartbeatInvalidPayload(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	resp, err := http.Post(srv.HeartbeatURL(), "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("POST heartbeat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
