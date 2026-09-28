// Package backend abstrai onde vivem os dados da app: LocalBackend fala
// directamente com o SQLite deste processo (e com o ai-worker/
// search-service locais); RemoteBackend fala por HTTP com outro
// processo — o "host" — que por sua vez usa um LocalBackend próprio. As
// duas implementações partilham exactamente a mesma interface, por isso
// app.go (o lado Wails) e cmd/server (o binário sem janela) nunca
// precisam de saber com qual estão a falar. Ver ROADMAP.md, Fase 6.
package backend

import (
	"io"

	"DocumentApp/internal/documents"
	"DocumentApp/internal/folders"
)

// SearchHit e SearchResponse eram tipos definidos em app.go (Fase 5) —
// mudaram-se para aqui porque agora são a forma de resultado tanto do
// Backend em si como da API HTTP entre host e cliente (RemoteBackend
// descodifica exactamente este JSON).
type SearchHit struct {
	DocumentID   string  `json:"documentId"`
	FolderID     string  `json:"folderId"`
	FileName     string  `json:"fileName"`
	DocumentType string  `json:"documentType"`
	Summary      string  `json:"summary"`
	Score        float64 `json:"score"`
}

type SearchResponse struct {
	Type    string      `json:"type"`
	Results []SearchHit `json:"results"`
}

// AssistantAnswer é a resposta de AskAssistant: texto gerado pelo modelo
// de chat local (RAG sobre os documentos já indexados) mais as fontes
// usadas, para a UI poder citá-las/navegar até lá — mesmo padrão de
// SearchHit já usado pela pesquisa.
type AssistantAnswer struct {
	Answer  string      `json:"answer"`
	Sources []SearchHit `json:"sources"`
}

// Backend é o contrato usado pelos métodos ligados ao Wails (app.go) e
// pela API HTTP do host (internal/apiserver) — as mesmas operações,
// quer os dados estejam neste processo (LocalBackend) quer noutro
// (RemoteBackend).
type Backend interface {
	ListFolders() ([]folders.Folder, error)
	CreateFolder(name, accentGradient string, isShared bool) (folders.Folder, error)
	DeleteFolder(id string) error
	ListDocuments(folderID string) ([]documents.Document, error)
	ApproveDocument(id string) error
	RejectDocument(id string) error
	DeleteDocument(id string) error

	// UpdateDocumentClassification aplica uma correção manual do
	// utilizador ao tipo de documento e às tags sugeridas pelo modelo,
	// durante a revisão (antes de aprovar) — ver IngestFlowPage.tsx.
	// Nunca mexe no resumo nem no estado de enriquecimento pendente.
	UpdateDocumentClassification(id, documentType string, tags []string) error
	Search(query string) (SearchResponse, error)

	// AskAssistant faz RAG simples sobre os documentos já indexados:
	// pesquisa no search-service, recupera os documentos das melhores
	// correspondências, monta contexto, e pede ao llm-service uma
	// resposta gerada em linguagem natural. Corre sempre no LocalBackend
	// do anfitrião — um RemoteBackend também o expõe (mesma paridade
	// Local/Remote da Fase 6), mas nunca carrega o seu próprio modelo.
	AskAssistant(question string) (AssistantAnswer, error)

	// IngestFile lê todo o conteúdo de r e entra na fila de IA — quem
	// chama (app.go) já não precisa de saber se o destino final é uma
	// pasta gerida local ou um upload HTTP para o host. size é só uma
	// indicação (ex. Content-Length de um upload); o tamanho real
	// gravado é sempre medido a escrever, nunca confiado a este valor.
	IngestFile(folderID, fileName string, size int64, r io.Reader) (documents.Document, error)

	// DocumentFile devolve o conteúdo do ficheiro de um documento (para
	// o endpoint /documents/{id}/file e para o download na UI) — quem
	// recebe é responsável por fechar o ReadCloser.
	DocumentFile(id string) (content io.ReadCloser, fileName string, err error)

	Close() error
}

// Emitter abstrai a notificação de "documento mudou" — o App desktop
// passa um wailsEmitter (emite um evento para o React); o cmd/server
// headless não tem UI nenhuma para onde emitir, por isso usa NoopEmitter
// (os clientes continuam a ver o estado actualizado ao chamar
// ListDocuments, só sem push em tempo real).
type Emitter interface {
	Emit(eventName string, data ...interface{})
}

type NoopEmitter struct{}

func (NoopEmitter) Emit(string, ...interface{}) {}
