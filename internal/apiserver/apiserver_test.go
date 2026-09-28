package apiserver

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"DocumentApp/internal/backend"
)

// newTestServer arranca um LocalBackend real (sem ai-worker nem
// search-service — só o que apiserver precisa de expor) atrás de um
// httptest.Server com o mux+auth do apiserver — testa o protocolo HTTP
// completo, não só handlers isolados.
func newTestServer(t *testing.T, token string) (*httptest.Server, *backend.LocalBackend) {
	t.Helper()
	dir := t.TempDir()
	lb, err := backend.NewLocal(backend.Config{
		DBPath:    filepath.Join(dir, "test.db"),
		FilesRoot: filepath.Join(dir, "files"),
	})
	if err != nil {
		t.Fatalf("backend.NewLocal: %v", err)
	}
	t.Cleanup(func() { lb.Close() })

	srv := New(lb, token)
	ts := httptest.NewServer(srv.withAuth(srv.mux))
	t.Cleanup(ts.Close)
	return ts, lb
}

func authedGet(t *testing.T, ts *httptest.Server, token, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func TestRejectsMissingOrWrongToken(t *testing.T) {
	ts, _ := newTestServer(t, "correct-token")

	resp := authedGet(t, ts, "", "/api/folders")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status = %d, want 401", resp.StatusCode)
	}

	resp2 := authedGet(t, ts, "wrong-token", "/api/folders")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong token: status = %d, want 401", resp2.StatusCode)
	}
}

func TestAcceptsCorrectToken(t *testing.T) {
	ts, _ := newTestServer(t, "correct-token")

	resp := authedGet(t, ts, "correct-token", "/api/folders")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRemoteBackendFullRoundTrip(t *testing.T) {
	ts, _ := newTestServer(t, "team-token")
	rb := backend.NewRemote(ts.URL, "team-token")
	defer rb.Close()

	folder, err := rb.CreateFolder("Contratos", "grad", true)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	folders, err := rb.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folders) != 1 || folders[0].ID != folder.ID {
		t.Fatalf("ListFolders = %+v, want just %+v", folders, folder)
	}

	doc, err := rb.IngestFile(folder.ID, "contrato-arrendamento.pdf", 9, strings.NewReader("conteudo!"))
	if err != nil {
		t.Fatalf("IngestFile: %v", err)
	}
	if doc.FileName != "contrato-arrendamento.pdf" {
		t.Fatalf("FileName = %q, want contrato-arrendamento.pdf", doc.FileName)
	}
	if doc.SizeBytes != 9 {
		t.Fatalf("SizeBytes = %d, want 9", doc.SizeBytes)
	}

	docs, err := rb.ListDocuments(folder.ID)
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 || docs[0].ID != doc.ID {
		t.Fatalf("ListDocuments = %+v, want just %+v", docs, doc)
	}

	if err := rb.ApproveDocument(doc.ID); err != nil {
		t.Fatalf("ApproveDocument: %v", err)
	}
	docs, _ = rb.ListDocuments(folder.ID)
	if docs[0].Status != "ready" {
		t.Fatalf("after Approve, status = %q, want ready", docs[0].Status)
	}

	rc, fileName, err := rb.DocumentFile(doc.ID)
	if err != nil {
		t.Fatalf("DocumentFile: %v", err)
	}
	defer rc.Close()
	if fileName != "contrato-arrendamento.pdf" {
		t.Fatalf("downloaded fileName = %q, want contrato-arrendamento.pdf", fileName)
	}
	buf := make([]byte, 9)
	n, _ := rc.Read(buf)
	if string(buf[:n]) != "conteudo!" {
		t.Fatalf("downloaded content = %q, want conteudo!", buf[:n])
	}
}

func TestRemoteBackendDeleteDocumentAndFolder(t *testing.T) {
	ts, _ := newTestServer(t, "team-token")
	rb := backend.NewRemote(ts.URL, "team-token")
	defer rb.Close()

	folder, _ := rb.CreateFolder("Contratos", "grad", false)
	doc, _ := rb.IngestFile(folder.ID, "a.pdf", 1, strings.NewReader("x"))

	if err := rb.DeleteDocument(doc.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	docs, _ := rb.ListDocuments(folder.ID)
	if len(docs) != 0 {
		t.Fatalf("ListDocuments after DeleteDocument: got %+v, want none", docs)
	}

	if err := rb.DeleteFolder(folder.ID); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	folders, _ := rb.ListFolders()
	if len(folders) != 0 {
		t.Fatalf("ListFolders after DeleteFolder: got %+v, want none", folders)
	}
}

func TestRemoteBackendSearchPropagatesHostError(t *testing.T) {
	// O host de teste não tem search-service (ver newTestServer) — isto
	// confirma que um erro do lado do host atravessa a fronteira HTTP em
	// vez de ser engolido em silêncio (ex. devolvido como "none").
	ts, _ := newTestServer(t, "team-token")
	rb := backend.NewRemote(ts.URL, "team-token")
	defer rb.Close()

	if _, err := rb.Search("qualquer coisa"); err == nil {
		t.Fatal("Search against a host without a search-service: expected an error, got nil")
	}
}

func TestRemoteBackendRejectDocumentAndListDocumentsMissingFolderID(t *testing.T) {
	ts, _ := newTestServer(t, "team-token")
	rb := backend.NewRemote(ts.URL, "team-token")
	defer rb.Close()

	folder, _ := rb.CreateFolder("Contratos", "grad", false)
	doc, _ := rb.IngestFile(folder.ID, "doc.pdf", 1, strings.NewReader("x"))

	if err := rb.RejectDocument(doc.ID); err != nil {
		t.Fatalf("RejectDocument: %v", err)
	}
	docs, _ := rb.ListDocuments(folder.ID)
	if docs[0].Status != "rejected" {
		t.Fatalf("after Reject, status = %q, want rejected", docs[0].Status)
	}

	// ListDocuments sem folder_id (chamada directa ao endpoint, não via
	// RemoteBackend, que nunca o omitiria) tem de devolver 400, não uma
	// lista vazia silenciosa.
	resp := authedGet(t, ts, "team-token", "/api/documents")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /api/documents sem folder_id: status = %d, want 400", resp.StatusCode)
	}
}
