// cmd/server é o "build servidor" da Fase 6 do ROADMAP.md — um binário
// separado do desktop (app.go/main.go, que usa Wails), sem qualquer
// dependência de janela/webview: só o núcleo Go + a API HTTP
// (internal/apiserver) sobre um backend.LocalBackend. Pensado para
// correr numa máquina sem ambiente gráfico (ex. um servidor caseiro),
// sempre em papel de host — não existe um "modo cliente" para este
// binário, porque sem janela não há nada para mostrar dados de outro
// host.
//
// "Dois builds do Go" (ver ROADMAP.md): este é o segundo — compila-se
// com `go build ./cmd/server`, nunca com `wails build`, e por isso nunca
// puxa a dependência do WebView2/Wails para dentro do binário.
package main

import (
	"flag"
	"log"

	"DocumentApp/internal/apiserver"
	"DocumentApp/internal/backend"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:8790", "endereço a escutar")
	token := flag.String("token", "", "token partilhado da equipa (gera-se um se vazio)")
	dbPath := flag.String("db", "", "caminho da base de dados SQLite (omisso: pasta de dados por omissão)")
	filesRoot := flag.String("files", "", "pasta gerida de ficheiros (omissa: pasta de dados por omissão)")
	aiWorkerBin := flag.String("ai-worker", "", "caminho do binário ai-worker (omisso: tenta o caminho de desenvolvimento)")
	searchServiceBin := flag.String("search-service", "", "caminho do binário search-service (omisso: tenta o caminho de desenvolvimento)")
	llmServiceBin := flag.String("llm-service", "", "caminho do binário llm-service (omisso: tenta o caminho de desenvolvimento)")
	aiChatModel := flag.String("ai-chat-model", "", "caminho do modelo de chat .gguf (omisso: tenta a pasta de dados por omissão)")
	aiEmbedModel := flag.String("ai-embed-model", "", "caminho do modelo de embeddings .gguf (omisso: tenta a pasta de dados por omissão)")
	maxWorkers := flag.Int("max-workers", 0, "máximo de ai-worker em simultâneo (omisso: min(NumCPU,4))")
	flag.Parse()

	db := *dbPath
	if db == "" {
		p, err := backend.DefaultDBPath()
		if err != nil {
			log.Fatalf("resolve db path: %v", err)
		}
		db = p
	}

	files := *filesRoot
	if files == "" {
		p, err := backend.DefaultFilesRoot()
		if err != nil {
			log.Fatalf("resolve files root: %v", err)
		}
		files = p
	}

	worker := *aiWorkerBin
	if worker == "" {
		if p, err := backend.DefaultAIWorkerPath(); err != nil {
			log.Printf("ai-worker indisponível, documentos vão ficar em 'uploading': %v", err)
		} else {
			worker = p
		}
	}

	search := *searchServiceBin
	if search == "" {
		if p, err := backend.DefaultSearchServicePath(); err != nil {
			log.Printf("search-service indisponível, pesquisa não vai funcionar: %v", err)
		} else {
			search = p
		}
	}

	llm := *llmServiceBin
	if llm == "" {
		if p, err := backend.DefaultLLMServicePath(); err != nil {
			log.Printf("llm-service indisponível, resumo/classificação/chat reais não vão funcionar: %v", err)
		} else {
			llm = p
		}
	}

	chatModel := *aiChatModel
	if chatModel == "" {
		if p, err := backend.DefaultAIChatModelPath(); err != nil {
			log.Printf("modelo de chat indisponível: %v", err)
		} else {
			chatModel = p
		}
	}

	embedModel := *aiEmbedModel
	if embedModel == "" {
		if p, err := backend.DefaultAIEmbedModelPath(); err != nil {
			log.Printf("modelo de embeddings indisponível: %v", err)
		} else {
			embedModel = p
		}
	}

	workers := *maxWorkers
	if workers <= 0 {
		workers = backend.DefaultMaxWorkers()
	}

	lb, err := backend.NewLocal(backend.Config{
		DBPath:               db,
		FilesRoot:            files,
		AIWorkerBinPath:      worker,
		SearchServiceBinPath: search,
		LLMServiceBinPath:    llm,
		AIChatModelPath:      chatModel,
		AIEmbedModelPath:     embedModel,
		MaxWorkers:           workers,
	})
	if err != nil {
		log.Fatalf("start local backend: %v", err)
	}
	defer lb.Close()

	t := *token
	if t == "" {
		t = backend.GenerateToken()
		log.Printf("nenhum --token dado, gerado automaticamente — dá isto aos clientes: %s", t)
	}

	log.Printf("db: %s", db)
	log.Printf("ficheiros: %s", files)
	srv := apiserver.New(lb, t)
	if err := srv.Serve(*listen); err != nil {
		log.Fatalf("apiserver: %v", err)
	}
}
