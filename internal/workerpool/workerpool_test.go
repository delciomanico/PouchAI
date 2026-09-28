package workerpool

import (
	"os"
	"testing"
	"time"
)

func TestShouldSpawn(t *testing.T) {
	cases := []struct {
		name               string
		active, queue, max int
		want               bool
	}{
		{"no workers, no work", 0, 0, 4, false},
		{"no workers, work waiting", 0, 1, 4, true},
		{"workers keep up with queue", 2, 2, 4, false},
		{"queue exceeds workers, room to grow", 2, 3, 4, true},
		{"already at max", 4, 10, 4, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := shouldSpawn(c.active, c.queue, c.max)
			if got != c.want {
				t.Errorf("shouldSpawn(%d, %d, %d) = %v, want %v", c.active, c.queue, c.max, got, c.want)
			}
		})
	}
}

// newFakePool aponta o pool para o próprio binário de testes: chamado
// com flags que não reconhece (--db, --callback, ...), sai de imediato
// com erro — real o suficiente para exercitar o ciclo
// spawn/Run/decremento de `active`, sem precisar do ai-worker compilado.
func newFakePool(maxWorkers int) *Pool {
	return &Pool{
		binPath:     os.Args[0],
		dbPath:      "unused.db",
		callbackURL: "http://127.0.0.1:0/unused",
		maxWorkers:  maxWorkers,
	}
}

func TestStartSpawnsOneWorker(t *testing.T) {
	p := newFakePool(4)
	p.Start()

	if got := p.ActiveWorkers(); got != 1 {
		t.Fatalf("ActiveWorkers right after Start = %d, want 1", got)
	}

	waitForActive(t, p, 0)
}

func TestEnsureCapacityRespectsMax(t *testing.T) {
	p := newFakePool(2)
	p.Start()
	waitForActive(t, p, 0) // deixa o primeiro processo terminar antes de continuar

	p.mu.Lock()
	p.active = 2 // simula 2 workers já a correr
	p.mu.Unlock()

	p.EnsureCapacity(100) // fila enorme, mas já estamos no máximo
	if got := p.ActiveWorkers(); got != 2 {
		t.Fatalf("ActiveWorkers after EnsureCapacity at max = %d, want 2 (should not exceed max)", got)
	}
}

func waitForActive(t *testing.T, p *Pool, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if p.ActiveWorkers() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("ActiveWorkers did not reach %d within 5s (got %d)", want, p.ActiveWorkers())
}
