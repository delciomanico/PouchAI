package documents

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"DocumentApp/internal/db"
	"DocumentApp/internal/folders"
)

func newTestFixtures(t *testing.T) (*sql.DB, *folders.Service, *Service) {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, folders.New(conn), New(conn)
}

// insertDocument grava uma linha directamente (sem passar pela fila de
// IA) só para preparar fixtures de teste destes serviços de leitura.
func insertDocument(t *testing.T, conn *sql.DB, folderID, fileName, status string) Document {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	d := Document{
		ID:        uuid.NewString(),
		FileName:  fileName,
		FolderID:  folderID,
		SizeBytes: 42,
		LocalPath: "/tmp/" + fileName,
		Status:    status,
		TagsJSON:  "[]",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := conn.Exec(
		`INSERT INTO documents (id, file_name, folder_id, size_bytes, local_path, status, tags_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.FileName, d.FolderID, d.SizeBytes, d.LocalPath, d.Status, d.TagsJSON, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("insertDocument: %v", err)
	}
	return d
}

func TestListByFolder(t *testing.T) {
	conn, folderSvc, docSvc := newTestFixtures(t)

	folder, err := folderSvc.Create("Contratos", "linear-gradient(a,b)", false)
	if err != nil {
		t.Fatalf("folders.Create: %v", err)
	}

	if docs, err := docSvc.ListByFolder(folder.ID); err != nil {
		t.Fatalf("ListByFolder (empty): %v", err)
	} else if len(docs) != 0 {
		t.Fatalf("ListByFolder (empty): got %d docs, want 0", len(docs))
	}

	inserted := insertDocument(t, conn, folder.ID, "contrato.pdf", StatusReady)

	docs, err := docSvc.ListByFolder(folder.ID)
	if err != nil {
		t.Fatalf("ListByFolder: %v", err)
	}
	if len(docs) != 1 || docs[0] != inserted {
		t.Fatalf("ListByFolder: got %+v, want [%+v]", docs, inserted)
	}
}

func TestGet(t *testing.T) {
	conn, folderSvc, docSvc := newTestFixtures(t)

	folder, _ := folderSvc.Create("Contratos", "linear-gradient(a,b)", false)
	inserted := insertDocument(t, conn, folder.ID, "doc.pdf", StatusReady)

	got, err := docSvc.Get(inserted.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != inserted {
		t.Fatalf("Get: got %+v, want %+v", got, inserted)
	}

	if _, err := docSvc.Get("id-inexistente"); err == nil {
		t.Fatal("Get: expected an error for an unknown id, got nil")
	}
}

func TestApproveAndReject(t *testing.T) {
	conn, folderSvc, docSvc := newTestFixtures(t)

	folder, _ := folderSvc.Create("Contratos", "linear-gradient(a,b)", false)

	toApprove := insertDocument(t, conn, folder.ID, "a.pdf", StatusPendingReview)
	if err := docSvc.Approve(toApprove.ID); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	got, err := docSvc.Get(toApprove.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusReady {
		t.Fatalf("after Approve: Status = %q, want %q", got.Status, StatusReady)
	}

	toReject := insertDocument(t, conn, folder.ID, "b.pdf", StatusPendingReview)
	if err := docSvc.Reject(toReject.ID); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	got, err = docSvc.Get(toReject.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusRejected {
		t.Fatalf("after Reject: Status = %q, want %q", got.Status, StatusRejected)
	}

	if err := docSvc.Approve("id-inexistente"); err == nil {
		t.Fatal("Approve: expected an error for an unknown id, got nil")
	}
}

func TestDelete(t *testing.T) {
	conn, folderSvc, docSvc := newTestFixtures(t)
	folder, _ := folderSvc.Create("Contratos", "grad", false)

	doc := insertDocument(t, conn, folder.ID, "a.pdf", StatusPendingReview)
	if err := docSvc.Delete(doc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := docSvc.Get(doc.ID); err == nil {
		t.Fatal("Get after Delete: expected an error, got nil")
	}
}

func TestDeleteNotFound(t *testing.T) {
	_, _, docSvc := newTestFixtures(t)
	if err := docSvc.Delete("id-inexistente"); err == nil {
		t.Fatal("Delete: expected an error for an unknown id, got nil")
	}
}

func TestBuildIndexTextIncludesRealExcerptAndTags(t *testing.T) {
	d := Document{
		FileName:     "contrato-arrendamento.pdf",
		DocumentType: "PDF",
		TagsJSON:     `["imobiliario","contrato"]`,
		OCRExcerpt:   "Contrato de arrendamento entre Fulano e Beltrano, renda mensal de 500€.",
	}
	got := BuildIndexText(d)
	want := "contrato-arrendamento.pdf PDF imobiliario contrato Contrato de arrendamento entre Fulano e Beltrano, renda mensal de 500€."
	if got != want {
		t.Fatalf("BuildIndexText = %q, want %q", got, want)
	}
}

func TestBuildIndexTextHandlesEmptyOptionalFields(t *testing.T) {
	d := Document{FileName: "nota.txt"}
	got := BuildIndexText(d)
	want := "nota.txt"
	if got != want {
		t.Fatalf("BuildIndexText = %q, want %q", got, want)
	}
}
