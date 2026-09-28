// Package documents implementa a leitura e as transições de estado de
// documentos sobre a tabela `documents`. A criação de documentos vive em
// internal/jobs (Enqueue), que cria o documento e o job de IA numa única
// transação — ver ROADMAP.md, Fase 3.
package documents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	StatusUploading     = "uploading"
	StatusProcessing    = "processing"
	StatusPendingReview = "pending_review"
	StatusReady         = "ready"
	StatusFailed        = "failed"
	StatusRejected      = "rejected"
)

type Document struct {
	ID                    string  `json:"id"`
	FileName              string  `json:"fileName"`
	FolderID              string  `json:"folderId"`
	DocumentType          string  `json:"documentType"`
	Author                string  `json:"author"`
	SizeBytes             int64   `json:"sizeBytes"`
	LocalPath             string  `json:"localPath"`
	Status                string  `json:"status"`
	OCRConfidence         float64 `json:"ocrConfidence"`
	OCRExcerpt            string  `json:"ocrExcerpt"`
	Summary               string  `json:"summary"`
	TagsJSON              string  `json:"tagsJson"`
	ClassificationPending bool    `json:"classificationPending"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

const selectColumns = `id, file_name, folder_id, document_type, author, size_bytes,
	local_path, status, ocr_confidence, ocr_excerpt, summary, tags_json,
	classification_pending, created_at, updated_at`

func scanDocument(row interface{ Scan(...any) error }) (Document, error) {
	var d Document
	var documentType, author, ocrExcerpt, summary sql.NullString
	var ocrConfidence sql.NullFloat64
	var classificationPending int

	err := row.Scan(
		&d.ID, &d.FileName, &d.FolderID, &documentType, &author, &d.SizeBytes,
		&d.LocalPath, &d.Status, &ocrConfidence, &ocrExcerpt, &summary, &d.TagsJSON,
		&classificationPending, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return Document{}, err
	}
	d.DocumentType = documentType.String
	d.Author = author.String
	d.OCRExcerpt = ocrExcerpt.String
	d.Summary = summary.String
	d.OCRConfidence = ocrConfidence.Float64
	d.ClassificationPending = classificationPending != 0
	return d, nil
}

func (s *Service) ListByFolder(folderID string) ([]Document, error) {
	rows, err := s.db.Query(
		`SELECT `+selectColumns+` FROM documents WHERE folder_id = ? ORDER BY created_at ASC`,
		folderID,
	)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	docs := []Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	return docs, nil
}

func (s *Service) Get(id string) (Document, error) {
	row := s.db.QueryRow(`SELECT `+selectColumns+` FROM documents WHERE id = ?`, id)
	d, err := scanDocument(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Document{}, fmt.Errorf("get document %q: not found", id)
		}
		return Document{}, fmt.Errorf("get document %q: %w", id, err)
	}
	return d, nil
}

func (s *Service) setStatus(id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE documents SET status = ?, updated_at = ? WHERE id = ?`,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("set status of document %q to %q: %w", id, status, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set status of document %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("set status of document %q: not found", id)
	}
	return nil
}

func (s *Service) Approve(id string) error {
	return s.setStatus(id, StatusReady)
}

func (s *Service) Reject(id string) error {
	return s.setStatus(id, StatusRejected)
}

// Delete remove definitivamente a linha do documento — chamado depois
// de internal/jobs.DeleteByDocument (para não violar a foreign key
// ai_jobs.document_id) e antes de internal/filestore.Delete (que apaga
// o conteúdo em disco). Não há "soft delete"/tombstone nesta fase — ver
// ROADMAP.md, Fase 7, para quando isso vier a ser preciso (sync).
func (s *Service) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete document %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete document %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("delete document %q: not found", id)
	}
	return nil
}

