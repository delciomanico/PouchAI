package backend

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"DocumentApp/internal/documents"
)

// newTestLocalBackend arranca um LocalBackend sem ai-worker nem
// search-service (caminhos vazios — degradação graciosa, ver
// NewLocal) — suficiente para testar as operações de dados em si; o
// pipeline de IA real já está coberto pelos testes de
// internal/jobs/internal/workerpool/internal/callbackserver, e o
// search-service pelos de internal/searchservice.
func newTestLocalBackend(t *testing.T) (*LocalBackend, string) {
	t.Helper()
	dir := t.TempDir()
	filesRoot := filepath.Join(dir, "files")
	b, err := NewLocal(Config{
		DBPath:    filepath.Join(dir, "test.db"),
		FilesRoot: filesRoot,
	})
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	t.Cleanup(func() { b.Close() })
	return b, filesRoot
}

func TestLocalBackendFolderAndDocumentCRUD(t *testing.T) {
	b, _ := newTestLocalBackend(t)

	folder, err := b.CreateFolder("Contratos", "grad", false)
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	list, err := b.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(list) != 1 || list[0].ID != folder.ID {
		t.Fatalf("ListFolders = %+v, want just %+v", list, folder)
	}

	docs, err := b.ListDocuments(folder.ID)
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("ListDocuments (empty folder) = %+v, want none", docs)
	}
}

func TestLocalBackendIngestFileWritesToManagedFolderAndEnqueues(t *testing.T) {
	b, filesRoot := newTestLocalBackend(t)
	folder, _ := b.CreateFolder("Contratos", "grad", false)

	doc, err := b.IngestFile(folder.ID, "contrato.pdf", 9, strings.NewReader("conteudo!"))
	if err != nil {
		t.Fatalf("IngestFile: %v", err)
	}
	if doc.Status != documents.StatusUploading {
		t.Fatalf("Status = %q, want %q", doc.Status, documents.StatusUploading)
	}
	if doc.SizeBytes != 9 {
		t.Fatalf("SizeBytes = %d, want 9 (measured while writing, not trusted from caller)", doc.SizeBytes)
	}

	// O caminho gravado tem de estar dentro da pasta gerida (files/<id>/…),
	// nunca o caminho original — é o ponto central desta fase.
	wantDir := filepath.Join(filesRoot, doc.ID)
	if !strings.HasPrefix(doc.LocalPath, wantDir) {
		t.Fatalf("LocalPath = %q, want it under the managed folder %q", doc.LocalPath, wantDir)
	}

	got, err := os.ReadFile(doc.LocalPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", doc.LocalPath, err)
	}
	if string(got) != "conteudo!" {
		t.Fatalf("file content = %q, want %q", got, "conteudo!")
	}

	docs, err := b.ListDocuments(folder.ID)
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 || docs[0].ID != doc.ID {
		t.Fatalf("ListDocuments = %+v, want just %+v", docs, doc)
	}
}

func TestLocalBackendIngestFileRespectsBackpressure(t *testing.T) {
	b, _ := newTestLocalBackend(t)
	folder, _ := b.CreateFolder("Contratos", "grad", false)

	for i := 0; i < maxPendingJobs; i++ {
		if _, err := b.IngestFile(folder.ID, "doc.pdf", 1, strings.NewReader("x")); err != nil {
			t.Fatalf("IngestFile #%d: %v", i, err)
		}
	}

	if _, err := b.IngestFile(folder.ID, "doc.pdf", 1, strings.NewReader("x")); err == nil {
		t.Fatal("IngestFile past maxPendingJobs: expected an error, got nil")
	}
}

func TestLocalBackendApproveAndReject(t *testing.T) {
	b, _ := newTestLocalBackend(t)
	folder, _ := b.CreateFolder("Contratos", "grad", false)
	doc, _ := b.IngestFile(folder.ID, "doc.pdf", 1, strings.NewReader("x"))

	if err := b.ApproveDocument(doc.ID); err != nil {
		t.Fatalf("ApproveDocument: %v", err)
	}
	docs, _ := b.ListDocuments(folder.ID)
	if docs[0].Status != documents.StatusReady {
		t.Fatalf("after Approve, status = %q, want %q", docs[0].Status, documents.StatusReady)
	}

	if err := b.RejectDocument(doc.ID); err != nil {
		t.Fatalf("RejectDocument: %v", err)
	}
	docs, _ = b.ListDocuments(folder.ID)
	if docs[0].Status != documents.StatusRejected {
		t.Fatalf("after Reject, status = %q, want %q", docs[0].Status, documents.StatusRejected)
	}
}

func TestLocalBackendDocumentFile(t *testing.T) {
	b, _ := newTestLocalBackend(t)
	folder, _ := b.CreateFolder("Contratos", "grad", false)
	doc, _ := b.IngestFile(folder.ID, "contrato.pdf", 4, strings.NewReader("abcd"))

	rc, fileName, err := b.DocumentFile(doc.ID)
	if err != nil {
		t.Fatalf("DocumentFile: %v", err)
	}
	defer rc.Close()

	if fileName != "contrato.pdf" {
		t.Fatalf("fileName = %q, want contrato.pdf", fileName)
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != "abcd" {
		t.Fatalf("content = %q, want abcd", got)
	}
}

func TestLocalBackendDeleteDocumentRemovesRowAndFiles(t *testing.T) {
	b, _ := newTestLocalBackend(t)
	folder, _ := b.CreateFolder("Contratos", "grad", false)
	doc, _ := b.IngestFile(folder.ID, "contrato.pdf", 4, strings.NewReader("abcd"))

	if err := b.DeleteDocument(doc.ID); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	docs, err := b.ListDocuments(folder.ID)
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("ListDocuments after DeleteDocument: got %+v, want none", docs)
	}
	if _, err := os.Stat(doc.LocalPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) after DeleteDocument: err = %v, want IsNotExist", doc.LocalPath, err)
	}
}

func TestLocalBackendDeleteFolderCascadesToDocuments(t *testing.T) {
	b, _ := newTestLocalBackend(t)
	folder, _ := b.CreateFolder("Contratos", "grad", false)
	doc1, _ := b.IngestFile(folder.ID, "a.pdf", 1, strings.NewReader("x"))
	doc2, _ := b.IngestFile(folder.ID, "b.pdf", 1, strings.NewReader("y"))

	if err := b.DeleteFolder(folder.ID); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}

	folderList, err := b.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folderList) != 0 {
		t.Fatalf("ListFolders after DeleteFolder: got %+v, want none", folderList)
	}
	if _, err := os.Stat(doc1.LocalPath); !os.IsNotExist(err) {
		t.Fatalf("doc1 file should be gone, err = %v", err)
	}
	if _, err := os.Stat(doc2.LocalPath); !os.IsNotExist(err) {
		t.Fatalf("doc2 file should be gone, err = %v", err)
	}
}

func TestLocalBackendSearchWithoutSearchServiceErrors(t *testing.T) {
	b, _ := newTestLocalBackend(t)
	if _, err := b.Search("qualquer coisa"); err == nil {
		t.Fatal("Search without a search-service: expected an error, got nil")
	}
}

func TestLocalBackendSearchEmptyQueryReturnsNoneWithoutCallingSearchService(t *testing.T) {
	b, _ := newTestLocalBackend(t) // b.search é nil — se Search não devolver cedo, isto rebentava
	result, err := b.Search("   ")
	if err != nil {
		t.Fatalf("Search(empty): %v", err)
	}
	if result.Type != "none" {
		t.Fatalf("Type = %q, want none", result.Type)
	}
}
