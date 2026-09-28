# Gestão Documental — Roadmap de Desenvolvimento

## Regra de trabalho (ler antes de começar)

**Uma funcionalidade de cada vez, até 100%.** Não avances para a fase seguinte
enquanto a "Definição de 100%" da fase atual não estiver toda cumprida —
incluindo os testes. Não deixes código morto, TODOs por implementar, nem
"vou voltar aqui depois" — se algo ficar por fazer, marca a caixa como não
concluída e continua nessa fase.

Cada fase é independente e testável sozinha antes de a seguinte começar a
depender dela. Se precisares de adiantar trabalho de uma fase posterior
para desbloquear a atual, pára e sinaliza — normalmente é sinal de que a
ordem das fases precisa de ajuste, não de saltar a regra.

---

## Visão geral

App desktop nativa de gestão documental com IA, local-first, para equipas.

- **React** — interface (webview Wails)
- **Go** — núcleo: dados, orquestração, API HTTP, gestão dos workers
- **C++** — só IA: OCR/classificação (pool de processos) e pesquisa semântica
  (processo único, sempre ativo)
- **SQLite** — única fonte de verdade local

Dois caminhos de IA, propositadamente separados: OCR/classificação é
assíncrono (fila); pesquisa é síncrona (nunca passa pela fila).

---

## Fase 0 — Fundação do repositório

- [ ] Estrutura de pastas: `/` (Go), `/frontend` (React), `/ai-worker` (C++
      OCR/classificação), `/search-service` (C++ pesquisa), `/shared`
      (schema SQLite, contratos)
- [ ] `/shared/schema.sql` com as tabelas `folders`, `documents`, `ai_jobs`
      (ver campos na secção "Esquema SQLite" abaixo)
- [ ] Script de build (`Makefile` ou equivalente) que copia
      `/shared/schema.sql` para dentro do módulo Go antes de compilar —
      nunca duas cópias editadas à mão
- [ ] `go.mod` inicial, `wails.json`, `frontend/package.json` (Vite + React)
- [ ] `README.md` com instruções de build para as três partes

**Definição de 100%:** `wails dev` arranca uma janela vazia sem erros; os
três subprojectos compilam isoladamente (`go build`, `npm run build`,
`cmake --build` nos dois C++).

---

## Fase 1 — Go core: dados e CRUD, sem IA ainda

- [ ] `internal/db`: abre o SQLite, aplica o schema
- [ ] `internal/folders`: `List`, `Create`
- [ ] `internal/documents`: `ListByFolder`, `Get`, `Approve`, `Reject`
- [ ] `app.go`: métodos ligados ao Wails (`ListFolders`, `CreateFolder`,
      `ListDocuments`, `ApproveDocument`, `RejectDocument`)
- [ ] Sem fila de IA ainda — `IngestFile` grava o documento direto como
      `ready`, só para validar o caminho de dados

**Definição de 100%:** consegues criar uma pasta, listar pastas, "ingerir"
um documento e vê-lo listado — tudo via chamadas Go, testado com testes
unitários em cada serviço, sem React ainda.

---

## Fase 2 — React: casca funcional

- [ ] Lista de pastas (chama `ListFolders`)
- [ ] Lista de documentos de uma pasta (chama `ListDocuments`)
- [ ] Criar pasta, aprovar/rejeitar documento
- [ ] Tokens de tema (`theme.css`) — cores, tipografia, espaçamento

**Definição de 100%:** consegues fazer tudo o que a Fase 1 valida por
linha de comandos, agora a partir da janela da app.

---

## Fase 3 — Fila de IA (Go) + pool de OCR/classificação (C++)

- [ ] `internal/jobs`: `Enqueue` — cria documento (`status=uploading`) +
      job (`status=queued`) numa transação
- [ ] `ai-worker` em C++: liga ao SQLite, `claimNext` (UPDATE atómico
      `WHERE status='queued'`), processa (placeholder de OCR/classificação
      por agora), grava resultado, `status=pending_review`
- [ ] Notificação ativa: o worker faz `POST` a um endpoint HTTP interno do
      Go ao terminar — sem polling
- [ ] Go: endpoint `/internal/jobs/callback` + emite evento Wails
      (`document:updated`) para o React actualizar em tempo real
- [ ] Pool elástico: Go lança 1 worker no arranque; escala até um máximo
      quando a fila ultrapassa um limiar; encerra workers ociosos

**Definição de 100%:** arrastas um ficheiro, vês `uploading` →
`processing` → `pending_review` a mudar sozinho na interface, sem
recarregar nada. Testado com múltiplos ficheiros em simultâneo.

---

## Fase 4 — Fiabilidade do pool

- [ ] Lease de 90s: jobs `claimed` há mais tempo que isso voltam a
      `queued` automaticamente
- [ ] Heartbeat dos workers para o Go; um worker sem heartbeat é
      considerado morto, os seus jobs são libertados
- [ ] Contrapressão: acima de N jobs pendentes, `IngestFile` recusa novas
      entradas com erro claro (mostrado no React)
