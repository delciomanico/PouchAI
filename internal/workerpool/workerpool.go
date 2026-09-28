// Package workerpool gere o pool elástico de processos ai-worker (C++):
// lança o primeiro no arranque, escala até um máximo quando a fila
// ultrapassa a capacidade actual, e deixa os workers encerrarem-se
// sozinhos quando ficam ociosos (ver ai-worker/src/main.cpp) — este
// package só nota que já não estão activos.
package workerpool

import (
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

type Pool struct {
	binPath      string
	dbPath       string
	callbackURL  string
	heartbeatURL string
	maxWorkers   int

	mu     sync.Mutex
	active int
}

func New(binPath, dbPath, callbackURL, heartbeatURL string, maxWorkers int) *Pool {
	return &Pool{
		binPath:      binPath,
		dbPath:       dbPath,
		callbackURL:  callbackURL,
		heartbeatURL: heartbeatURL,
		maxWorkers:   maxWorkers,
	}
}

// Start lança o primeiro worker, incondicionalmente (ver ROADMAP.md,
// Fase 3: "Go lança 1 worker no arranque").
func (p *Pool) Start() {
	p.spawn()
}

// EnsureCapacity lança mais um worker se a fila justificar — quando não
// há nenhum activo mas há trabalho, ou quando há mais jobs em espera do
// que workers a correr — até ao máximo configurado.
func (p *Pool) EnsureCapacity(queueDepth int) {
	p.mu.Lock()
	active := p.active
	p.mu.Unlock()

	if shouldSpawn(active, queueDepth, p.maxWorkers) {
		p.spawn()
	}
}

func shouldSpawn(active, queueDepth, maxWorkers int) bool {
	if active >= maxWorkers {
		return false
	}
	if active == 0 {
		return queueDepth > 0
	}
	return queueDepth > active
}

func (p *Pool) ActiveWorkers() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}

func (p *Pool) spawn() {
	p.mu.Lock()
	p.active++
	p.mu.Unlock()

	cmd := exec.Command(p.binPath,
		"--db", p.dbPath,
		"--callback", p.callbackURL,
		"--heartbeat-url", p.heartbeatURL,
		"--worker-id", uuid.NewString(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		defer func() {
			p.mu.Lock()
			p.active--
			p.mu.Unlock()
		}()
		if err := cmd.Run(); err != nil {
			log.Printf("workerpool: ai-worker terminou com erro: %v", err)
		}
	}()
}
