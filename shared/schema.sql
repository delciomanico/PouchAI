-- Fonte única de verdade do esquema SQLite.
-- Nunca editar uma cópia deste ficheiro dentro do módulo Go — o script de
-- build (Makefile, alvo `schema`) copia este ficheiro para lá antes de
-- compilar. Editar sempre aqui.

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS folders (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    accent_gradient TEXT NOT NULL,
    is_shared       INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS documents (
    id              TEXT PRIMARY KEY,
    file_name       TEXT NOT NULL,
    folder_id       TEXT NOT NULL REFERENCES folders(id),
    document_type   TEXT,
    author          TEXT,
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    local_path      TEXT NOT NULL,
    -- 'rejected' não consta na lista de referência do ROADMAP.md, mas é
    -- o estado terminal de RejectDocument (Fase 1) — sem ele, rejeitar
    -- um documento não tinha para onde ir.
    status          TEXT NOT NULL DEFAULT 'uploading'
                    CHECK (status IN ('uploading','processing','pending_review','ready','failed','rejected')),
    ocr_confidence  REAL,
    ocr_excerpt     TEXT,
    -- summary: derivado (document_type + ocr_excerpt) quando o OCR termina,
    -- ver internal/documents.BuildSummary — é o texto indexado pelo
    -- search-service (Fase 5), guardado aqui para não depender de o
    -- search-service (processo separado, índice só em memória) sobreviver
    -- a um reinício sem reindexação.
    summary         TEXT,
    tags_json       TEXT NOT NULL DEFAULT '[]',
    -- classification_pending: true entre o momento em que o OCR termina
    -- (document_type/summary já têm o fallback derivado por extensão) e o
    -- momento em que o llm-service devolve o resumo/classificação reais
    -- (ver internal/callbackserver.enrichAsync) — o enriquecimento corre
    -- em segundo plano e pode demorar dezenas de segundos. A UI usa isto
    -- para só mostrar "Aprovar" depois de ter a classificação real, em
    -- vez do fallback (ver IngestFlowPage.tsx).
    classification_pending INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_folder_id ON documents(folder_id);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);

CREATE TABLE IF NOT EXISTS ai_jobs (
    id              TEXT PRIMARY KEY,
    document_id     TEXT NOT NULL REFERENCES documents(id),
    job_type        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued','claimed','done','failed')),
    claimed_by      TEXT,
    result_json     TEXT,
    error           TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_jobs_status ON ai_jobs(status);
CREATE INDEX IF NOT EXISTS idx_ai_jobs_document_id ON ai_jobs(document_id);
