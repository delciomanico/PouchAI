// Package filestore grava o conteúdo dos documentos numa pasta gerida
// (<raiz>/<document-id>/<file-name>) em vez de depender do caminho
// original escolhido no diálogo nativo — ver ROADMAP.md, Fase 6. Isto
// importa sobretudo para o modo cliente/servidor: o processo que faz o
// OCR (ai-worker) só consegue ler ficheiros que estejam no disco do
// host, nunca no disco de um cliente remoto, por isso o conteúdo tem de
// ser copiado para aqui antes de entrar na fila de IA — mesmo quando o
// pedido de ingestão vem localmente (App desktop em modo "solo").
package filestore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Store struct {
	root string
}

func New(root string) *Store {
	return &Store{root: root}
}

// Save copia o conteúdo de r para <root>/<id>/<fileName>, criando as
// pastas necessárias. Devolve o caminho absoluto gravado e o número de
// bytes escritos (a fonte de verdade do tamanho — nunca confiar num
// size_bytes que o chamador diga, sobretudo vindo de um cliente remoto).
func (s *Store) Save(id, fileName string, r io.Reader) (path string, written int64, err error) {
	dir := filepath.Join(s.root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("filestore: criar pasta %q: %w", dir, err)
	}

	path = filepath.Join(dir, filepath.Base(fileName))
	f, err := os.Create(path)
	if err != nil {
		return "", 0, fmt.Errorf("filestore: criar ficheiro %q: %w", path, err)
	}
	defer f.Close()

	written, err = io.Copy(f, r)
	if err != nil {
		return "", 0, fmt.Errorf("filestore: gravar %q: %w", path, err)
	}
	return path, written, nil
}

// Delete remove a pasta inteira de um documento (<root>/<id>/…) — chamado
// quando um documento é apagado definitivamente (ver
// backend.LocalBackend.DeleteDocument). Não é erro apagar algo que já
// não existe: o objectivo final ("este documento não tem ficheiros
// geridos") já está cumprido.
func (s *Store) Delete(id string) error {
	dir := filepath.Join(s.root, id)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("filestore: apagar %q: %w", dir, err)
	}
	return nil
}
