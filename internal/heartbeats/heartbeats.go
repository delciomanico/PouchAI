// Package heartbeats mantém, em memória, quando cada ai-worker foi
// visto pela última vez — usado para detetar workers mortos mais
// depressa do que a lease de 90s dos jobs (ver ROADMAP.md, Fase 4).
package heartbeats

import (
	"sync"
	"time"
)

type Tracker struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
}

func NewTracker() *Tracker {
	return &Tracker{lastSeen: make(map[string]time.Time)}
}

// Touch regista que workerID está vivo agora.
func (t *Tracker) Touch(workerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastSeen[workerID] = time.Now()
}

// Dead devolve os workers cujo último heartbeat é mais antigo que
// olderThan, e esquece-os — cada worker morto só é reportado uma vez,
// para quem chama não libertar os mesmos jobs repetidamente.
func (t *Tracker) Dead(olderThan time.Duration) []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	var dead []string
	for id, seen := range t.lastSeen {
		if seen.Before(cutoff) {
			dead = append(dead, id)
			delete(t.lastSeen, id)
		}
	}
	return dead
}
