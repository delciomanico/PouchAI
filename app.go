package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"DocumentApp/internal/apiserver"
	"DocumentApp/internal/backend"
	"DocumentApp/internal/documents"
	"DocumentApp/internal/folders"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct — desde a Fase 6, é só um adaptador fino entre o Wails e um
// backend.Backend: LocalBackend (modos "solo"/"host", dados neste
// processo) ou RemoteBackend (modo "cliente", dados noutro processo via
// HTTP — ver ROADMAP.md, Fase 6, e runmode.go para como o modo é
// escolhido). Todos os métodos ligados ao Wails delegam directamente,
// sem lógica própria — a lógica real vive em internal/backend.
type App struct {
	ctx     context.Context
	mode    runMode
	backend backend.Backend
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{mode: parseRunMode()}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if a.mode.kind == runModeClient {
		if a.mode.hostURL == "" || a.mode.token == "" {
			panic(fmt.Errorf("modo cliente exige --host e --token"))
		}
		a.backend = backend.NewRemote(a.mode.hostURL, a.mode.token)
		return
	}

	dbPath, err := backend.DefaultDBPath()
	if err != nil {
		panic(fmt.Errorf("resolve db path: %w", err))
	}
	filesRoot, err := backend.DefaultFilesRoot()
	if err != nil {
		panic(fmt.Errorf("resolve files root: %w", err))
	}

	workerBin, err := backend.DefaultAIWorkerPath()
	if err != nil {
		log.Printf("ai-worker indisponível, documentos vão ficar em 'uploading': %v", err)
	}
	searchBin, err := backend.DefaultSearchServicePath()
	if err != nil {
		log.Printf("search-service indisponível, pesquisa não vai funcionar: %v", err)
	}
	llmBin, err := backend.DefaultLLMServicePath()
	if err != nil {
		log.Printf("llm-service indisponível, resumo/classificação/chat reais não vão funcionar: %v", err)
	}
	chatModel, err := backend.DefaultAIChatModelPath()
	if err != nil {
		log.Printf("modelo de chat indisponível: %v", err)
	}
	embedModel, err := backend.DefaultAIEmbedModelPath()
	if err != nil {
		log.Printf("modelo de embeddings indisponível: %v", err)
	}

	lb, err := backend.NewLocal(backend.Config{
		DBPath:               dbPath,
		FilesRoot:            filesRoot,
		AIWorkerBinPath:      workerBin,
		SearchServiceBinPath: searchBin,
		LLMServiceBinPath:    llmBin,
		AIChatModelPath:      chatModel,
		AIEmbedModelPath:     embedModel,
		MaxWorkers:           backend.DefaultMaxWorkers(),
		Emit:                 wailsEmitter{ctx},
	})
	if err != nil {
		panic(fmt.Errorf("start local backend: %w", err))
	}
	a.backend = lb

	if a.mode.kind == runModeHost {
		token := a.mode.token
		if token == "" {
			token = backend.GenerateToken()
			log.Printf("modo host: nenhum --token dado, gerado automaticamente — dá isto aos clientes: %s", token)
		}
		srv := apiserver.New(lb, token)
		go func() {
			if err := srv.Serve(a.mode.listenAddr); err != nil {
				log.Printf("apiserver: parou: %v", err)
			}
		}()
		log.Printf("modo host: a servir a equipa em %s (token: %s)", a.mode.listenAddr, token)
	}
}

// shutdown is called when the app closes, to release the backend
// (base de dados, callback server, reaper, pool de ai-worker,
// search-service — ver backend.LocalBackend.Close).
func (a *App) shutdown(ctx context.Context) {
	if a.backend != nil {
		if err := a.backend.Close(); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}

// wailsEmitter adapta wailsruntime.EventsEmit à interface
// backend.Emitter.
type wailsEmitter struct {
	ctx context.Context
}

func (e wailsEmitter) Emit(eventName string, data ...interface{}) {
	wailsruntime.EventsEmit(e.ctx, eventName, data...)
}

// SelectDocumentFile abre o diálogo nativo de escolha de ficheiro e
// devolve o caminho escolhido, ou "" se o utilizador cancelar. Fica em
// app.go (não em internal/backend) porque é sempre um caminho no disco
// DESTE processo (o que corre a janela), mesmo em modo cliente.
func (a *App) SelectDocumentFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Selecionar documento",
	})
}

func (a *App) ListFolders() ([]folders.Folder, error) {
	return a.backend.ListFolders()
}

func (a *App) CreateFolder(name, accentGradient string, isShared bool) (folders.Folder, error) {
	return a.backend.CreateFolder(name, accentGradient, isShared)
}

func (a *App) DeleteFolder(id string) error {
	return a.backend.DeleteFolder(id)
}

func (a *App) ListDocuments(folderID string) ([]documents.Document, error) {
	return a.backend.ListDocuments(folderID)
}

func (a *App) ApproveDocument(id string) error {
	return a.backend.ApproveDocument(id)
}

func (a *App) RejectDocument(id string) error {
	return a.backend.RejectDocument(id)
}

func (a *App) UpdateDocumentClassification(id, documentType string, tags []string) error {
	return a.backend.UpdateDocumentClassification(id, documentType, tags)
}

