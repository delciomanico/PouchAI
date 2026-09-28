// Package db abre a ligação SQLite e aplica o schema partilhado.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Open abre (ou cria) a base de dados em path e aplica shared/schema.sql.
//
// MaxOpenConns é fixado a 1: o SQLite não lida bem com múltiplos
// escritores, e fixar a uma única ligação evita que PRAGMAs por-ligação
// (foreign_keys, busy_timeout) se percam quando o database/sql decide
// abrir uma segunda ligação do pool.
//
// busy_timeout é necessário já na Fase 3, não só na Fase 4: com vários
// ai-worker em C++ a escrever ao mesmo tempo (cada um com a sua própria
// ligação), um SELECT vindo do Go (ex. ListDocuments) apanha
// SQLITE_BUSY sem isto — visto em primeira mão a testar múltiplos
// ficheiros em simultâneo. O lado C++ já define o seu próprio
// busy_timeout (ver ai-worker/src/main.cpp).
func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := conn.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	if err := applySchema(conn); err != nil {
		conn.Close()
		return nil, err
	}
	if err := applyMigrations(conn); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func applySchema(conn *sql.DB) error {
	for _, stmt := range strings.Split(schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := conn.Exec(stmt); err != nil {
			return fmt.Errorf("apply schema statement %q: %w", stmt, err)
		}
	}
	return nil
}

// applyMigrations cobre colunas adicionadas depois de uma base de dados já
// ter sido criada em disco — "CREATE TABLE IF NOT EXISTS" em applySchema
// não altera uma tabela que já existe. Sem gestor de migrações a sério
// (não há utilizadores em produção ainda para justificar um): cada
// ALTER TABLE é tentado sempre, e o erro "duplicate column name" (coluna
// já existe, de uma base de dados criada antes desta mudança) é ignorado
// — qualquer outro erro propaga.
func applyMigrations(conn *sql.DB) error {
	migrations := []string{
		`ALTER TABLE documents ADD COLUMN classification_pending INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range migrations {
		if _, err := conn.Exec(stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("apply migration %q: %w", stmt, err)
		}
	}
	return nil
}
