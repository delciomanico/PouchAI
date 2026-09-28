# Build orquestrado dos quatro subprojectos (Go, frontend React, e os dois
# serviços C++). O alvo `schema` é o único sítio que copia
# shared/schema.sql para dentro do módulo Go — nunca editar a cópia em
# internal/db/schema.sql à mão, editar sempre o ficheiro em shared/.

GO_SCHEMA_DEST := internal/db/schema.sql

.PHONY: all schema build build-go build-server build-frontend build-ai-worker build-search-service build-llm-service dev clean

all: build

schema:
	mkdir -p $(dir $(GO_SCHEMA_DEST))
	cp shared/schema.sql $(GO_SCHEMA_DEST)

build: schema build-frontend build-go build-server build-ai-worker build-search-service build-llm-service

# build-go: o build "desktop" da Fase 6 (janela Wails) — via `go build`
# aqui só para verificar que compila; o instalável de verdade usa
# `wails build`.
build-go: schema
	go build ./...

# build-server: o build "servidor" da Fase 6 — binário separado, sem
# nenhuma dependência de Wails/WebView2, para correr sem ambiente
# gráfico (ver cmd/server/main.go).
build-server: schema
	go build -o build/bin/server ./cmd/server

build-frontend:
	cd frontend && npm install && npm run build

build-ai-worker:
	cmake -B ai-worker/build -S ai-worker
	cmake --build ai-worker/build

build-search-service:
	cmake -B search-service/build -S search-service
	cmake --build search-service/build

# build-llm-service: compila o llama.cpp vendored (shared/vendor/llama.cpp)
# como parte do alvo — é o build mais lento dos três (minutos, não
# segundos), por ser uma árvore de código bastante maior que sqlite3/
# httplib; não é sinal de o build ter encravado.
build-llm-service:
	cmake -B llm-service/build -S llm-service
	cmake --build llm-service/build

dev: schema
	wails dev

clean:
	rm -rf build ai-worker/build search-service/build llm-service/build frontend/dist $(GO_SCHEMA_DEST)
