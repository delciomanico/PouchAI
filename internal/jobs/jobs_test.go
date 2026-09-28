package jobs

import (
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"DocumentApp/internal/db"
	"DocumentApp/internal/documents"
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

// enqueueTestDoc chama Enqueue com valores sintéticos razoáveis — o que
// o internal/backend.LocalBackend faria depois de gravar o ficheiro via
// internal/filestore, sem precisar de um ficheiro real em disco aqui.
func enqueueTestDoc(t *testing.T, jobSvc *Service, folderID string) documents.Document {
	t.Helper()
	doc, err := jobSvc.Enqueue(uuid.NewString(), folderID, "contrato.pdf", "/managed/path/contrato.pdf", 9)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	return doc
}

func TestEnqueueCreatesDocumentAndJob(t *testing.T) {
	conn, folderSvc, jobSvc := newTestFixtures(t)

	folder, err := folderSvc.Create("Contratos", "linear-gradient(a,b)", false)
	if err != nil {
		t.Fatalf("folders.Create: %v", err)
	}

	doc := enqueueTestDoc(t, jobSvc, folder.ID)
	if doc.Status != documents.StatusUploading {
		t.Fatalf("Enqueue: document Status = %q, want %q", doc.Status, documents.StatusUploading)
	}

	var jobStatus, jobType, documentID string
	err = conn.QueryRow(
		`SELECT status, job_type, document_id FROM ai_jobs WHERE document_id = ?`, doc.ID,
	).Scan(&jobStatus, &jobType, &documentID)
	if err != nil {
		t.Fatalf("query ai_jobs: %v", err)
	}
	if jobStatus != StatusQueued {
		t.Fatalf("job status = %q, want %q", jobStatus, StatusQueued)
	}
	if jobType != JobTypeOCRClassify {
		t.Fatalf("job_type = %q, want %q", jobType, JobTypeOCRClassify)
	}
	if documentID != doc.ID {
		t.Fatalf("ai_jobs.document_id = %q, want %q", documentID, doc.ID)
	}
}

func TestEnqueueDuplicateIDFails(t *testing.T) {
	_, folderSvc, jobSvc := newTestFixtures(t)
	folder, _ := folderSvc.Create("Contratos", "grad", false)

	id := uuid.NewString()
	if _, err := jobSvc.Enqueue(id, folder.ID, "a.pdf", "/managed/a.pdf", 1); err != nil {
		t.Fatalf("first Enqueue: %v", err)
	}
	if _, err := jobSvc.Enqueue(id, folder.ID, "a.pdf", "/managed/a.pdf", 1); err == nil {
		t.Fatal("Enqueue: expected an error reusing the same document ID, got nil")
	}

	n, err := jobSvc.CountQueued()
	if err != nil {
		t.Fatalf("CountQueued: %v", err)
	}
	if n != 1 {
		t.Fatalf("CountQueued after a failed Enqueue: got %d, want 1 (second transaction should have rolled back)", n)
	}
}

func TestCountQueued(t *testing.T) {
	_, folderSvc, jobSvc := newTestFixtures(t)

	folder, _ := folderSvc.Create("Contratos", "linear-gradient(a,b)", false)

	if n, err := jobSvc.CountQueued(); err != nil || n != 0 {
		t.Fatalf("CountQueued (empty): n=%d err=%v", n, err)
	}

	for i := 0; i < 3; i++ {
		enqueueTestDoc(t, jobSvc, folder.ID)
	}

	if n, err := jobSvc.CountQueued(); err != nil || n != 3 {
		t.Fatalf("CountQueued: n=%d err=%v, want 3", n, err)
	}
}

// enqueueAndClaim cria um documento+job (via Enqueue) e simula um
// ai-worker a reclamá-lo, com um updated_at à escolha do teste — é o
// que claimNext faria em C++ (ver ai-worker/src/main.cpp), reescrito
// aqui em SQL simples para não depender do binário nos testes Go.
func enqueueAndClaim(t *testing.T, conn *sql.DB, jobSvc *Service, folderID, workerID string, claimedAt time.Time) (jobID, documentID string) {
	t.Helper()
	doc := enqueueTestDoc(t, jobSvc, folderID)

	err := conn.QueryRow(`SELECT id FROM ai_jobs WHERE document_id = ?`, doc.ID).Scan(&jobID)
	if err != nil {
		t.Fatalf("query ai_jobs: %v", err)
	}

	claimedAtStr := claimedAt.UTC().Format(time.RFC3339)
	_, err = conn.Exec(
		`UPDATE ai_jobs SET status = ?, claimed_by = ?, updated_at = ? WHERE id = ?`,
		StatusClaimed, workerID, claimedAtStr, jobID,
	)
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	_, err = conn.Exec(`UPDATE documents SET status = ?, updated_at = ? WHERE id = ?`,
		documents.StatusProcessing, claimedAtStr, doc.ID)
	if err != nil {
		t.Fatalf("mark document processing: %v", err)
	}

	return jobID, doc.ID
}

func TestCountPending(t *testing.T) {
	conn, folderSvc, jobSvc := newTestFixtures(t)
	folder, _ := folderSvc.Create("Contratos", "grad", false)

	if n, err := jobSvc.CountPending(); err != nil || n != 0 {
		t.Fatalf("CountPending (empty): n=%d err=%v", n, err)
	}

	enqueueTestDoc(t, jobSvc, folder.ID)
	enqueueAndClaim(t, conn, jobSvc, folder.ID, "worker-1", time.Now())

	if n, err := jobSvc.CountPending(); err != nil || n != 2 {
		t.Fatalf("CountPending: n=%d err=%v, want 2 (1 queued + 1 claimed)", n, err)
	}
}

func TestReleaseStale(t *testing.T) {
	conn, folderSvc, jobSvc := newTestFixtures(t)
	folder, _ := folderSvc.Create("Contratos", "grad", false)

	staleJobID, staleDocID := enqueueAndClaim(t, conn, jobSvc, folder.ID, "worker-dead", time.Now().Add(-2*time.Hour))
	freshJobID, _ := enqueueAndClaim(t, conn, jobSvc, folder.ID, "worker-alive", time.Now())

	released, err := jobSvc.ReleaseStale(90 * time.Second)
	if err != nil {
		t.Fatalf("ReleaseStale: %v", err)
	}
	if len(released) != 1 || released[0] != staleDocID {
		t.Fatalf("ReleaseStale released %v, want [%s]", released, staleDocID)
	}

	var staleJobStatus, staleClaimedBy sql.NullString
	conn.QueryRow(`SELECT status, claimed_by FROM ai_jobs WHERE id = ?`, staleJobID).
		Scan(&staleJobStatus, &staleClaimedBy)
	if staleJobStatus.String != StatusQueued {
		t.Fatalf("stale job status = %q, want %q", staleJobStatus.String, StatusQueued)
	}
	if staleClaimedBy.Valid {
		t.Fatalf("stale job claimed_by = %q, want NULL", staleClaimedBy.String)
	}

	var staleDocStatus string
	conn.QueryRow(`SELECT status FROM documents WHERE id = ?`, staleDocID).Scan(&staleDocStatus)
	if staleDocStatus != documents.StatusUploading {
		t.Fatalf("stale document status = %q, want %q", staleDocStatus, documents.StatusUploading)
	}

	var freshJobStatus string
	conn.QueryRow(`SELECT status FROM ai_jobs WHERE id = ?`, freshJobID).Scan(&freshJobStatus)
	if freshJobStatus != StatusClaimed {
		t.Fatalf("fresh job status = %q, want %q (should be untouched)", freshJobStatus, StatusClaimed)
	}
}

func TestReleaseByWorker(t *testing.T) {
	conn, folderSvc, jobSvc := newTestFixtures(t)
	folder, _ := folderSvc.Create("Contratos", "grad", false)

	_, doc1 := enqueueAndClaim(t, conn, jobSvc, folder.ID, "dead-worker", time.Now())
	_, doc2 := enqueueAndClaim(t, conn, jobSvc, folder.ID, "dead-worker", time.Now())
	_, doc3 := enqueueAndClaim(t, conn, jobSvc, folder.ID, "other-worker", time.Now())

	released, err := jobSvc.ReleaseByWorker("dead-worker")
	if err != nil {
		t.Fatalf("ReleaseByWorker: %v", err)
	}
	sort.Strings(released)
	want := []string{doc1, doc2}
	sort.Strings(want)
	if len(released) != 2 || released[0] != want[0] || released[1] != want[1] {
		t.Fatalf("ReleaseByWorker released %v, want %v", released, want)
	}

	var otherStatus string
	conn.QueryRow(`SELECT status FROM ai_jobs WHERE document_id = ?`, doc3).Scan(&otherStatus)
	if otherStatus != StatusClaimed {
		t.Fatalf("other-worker's job status = %q, want %q (should be untouched)", otherStatus, StatusClaimed)
	}
}

func TestDeleteByDocument(t *testing.T) {
	conn, folderSvc, jobSvc := newTestFixtures(t)
	folder, _ := folderSvc.Create("Contratos", "grad", false)

	doc := enqueueTestDoc(t, jobSvc, folder.ID)
	otherDoc := enqueueTestDoc(t, jobSvc, folder.ID)

	if err := jobSvc.DeleteByDocument(doc.ID); err != nil {
		t.Fatalf("DeleteByDocument: %v", err)
	}

	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM ai_jobs WHERE document_id = ?`, doc.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("ai_jobs for deleted document: %d rows left, want 0", n)
	}
	conn.QueryRow(`SELECT COUNT(*) FROM ai_jobs WHERE document_id = ?`, otherDoc.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("ai_jobs for the OTHER document: %d rows, want 1 (should be untouched)", n)
	}

	// Depois de não haver mais ai_jobs a referenciar o documento, apagar
	// a linha do documento não pode violar a foreign key.
	if _, err := conn.Exec(`DELETE FROM documents WHERE id = ?`, doc.ID); err != nil {
		t.Fatalf("DELETE FROM documents depois de DeleteByDocument: %v", err)
	}
}
