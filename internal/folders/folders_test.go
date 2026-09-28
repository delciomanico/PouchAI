package folders

import (
	"testing"

	"DocumentApp/internal/db"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn)
}

func TestCreateAndList(t *testing.T) {
	s := newTestService(t)

	if folders, err := s.List(); err != nil {
		t.Fatalf("List (empty): %v", err)
	} else if len(folders) != 0 {
		t.Fatalf("List (empty): got %d folders, want 0", len(folders))
	}

	created, err := s.Create("Contratos", "linear-gradient(a,b)", false)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create: expected a generated ID")
	}
	if created.Name != "Contratos" {
		t.Fatalf("Create: Name = %q, want %q", created.Name, "Contratos")
	}

	folders, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("List: got %d folders, want 1", len(folders))
	}
	if folders[0] != created {
		t.Fatalf("List: got %+v, want %+v", folders[0], created)
	}
}

func TestCreateSharedFolder(t *testing.T) {
	s := newTestService(t)

	created, err := s.Create("Equipa", "linear-gradient(c,d)", true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !created.IsShared {
		t.Fatal("Create: IsShared = false, want true")
	}

	folders, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(folders) != 1 || !folders[0].IsShared {
		t.Fatalf("List: got %+v, want a single shared folder", folders)
	}
}

func TestDelete(t *testing.T) {
	s := newTestService(t)
	folder, _ := s.Create("Contratos", "grad", false)

	if err := s.Delete(folder.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("List after Delete: got %+v, want none", list)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newTestService(t)
	if err := s.Delete("id-inexistente"); err == nil {
		t.Fatal("Delete: expected an error for an unknown id, got nil")
	}
}