// AppInfo é informação estática da instância, para o painel de
// Definições — nunca passa pelo Backend (Local/Remote): é sobre este
// processo em si (qual dos três modos de runmode.go está a correr),
// não sobre os dados.
type AppInfo struct {
	Name string `json:"name"`
	Mode string `json:"mode"`
}

func (a *App) GetAppInfo() AppInfo {
	return AppInfo{Name: "PouchIA", Mode: a.mode.kind}
}

func (a *App) DeleteDocument(id string) error {
	return a.backend.DeleteDocument(id)
}

func (a *App) Search(query string) (backend.SearchResponse, error) {
	return a.backend.Search(query)
}

func (a *App) AskAssistant(question string) (backend.AssistantAnswer, error) {
	return a.backend.AskAssistant(question)
}

// IngestFile lê o ficheiro já escolhido via SelectDocumentFile (um
// caminho no disco deste processo) e entrega os bytes ao backend — em
// modo cliente isto faz um upload real para o host por HTTP; em modo
// solo/host é uma cópia local para a pasta gerida (ver
// internal/filestore). Nenhum dos dois lados de IngestFile sabe qual dos
// dois está a acontecer.
func (a *App) IngestFile(folderID, localPath string) (documents.Document, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return documents.Document{}, fmt.Errorf("abrir %q: %w", localPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return documents.Document{}, fmt.Errorf("stat %q: %w", localPath, err)
	}

	return a.backend.IngestFile(folderID, info.Name(), info.Size(), f)
}

// DownloadDocumentFile pede o conteúdo do documento ao backend (local ou
// remoto — ver internal/backend.Backend.DocumentFile, o
// /documents/{id}/file da Fase 6) e deixa o utilizador escolher onde
// gravar, via diálogo nativo. Devolve "" se o utilizador cancelar.
func (a *App) DownloadDocumentFile(id string) (string, error) {
	content, fileName, err := a.backend.DocumentFile(id)
	if err != nil {
		return "", err
	}
	defer content.Close()

	dest, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Guardar documento",
		DefaultFilename: fileName,
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil // utilizador cancelou
	}

	out, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("criar %q: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, content); err != nil {
		return "", fmt.Errorf("gravar %q: %w", dest, err)
	}
	return dest, nil
}

// OpenDocumentFile mostra o conteúdo do documento na aplicação
// predefinida do sistema operativo (visualizador de PDF, imagens, etc.)
// — funciona tanto em modo local como em modo cliente: pede sempre o
// conteúdo ao backend (ver Backend.DocumentFile), nunca assume que o
// ficheiro já está neste disco, e grava uma cópia temporária antes de
// pedir ao SO para a abrir.
func (a *App) OpenDocumentFile(id string) error {
	content, fileName, err := a.backend.DocumentFile(id)
	if err != nil {
		return err
	}
	defer content.Close()

	tmpDir := filepath.Join(os.TempDir(), "DocumentApp-preview")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("criar pasta temporária: %w", err)
	}
	tmpPath := filepath.Join(tmpDir, id+"-"+fileName)

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("criar %q: %w", tmpPath, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, content); err != nil {
		return fmt.Errorf("gravar %q: %w", tmpPath, err)
	}
	out.Close()

	return openWithDefaultApp(tmpPath)
}

// DocumentPreview é o suficiente para o React desenhar uma
// pré-visualização inline (imagem ou PDF, os dois tipos que qualquer
// webview Chromium consegue mostrar sozinha, sem bibliotecas extra).
// Previewable=false para tudo o resto (ex. .docx) — a UI mostra nesse
// caso "abrir no leitor do sistema" / "descarregar" em vez de tentar
// desenhar algo.
type DocumentPreview struct {
	FileName    string `json:"fileName"`
	MimeType    string `json:"mimeType"`
	Previewable bool   `json:"previewable"`
	DataURL     string `json:"dataUrl"` // "" quando Previewable é false
}

// GetDocumentPreview devolve o conteúdo do documento como data: URL,
// pronto a pôr directamente num <img>/<iframe> — evita ter de expor mais
// um servidor HTTP só para a pré-visualização, reaproveitando a mesma
// ponte IPC do Wails que todos os outros métodos já usam. Para
// ficheiros grandes isto tem um custo (base64 + round-trip JSON), mas é
// o suficiente para os documentos típicos desta app.
func (a *App) GetDocumentPreview(id string) (DocumentPreview, error) {
	content, fileName, err := a.backend.DocumentFile(id)
	if err != nil {
		return DocumentPreview{}, err
	}
	defer content.Close()

	mimeType := mime.TypeByExtension(filepath.Ext(fileName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if !isPreviewableMime(mimeType) {
		return DocumentPreview{FileName: fileName, MimeType: mimeType, Previewable: false}, nil
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return DocumentPreview{}, fmt.Errorf("ler conteúdo do documento %q: %w", id, err)
	}

	return DocumentPreview{
		FileName:    fileName,
		MimeType:    mimeType,
		Previewable: true,
		DataURL:     "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func isPreviewableMime(mimeType string) bool {
	base, _, _ := strings.Cut(mimeType, ";")
	return strings.HasPrefix(base, "image/") || base == "application/pdf"
}

// openWithDefaultApp pede ao SO para abrir path com a aplicação
// predefinida para o seu tipo — equivalente a fazer duplo-clique no
// ficheiro no explorador de ficheiros.
func openWithDefaultApp(path string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("abrir %q com a aplicação predefinida: %w", path, err)
	}
	return nil
}
