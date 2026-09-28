// Package llmservice gere o processo C++ llm-service (motor de IA local,
// ver plano em unified-popping-sunset.md): arranca-o uma vez (processo
// único, sempre ativo, com dois modelos GGUF carregados em memória — nunca
// elástico, ao contrário do pool do ai-worker) e fala com ele por HTTP
// loopback, síncrono. Mesmo molde de internal/searchservice.
//
// Protocolo interno em texto simples, não JSON — só o Go e o llm-service
// falam este protocolo, os dois escritos por nós:
//
//	GET  /internal/health -> "ok" (só depois dos dois modelos carregados)
//	POST /summarize -> corpo "<nome-do-ficheiro>\n---\n<texto extraído>"
//	                    resposta: resumo gerado em 1-2 frases
//	POST /classify  -> corpo "<nome-do-ficheiro>\n---\n<texto extraído>"
//	                    resposta: linha 1 = tipo de documento
//	                              linha 2 = tags separadas por vírgula
//	POST /chat      -> corpo "<pergunta>\n---CONTEXT---\n<excertos>"
//	                    resposta: texto gerado
//	POST /embed     -> corpo texto livre
//	                    resposta: vector normalizado, floats separados
//	                    por vírgula (ver internal/searchservice, que
//	                    consome isto para indexar/pesquisar por
//	                    significado em vez de correspondência exacta de
//	                    palavras)
//
// Ver llm-service/src/main.cpp para o outro lado.
package llmservice

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	binPath        string
	chatModelPath  string
	embedModelPath string
	baseURL        string
	cmd            *exec.Cmd
	client         *http.Client
}

func New(binPath, chatModelPath, embedModelPath string) *Manager {
	return &Manager{
		binPath:        binPath,
		chatModelPath:  chatModelPath,
		embedModelPath: embedModelPath,
		// As gerações de texto podem demorar bem mais que uma pesquisa no
		// search-service — CPU pura, sem GPU, ver plano. O próprio
		// llm-service agora auto-limita cada geração a 90s de parede
		// (kGenerationDeadline em main.cpp — sem isto, um documento real
		// cujo texto não leve o modelo a parar sozinho podia correr
		// vários MINUTOS sem nunca responder, medido em primeira mão);
		// 120s aqui dá margem para esse limite mais o processamento do
		// prompt em si, sem esconder um pedido genuinamente preso
		// (processo morto) durante minutos a fio.
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Start escolhe um porto livre em 127.0.0.1, lança o llm-service já
// apontado para esse porto e para os dois modelos, e espera até
// /internal/health responder. O deadline é bem maior que o do
// search-service: carregar dois modelos GGUF é mais lento que arrancar o
// índice em memória do search-service.
func (m *Manager) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("llmservice: escolher porto: %w", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	m.baseURL = "http://" + addr

	cmd := exec.Command(m.binPath,
		"--listen", addr,
		"--chat-model", m.chatModelPath,
		"--embed-model", m.embedModelPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("llmservice: arrancar %q: %w", m.binPath, err)
	}
	m.cmd = cmd

	// 60s chegava para os modelos de teste desta funcionalidade, mas
	// medido em primeira mão com os modelos de produção (Qwen2.5-1.5B +
	// multilingual-e5-small, ~1.25GB ao todo) num arranque com o disco
	// "frio": carregar os dois modelos + pré-calcular os embeddings das
	// categorias/tags candidatas levou 65s, matando o processo mesmo
	// antes de ficar pronto. 150s dá margem sem esconder um arranque
	// verdadeiramente preso para sempre.
	deadline := time.Now().Add(150 * time.Second)
	healthClient := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := healthClient.Get(m.baseURL + "/internal/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("health devolveu estado %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}

	m.Stop()
	return fmt.Errorf("llmservice: não respondeu em %s a tempo: %w", m.baseURL, lastErr)
}

// Stop termina o processo do llm-service, se estiver a correr.
func (m *Manager) Stop() {
	if m.cmd == nil || m.cmd.Process == nil {
		return
	}
	_ = m.cmd.Process.Kill()
	_ = m.cmd.Wait()
}

func (m *Manager) post(path, body string) (string, error) {
	resp, err := m.client.Post(m.baseURL+path, "text/plain; charset=utf-8", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("llmservice: %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llmservice: %s: ler resposta: %w", path, err)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("llmservice: %s devolveu estado %d: %s", path, resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}

// Summarize pede ao modelo de chat um resumo em 1-2 frases do texto
// extraído de um documento.
func (m *Manager) Summarize(fileName, text string) (string, error) {
	body := fileName + "\n---\n" + text
	resp, err := m.post("/summarize", body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

// Classify pede ao modelo de embeddings o tipo de documento e as tags
// (zero-shot, por comparação de cosseno contra listas fixas — ver plano e
// llm-service/src/main.cpp).
func (m *Manager) Classify(fileName, text string) (documentType string, tags []string, err error) {
	body := fileName + "\n---\n" + text
	resp, err := m.post("/classify", body)
	if err != nil {
		return "", nil, err
	}

	lines := strings.SplitN(resp, "\n", 2)
	documentType = strings.TrimSpace(lines[0])
	if len(lines) > 1 {
		tagsLine := strings.TrimSpace(lines[1])
		if tagsLine != "" {
			for _, tag := range strings.Split(tagsLine, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
	}
	return documentType, tags, nil
}

// Embed pede ao modelo de embeddings o vector semântico (normalizado) de
// um texto livre — usado por internal/searchservice para indexar e
// pesquisar documentos por significado, reaproveitando o mesmo modelo já
// usado para Classify em vez de um segundo motor mais fraco noutro
// processo (ver comentário em search-service/src/main.cpp).
func (m *Manager) Embed(text string) ([]float32, error) {
	resp, err := m.post("/embed", text)
	if err != nil {
		return nil, err
	}
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return nil, nil
	}
	parts := strings.Split(resp, ",")
	vec := make([]float32, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("llmservice: embed: componente do vector inválida %q: %w", p, err)
		}
		vec = append(vec, float32(f))
	}
	return vec, nil
}

// Chat pede ao modelo de chat uma resposta gerada em linguagem natural a
// uma pergunta, com base nos excertos de contexto dados (RAG — o Go é
// quem já recuperou esses excertos via search-service + SQLite).
func (m *Manager) Chat(question, context string) (string, error) {
	body := question + "\n---CONTEXT---\n" + context
	resp, err := m.post("/chat", body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}