// ListIndexable devolve os documentos que o search-service (Fase 5) deve
// ter no seu índice em memória: já passaram pelo OCR (têm summary) e
// ainda estão num estado em que faz sentido aparecer numa pesquisa.
func (s *Service) ListIndexable() ([]Document, error) {
	rows, err := s.db.Query(
		`SELECT ` + selectColumns + ` FROM documents
		 WHERE status IN ('pending_review', 'ready') AND summary IS NOT NULL AND summary != ''
		 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list indexable documents: %w", err)
	}
	defer rows.Close()

	docs := []Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list indexable documents: %w", err)
	}
	return docs, nil
}

// UpdateSummary grava o resumo derivado (ver BuildSummary) calculado
// quando o OCR/classificação termina — é o que aparece nos resultados de
// pesquisa no React (o texto indexado para a pesquisa em si é diferente,
// ver BuildIndexText).
func (s *Service) UpdateSummary(id, summary string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE documents SET summary = ?, updated_at = ? WHERE id = ?`,
		summary, now, id,
	)
	if err != nil {
		return fmt.Errorf("update summary of document %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update summary of document %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("update summary of document %q: not found", id)
	}
	return nil
}

// UpdateClassification grava, numa só escrita, o resultado do
// enriquecimento real via llm-service (ver internal/callbackserver.Enricher
// e o plano em unified-popping-sunset.md): tipo de documento (embeddings +
// cosseno), resumo (gerado pelo modelo de chat) e tags (JSON, mesmo formato
// de TagsJSON). Substitui a necessidade de três UPDATE separados; o
// caminho de fallback sem llm-service continua a usar só UpdateSummary.
// Limpa sempre classification_pending — chamada tanto quando o
// enriquecimento é bem sucedido como quando falha (ver enrichAsync), nos
// dois casos já não há nada a aguardar.
func (s *Service) UpdateClassification(id, documentType, summary, tagsJSON string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE documents SET document_type = ?, summary = ?, tags_json = ?, classification_pending = 0, updated_at = ? WHERE id = ?`,
		documentType, summary, tagsJSON, now, id,
	)
	if err != nil {
		return fmt.Errorf("update classification of document %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update classification of document %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("update classification of document %q: not found", id)
	}
	return nil
}

// MarkEnrichmentPending assinala que o enriquecimento real (llm-service)
// vai começar a correr em segundo plano para este documento — chamado
// mesmo antes de lançar a goroutine (ver callbackserver.handleCallback).
// UpdateClassification limpa sempre esta flag quando o enriquecimento
// termina, com sucesso ou não.
func (s *Service) MarkEnrichmentPending(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE documents SET classification_pending = 1, updated_at = ? WHERE id = ?`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("mark enrichment pending for document %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark enrichment pending for document %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("mark enrichment pending for document %q: not found", id)
	}
	return nil
}

// UpdateManualClassification aplica uma correção manual do utilizador ao
// tipo de documento e às tags sugeridas pelo modelo — ao contrário de
// UpdateClassification, nunca mexe no resumo nem em classification_pending
// (isto acontece depois do enriquecimento automático já ter terminado,
// durante a revisão antes de aprovar).
func (s *Service) UpdateManualClassification(id, documentType, tagsJSON string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE documents SET document_type = ?, tags_json = ?, updated_at = ? WHERE id = ?`,
		documentType, tagsJSON, now, id,
	)
	if err != nil {
		return fmt.Errorf("update manual classification of document %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update manual classification of document %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("update manual classification of document %q: not found", id)
	}
	return nil
}

// BuildSummary deriva o texto de resumo (mostrado ao utilizador nos
// resultados de pesquisa) a partir dos campos que o ai-worker já
// preencheu. Não há resumo real de linguagem natural (não há OCR real
// ainda — ver ai-worker/src/main.cpp); isto é a melhor aproximação
// honesta com o que existe: tipo + excerto.
func BuildSummary(d Document) string {
	parts := make([]string, 0, 2)
	if d.DocumentType != "" {
		parts = append(parts, d.DocumentType)
	}
	if d.OCRExcerpt != "" {
		parts = append(parts, d.OCRExcerpt)
	}
	return strings.Join(parts, " — ")
}

// BuildIndexText deriva o texto que é efectivamente indexado pelo
// search-service. Historicamente excluía o OCRExcerpt de propósito,
// porque era um placeholder fixo ("Texto extraído (placeholder) de
// <nome>.") igual em todos os documentos, que afogava o único sinal
// distintivo (o nome do ficheiro). O ai-worker já extrai texto real de
// PDFs com camada de texto (ver ai-worker/src/main.cpp,
// extractPdfText) — continuar a ignorá-lo deixava a pesquisa e o
// assistente (RAG) cegos ao conteúdo real dos documentos, só a
// encontrar coisas por nome de ficheiro/tipo. Inclui também as tags
// (classificação) como sinal extra de pesquisa/filtragem.
func BuildIndexText(d Document) string {
	parts := make([]string, 0, 4)
	if d.FileName != "" {
		parts = append(parts, d.FileName)
	}
	if d.DocumentType != "" {
		parts = append(parts, d.DocumentType)
	}
	if tags := parseTagsJSON(d.TagsJSON); len(tags) > 0 {
		parts = append(parts, strings.Join(tags, " "))
	}
	if d.OCRExcerpt != "" {
		parts = append(parts, d.OCRExcerpt)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// parseTagsJSON descodifica o campo tags_json (array JSON de strings);
// devolve nil silenciosamente em caso de JSON vazio/inválido — tags são
// um sinal auxiliar de indexação, nunca crítico o suficiente para negar
// a indexação do resto do documento por causa disto.
func parseTagsJSON(tagsJSON string) []string {
	if tagsJSON == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		return nil
	}
	return tags
}
