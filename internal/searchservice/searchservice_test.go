package searchservice

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// stubEmbedder finge o llm-service: devolve um vector fixo por texto (ou
// o mesmo vector para todos, se textToVec for nil), sem correr modelo
// nenhum — testa o protocolo HTTP do Manager (IndexUpsert, Search), não a
// qualidade dos embeddings em si (isso é validado manualmente com a app a
// correr, ver memória motor-ia-local).
type stubEmbedder struct {
	textToVec map[string][]float32
	err       error
}

func (s *stubEmbedder) Embed(text string) ([]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	if v, ok := s.textToVec[text]; ok {
		return v, nil
	}
	return []float32{1, 0, 0}, nil
}

// newTestManager aponta um Manager directamente para um httptest.Server
// que finge ser o search-service — testa o protocolo HTTP (IndexUpsert,
// Search) sem precisar do binário C++ compilado nem de Start() (que
// spawna um processo real; esse caminho é validado manualmente com a app
// a correr, ver memória da Fase 5).
func newTestManager(handler http.HandlerFunc, embedder Embedder) (*Manager, *httptest.Server) {
	srv := httptest.NewServer(handler)
	m := &Manager{
		baseURL: srv.URL,
		client:  &http.Client{Timeout: 2 * time.Second},
	}
	if embedder != nil {
		m.SetEmbedder(embedder)
	}
	return m, srv
}

func TestIndexUpsertSendsDocumentIDHeaderAndEncodedVector(t *testing.T) {
	var gotMethod, gotPath, gotHeader, gotBody string
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("X-Document-Id")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}, &stubEmbedder{textToVec: map[string][]float32{"PDF — contrato de arrendamento": {0.5, -0.25, 0.1}}})
	defer srv.Close()

	if err := m.IndexUpsert("doc-1", "PDF — contrato de arrendamento"); err != nil {
		t.Fatalf("IndexUpsert: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/internal/index/upsert" {
		t.Errorf("path = %q, want /internal/index/upsert", gotPath)
	}
	if gotHeader != "doc-1" {
		t.Errorf("X-Document-Id = %q, want doc-1", gotHeader)
	}
	want := "0.5,-0.25,0.1"
	if gotBody != want {
		t.Errorf("body = %q, want %q (vector codificado, não o texto)", gotBody, want)
	}
}

func TestIndexUpsertWithoutEmbedderFailsClearly(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("não devia chegar a contactar o search-service sem embedder")
	}, nil)
	defer srv.Close()

	if err := m.IndexUpsert("doc-1", "texto"); err == nil {
		t.Fatal("IndexUpsert did not return an error when no embedder is set")
	}
}

func TestIndexUpsertPropagatesEmbedError(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("não devia chegar a contactar o search-service se o embed falhar")
	}, &stubEmbedder{err: fmt.Errorf("llm-service indisponível")})
	defer srv.Close()

	if err := m.IndexUpsert("doc-1", "texto"); err == nil {
		t.Fatal("IndexUpsert did not return an error when Embed fails")
	}
}

func TestIndexUpsertPropagatesServerError(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "falta o cabecalho", http.StatusBadRequest)
	}, &stubEmbedder{})
	defer srv.Close()

	if err := m.IndexUpsert("doc-1", "texto"); err == nil {
		t.Fatal("IndexUpsert did not return an error for a 400 response")
	}
}

func TestSearchParsesAnswerWhenQuestionAndConfident(t *testing.T) {
	var gotBody string
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("doc-42\t0.87\ndoc-7\t0.20\n"))
	}, &stubEmbedder{textToVec: map[string][]float32{"quem assinou o contrato de arrendamento?": {1, 0, 0}}})
	defer srv.Close()

	result, err := m.Search("quem assinou o contrato de arrendamento?")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotBody != "1,0,0" {
		t.Errorf("query enviada ao search-service = %q, want o vector codificado", gotBody)
	}
	if result.Type != "answer" {
		t.Fatalf("Type = %q, want answer", result.Type)
	}
	if len(result.Hits) != 1 || result.Hits[0].DocumentID != "doc-42" {
		t.Fatalf("Hits = %+v, want one hit for doc-42", result.Hits)
	}
	if result.Hits[0].Score != 0.87 {
		t.Fatalf("Score = %v, want 0.87", result.Hits[0].Score)
	}
}

func TestSearchFallsBackToListWhenQuestionButNotConfident(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		// Margem pequena entre 1º e 2º: não deve virar "answer" mesmo
		// sendo claramente uma pergunta.
		w.Write([]byte("doc-1\t0.40\ndoc-2\t0.38\n"))
	}, &stubEmbedder{})
	defer srv.Close()

	result, err := m.Search("qual o valor da fatura?")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Type != "list" {
		t.Fatalf("Type = %q, want list (margem insuficiente para answer)", result.Type)
	}
	if len(result.Hits) != 2 {
		t.Fatalf("Hits = %+v, want 2 entries", result.Hits)
	}
}

func TestSearchParsesListForNonQuestionQuery(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("doc-1\t0.5\ndoc-2\t0.3\n"))
	}, &stubEmbedder{})
	defer srv.Close()

	result, err := m.Search("arrendamento")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Type != "list" {
		t.Fatalf("Type = %q, want list", result.Type)
	}
	if len(result.Hits) != 2 {
		t.Fatalf("Hits = %+v, want 2 entries", result.Hits)
	}
	if result.Hits[0].DocumentID != "doc-1" || result.Hits[1].DocumentID != "doc-2" {
		t.Fatalf("Hits in wrong order: %+v", result.Hits)
	}
}

func TestSearchFiltersOutScoresBelowListFloor(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("doc-1\t0.5\ndoc-2\t0.02\n"))
	}, &stubEmbedder{})
	defer srv.Close()

	result, err := m.Search("arrendamento")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(result.Hits) != 1 || result.Hits[0].DocumentID != "doc-1" {
		t.Fatalf("Hits = %+v, want only doc-1 (doc-2 abaixo do kListFloor é ruído)", result.Hits)
	}
}

func TestSearchParsesNone(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(""))
	}, &stubEmbedder{})
	defer srv.Close()

	result, err := m.Search("assunto inexistente")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Type != "none" {
		t.Fatalf("Type = %q, want none", result.Type)
	}
	if len(result.Hits) != 0 {
		t.Fatalf("Hits = %+v, want none", result.Hits)
	}
}

func TestSearchWithoutEmbedderFailsClearly(t *testing.T) {
	m, srv := newTestManager(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("não devia chegar a contactar o search-service sem embedder")
	}, nil)
	defer srv.Close()

	if _, err := m.Search("qualquer coisa"); err == nil {
		t.Fatal("Search did not return an error when no embedder is set")
	}
}

func TestStartFailsFastWhenBinaryMissing(t *testing.T) {
	m := New("./this-binary-does-not-exist")
	if err := m.Start(); err == nil {
		t.Fatal("Start did not fail for a nonexistent binary")
	}
}
