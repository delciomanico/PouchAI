package filestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveWritesUnderIDSubfolder(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	path, written, err := s.Save("doc-1", "contrato.pdf", strings.NewReader("conteudo de teste"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	wantPath := filepath.Join(root, "doc-1", "contrato.pdf")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}
	if written != int64(len("conteudo de teste")) {
		t.Errorf("written = %d, want %d", written, len("conteudo de teste"))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "conteudo de teste" {
		t.Errorf("file content = %q, want %q", string(got), "conteudo de teste")
	}
}

func TestSaveStripsDirectoryFromFileName(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	path, _, err := s.Save("doc-2", "../../etc/passwd", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	wantPath := filepath.Join(root, "doc-2", "passwd")
	if path != wantPath {
		t.Errorf("path = %q, want %q (directory components in fileName must be stripped)", path, wantPath)
	}
}

func TestDeleteRemovesDocumentFolder(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	path, _, err := s.Save("doc-1", "contrato.pdf", strings.NewReader("x"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Delete("doc-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) after Delete: err = %v, want IsNotExist", path, err)
	}
	if _, err := os.Stat(filepath.Join(root, "doc-1")); !os.IsNotExist(err) {
		t.Fatalf("the whole doc-1 folder should be gone, err = %v", err)
	}
}

func TestDeleteNonExistentIsNotAnError(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if err := s.Delete("never-existed"); err != nil {
		t.Fatalf("Delete of a never-existing document: %v, want nil", err)
	}
}

func TestSaveTwoDocumentsDoNotCollide(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	p1, _, err := s.Save("doc-1", "mesmo-nome.pdf", strings.NewReader("um"))
	if err != nil {
		t.Fatalf("Save doc-1: %v", err)
	}
	p2, _, err := s.Save("doc-2", "mesmo-nome.pdf", strings.NewReader("dois"))
	if err != nil {
		t.Fatalf("Save doc-2: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("two different documents with the same file name collided at %q", p1)
	}

	got1, _ := os.ReadFile(p1)
	got2, _ := os.ReadFile(p2)
	if string(got1) != "um" || string(got2) != "dois" {
		t.Fatalf("content mixed up: p1=%q p2=%q", got1, got2)
	}
}