- [ ] `busy_timeout` configurado em todas as ligações SQLite (Go e C++)

**Definição de 100%:** matar o processo `ai-worker` a meio de um job (`kill
-9`) resulta no job de volta à fila dentro de 90s, sem intervenção manual.
Testado.

---

## Fase 5 — Serviço de pesquisa IA

- [ ] `search-service` em C++: processo único, sempre ativo, modelo
      (embeddings) carregado em memória
- [ ] Ao concluir OCR/classificação (Fase 3), o `ai-worker` calcula também
      o resumo + embedding do documento
- [ ] Go chama o `search-service` por HTTP loopback, síncrono
- [ ] Distinção pergunta vs busca: pergunta directa devolve resposta +
      resumo; busca genérica devolve lista ordenada por relevância
- [ ] React: campo de pesquisa, mostra resposta ou lista consoante o tipo

**Definição de 100%:** pesquisar por um assunto devolve o documento certo
com o resumo, em menos de 1s, mesmo com o pool de OCR ocupado a processar
outros ficheiros.

---

## Fase 6 — Modo equipa: host, cliente, dois builds do Go

- [ ] `Backend` interface em Go: `LocalBackend` (SQLite direto) e
      `RemoteBackend` (cliente HTTP)
- [ ] Flag de arranque: build **desktop** (janela Wails) vs build
      **servidor** (`--headless`, sem janela)
- [ ] Servidor HTTP do Go: bind em `127.0.0.1` (standalone) ou também
      `0.0.0.0` (modo host)
- [ ] Autenticação por token para clientes remotos
- [ ] Endpoint `/documents/{id}/file` para servir o conteúdo do ficheiro
      (local ou remoto)
- [ ] Pasta gerida de ficheiros (`.../GestaoDocumental/files/<id>`) em vez
      de depender do caminho original

**Definição de 100%:** duas instâncias da app (uma como host, outra como
cliente ligado por IP) partilham a mesma biblioteca — pastas, documentos,
pesquisa — tudo a funcionar de ambos os lados.

---

## Fase 7 — Híbrido: sincronização opcional, controlada pelo admin

- [ ] Definição "sincronização com a nuvem" nas Definições, desligada por
      omissão
- [ ] `sync_outbox`: toda a escrita local relevante entra numa fila de
      saída
- [ ] Rotina em segundo plano envia a caixa de saída quando há ligação;
      nunca bloqueia a interface
- [ ] Resolução de conflitos: último a escrever ganha, por `updated_at`;
      tabela `conflicts` regista o que foi substituído (não silencioso)
- [ ] Apagar usa marcador ("tombstone"), nunca remove a linha localmente
      antes de sincronizar
- [ ] Nuvem: metadados + resumo sempre sincronizados; conteúdo do ficheiro
      sobe em segundo plano e só desce localmente quando aberto (lazy)
- [ ] Armazenamento de objectos (ex. S3-compatível) em vez de disco
      próprio a gerir

**Definição de 100%:** com a sincronização desligada, a app funciona
100% sem rede. Ligada, um documento criado num dispositivo aparece
(metadados + resumo) noutro sem o ficheiro estar lá, e abre com download
sob pedido.

---

## Fase 8 — Empacotamento: instalador único

- [ ] Pipeline de build por plataforma: compila `ai-worker` e
      `search-service` com CMake → copia para `bin/` no projecto Go →
      `go:embed` → `wails build`
- [ ] Extracção dos binários embebidos para a pasta de dados da app no
      primeiro arranque, com permissão de execução
- [ ] Instalador Windows (Inno Setup/NSIS), `.dmg` no macOS, AppImage no
      Linux
- [ ] Assinatura de código (Windows) e assinatura + notarização (macOS)
- [ ] GitHub Actions: build matrix para as três plataformas

**Definição de 100%:** alguém sem nada instalado descarrega um ficheiro
único, corre o instalador do seu SO, e a app abre — sem saber que há Go,
React e C++ lá dentro.

---

## Fase 9 — Polimento e testes de integração

- [ ] Ambiente de teste (docker-compose ou equivalente): Go + worker +
      SQLite temporário
- [ ] Cenários: matar worker a meio de um job, fila cheia, cliente a
      perder ligação ao host, conflito de sincronização
- [ ] Logging estruturado com ID de pedido a atravessar Go → worker →
      resposta
- [ ] Verificação de actualizações no arranque

**Definição de 100%:** todos os cenários da lista de cima têm um teste
automatizado que passa, correndo em CI.

---

## Esquema SQLite (referência — a fonte real é `/shared/schema.sql`)

```sql
folders(id, name, accent_gradient, is_shared, created_at, updated_at)

documents(id, file_name, folder_id, document_type, author, size_bytes,
          local_path, status, ocr_confidence, ocr_excerpt, tags_json,
          created_at, updated_at)
-- status: uploading | processing | pending_review | ready | failed

ai_jobs(id, document_id, job_type, status, claimed_by, result_json,
        error, created_at, updated_at)
-- status: queued | claimed | done | failed
```
