// Package jobs implementa a fila de IA: Enqueue cria o documento e o
// job numa única transação. A partir daí, quem faz a transição de
// estados do documento (uploading -> processing -> pending_review /
// failed) e do job (queued -> claimed -> done / failed) é o ai-worker
// em C++ (claimNext e finish, ver ai-worker/src/main.cpp) — este
// package Go não escreve nesses campos depois do Enqueue. Ver
// ROADMAP.md, Fase 3.
package jobs

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"DocumentApp/internal/documents"
)

const JobTypeOCRClassify = "ocr_classify"

const (
	StatusQueued  = "queued"
	StatusClaimed = "claimed"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// Enqueue grava o documento como "uploading" e um job "queued" para o
// processar, numa única transação — só depois disto o ai-worker o pode
// reclamar.
//
// id, fileName, localPath e sizeBytes vêm já resolvidos pelo chamador
// (internal/backend.LocalBackend, que primeiro grava o conteúdo na
// pasta gerida via internal/filestore — ver ROADMAP.md, Fase 6): este
// package deixou de fazer os.Stat de um caminho local, porque em modo
// cliente/servidor esse caminho pode nem existir no disco de quem
// chama a API, só no do host.
func (s *Service) Enqueue(id, folderID, fileName, localPath string, sizeBytes int64) (documents.Document, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	doc := documents.Document{
		ID:        id,
		FileName:  fileName,
		FolderID:  folderID,
		SizeBytes: sizeBytes,
		LocalPath: localPath,
		Status:    documents.StatusUploading,
		TagsJSON:  "[]",
		CreatedAt: now,
		UpdatedAt: now,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return documents.Document{}, fmt.Errorf("enqueue %q: %w", fileName, err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO documents (id, file_name, folder_id, size_bytes, local_path, status, tags_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.FileName, doc.FolderID, doc.SizeBytes, doc.LocalPath, doc.Status, doc.TagsJSON, doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return documents.Document{}, fmt.Errorf("enqueue %q: insert document: %w", fileName, err)
	}

	_, err = tx.Exec(
		`INSERT INTO ai_jobs (id, document_id, job_type, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), doc.ID, JobTypeOCRClassify, StatusQueued, now, now,
	)
	if err != nil {
		return documents.Document{}, fmt.Errorf("enqueue %q: insert job: %w", fileName, err)
	}

	if err := tx.Commit(); err != nil {
		return documents.Document{}, fmt.Errorf("enqueue %q: %w", fileName, err)
	}
	return doc, nil
}

// DeleteByDocument apaga todos os ai_jobs de um documento — tem de
// correr antes de documents.Service.Delete, porque ai_jobs.document_id
// referencia documents(id) e o schema tem PRAGMA foreign_keys=ON.
func (s *Service) DeleteByDocument(documentID string) error {
	if _, err := s.db.Exec(`DELETE FROM ai_jobs WHERE document_id = ?`, documentID); err != nil {
		return fmt.Errorf("delete jobs for document %q: %w", documentID, err)
	}
	return nil
}

// CountQueued devolve quantos jobs estão à espera de ser reclamados —
// usado pelo pool elástico para decidir se vale a pena lançar mais um
// worker (ver internal/workerpool).
func (s *Service) CountQueued() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM ai_jobs WHERE status = ?`, StatusQueued).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count queued jobs: %w", err)
	}
	return n, nil
}

// CountPending devolve quantos jobs estão em curso (queued + claimed)
// — usado por IngestFile para recusar novas entradas com a fila cheia
// (contrapressão, ver ROADMAP.md Fase 4).
func (s *Service) CountPending() (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM ai_jobs WHERE status IN (?, ?)`, StatusQueued, StatusClaimed,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count pending jobs: %w", err)
	}
	return n, nil
}

// ReleaseStale devolve à fila (status "queued", claimed_by limpo) todo
// o job "claimed" há mais tempo que olderThan, e o documento associado
// a "uploading" — a lease que apanha um ai-worker morto (ou preso)
// mesmo sem depender do heartbeat. Devolve os IDs dos documentos
// afectados, para quem chamar poder emitir "document:updated".
func (s *Service) ReleaseStale(olderThan time.Duration) ([]string, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	return s.release(`updated_at < ?`, cutoff)
}

// ReleaseByWorker devolve à fila todos os jobs actualmente reclamados
// por workerID — chamado quando o heartbeat desse worker pára de
// chegar (ver internal/heartbeats).
func (s *Service) ReleaseByWorker(workerID string) ([]string, error) {
	return s.release(`claimed_by = ?`, workerID)
}

func (s *Service) release(whereExtra string, arg interface{}) ([]string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("release jobs: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		`SELECT id, document_id FROM ai_jobs WHERE status = ? AND `+whereExtra,
		StatusClaimed, arg,
	)
	if err != nil {
		return nil, fmt.Errorf("release jobs: select candidates: %w", err)
	}
	type candidate struct{ jobID, documentID string }
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.jobID, &c.documentID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("release jobs: scan candidate: %w", err)
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("release jobs: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	documentIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if _, err := tx.Exec(
			`UPDATE ai_jobs SET status = ?, claimed_by = NULL, updated_at = ? WHERE id = ?`,
			StatusQueued, now, c.jobID,
		); err != nil {
			return nil, fmt.Errorf("release job %q: %w", c.jobID, err)
		}
		if _, err := tx.Exec(
			`UPDATE documents SET status = ?, updated_at = ? WHERE id = ?`,
			documents.StatusUploading, now, c.documentID,
		); err != nil {
			return nil, fmt.Errorf("release job %q: update document: %w", c.jobID, err)
		}
		documentIDs = append(documentIDs, c.documentID)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("release jobs: %w", err)
	}
	return documentIDs, nil
}
