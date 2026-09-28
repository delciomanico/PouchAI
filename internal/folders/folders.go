// Package folders implementa o CRUD de pastas sobre a tabela `folders`.
package folders

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	AccentGradient string `json:"accentGradient"`
	IsShared       bool   `json:"isShared"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]Folder, error) {
	rows, err := s.db.Query(
		`SELECT id, name, accent_gradient, is_shared, created_at, updated_at
		 FROM folders ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	defer rows.Close()

	folders := []Folder{}
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.AccentGradient, &f.IsShared, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan folder: %w", err)
		}
		folders = append(folders, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	return folders, nil
}

func (s *Service) Create(name, accentGradient string, isShared bool) (Folder, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	f := Folder{
		ID:             uuid.NewString(),
		Name:           name,
		AccentGradient: accentGradient,
		IsShared:       isShared,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := s.db.Exec(
		`INSERT INTO folders (id, name, accent_gradient, is_shared, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, f.Name, f.AccentGradient, f.IsShared, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return Folder{}, fmt.Errorf("create folder: %w", err)
	}
	return f, nil
}

// Delete remove definitivamente a linha da pasta. Quem chama
// (backend.LocalBackend.DeleteFolder) tem de já ter apagado todos os
// documentos dessa pasta primeiro — documents.folder_id referencia
// folders(id) e o schema tem PRAGMA foreign_keys=ON.
func (s *Service) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM folders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete folder %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete folder %q: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("delete folder %q: not found", id)
	}
	return nil
}
