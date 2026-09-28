package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"

	"DocumentApp/internal/documents"
	"DocumentApp/internal/folders"
)

// RemoteBackend fala com um host remoto (outra instância da app em modo
// host, ou o binário cmd/server) pela API HTTP de internal/apiserver —
// ver ROADMAP.md, Fase 6: "cliente ligado por IP".
type RemoteBackend struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewRemote(baseURL, token string) *RemoteBackend {
	return &RemoteBackend{
		baseURL: baseURL,
		token:   token,
		// Sem timeout global: um upload de um ficheiro grande ou uma
		// pesquisa nunca deviam demorar minutos, mas usar um timeout
		// curto aqui cortaria uploads legítimos de ficheiros grandes a
		// meio. Cada pedido individual mais sensível a latência
		// (Search) já é naturalmente rápido do lado do host.
		client: &http.Client{},
	}
}

func (b *RemoteBackend) authed(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer "+b.token)
	return req
}

func (b *RemoteBackend) do(req *http.Request, out interface{}) error {
	resp, err := b.client.Do(b.authed(req))
	if err != nil {
		return fmt.Errorf("remote backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote backend: %s %s devolveu %d: %s", req.Method, req.URL.Path, resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (b *RemoteBackend) ListFolders() ([]folders.Folder, error) {
	req, err := http.NewRequest(http.MethodGet, b.baseURL+"/api/folders", nil)
	if err != nil {
		return nil, err
	}
	var out []folders.Folder
	if err := b.do(req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (b *RemoteBackend) CreateFolder(name, accentGradient string, isShared bool) (folders.Folder, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"name": name, "accentGradient": accentGradient, "isShared": isShared,
	})
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/folders", bytes.NewReader(payload))
	if err != nil {
		return folders.Folder{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out folders.Folder
	if err := b.do(req, &out); err != nil {
		return folders.Folder{}, err
	}
	return out, nil
}

func (b *RemoteBackend) DeleteFolder(id string) error {
	req, err := http.NewRequest(http.MethodDelete, b.baseURL+"/api/folders/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	return b.do(req, nil)
}

func (b *RemoteBackend) ListDocuments(folderID string) ([]documents.Document, error) {
	u := b.baseURL + "/api/documents?folder_id=" + url.QueryEscape(folderID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out []documents.Document
	if err := b.do(req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// IngestFile faz streaming directo de r para o corpo do pedido HTTP —
// nunca carrega o ficheiro inteiro para memória primeiro, para uploads
// grandes num cliente remoto não esgotarem RAM.
func (b *RemoteBackend) IngestFile(folderID, fileName string, size int64, r io.Reader) (documents.Document, error) {
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/documents", r)
	if err != nil {
		return documents.Document{}, err
	}
	req.Header.Set("X-Folder-Id", folderID)
	req.Header.Set("X-File-Name", fileName)
	req.Header.Set("Content-Type", "application/octet-stream")
	if size >= 0 {
		req.ContentLength = size
	}

	var out documents.Document
	if err := b.do(req, &out); err != nil {
		return documents.Document{}, err
	}
	return out, nil
}

func (b *RemoteBackend) ApproveDocument(id string) error {
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/documents/"+url.PathEscape(id)+"/approve", nil)
	if err != nil {
		return err
	}
	return b.do(req, nil)
}

func (b *RemoteBackend) RejectDocument(id string) error {
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/documents/"+url.PathEscape(id)+"/reject", nil)
	if err != nil {
		return err
	}
	return b.do(req, nil)
}

func (b *RemoteBackend) UpdateDocumentClassification(id, documentType string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"documentType": documentType, "tags": tags,
	})
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/documents/"+url.PathEscape(id)+"/classification", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return b.do(req, nil)
}

func (b *RemoteBackend) DeleteDocument(id string) error {
	req, err := http.NewRequest(http.MethodDelete, b.baseURL+"/api/documents/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	return b.do(req, nil)
}

func (b *RemoteBackend) Search(query string) (SearchResponse, error) {
	payload, _ := json.Marshal(map[string]string{"query": query})
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/search", bytes.NewReader(payload))
	if err != nil {
		return SearchResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out SearchResponse
	if err := b.do(req, &out); err != nil {
		return SearchResponse{}, err
	}
	return out, nil
}

func (b *RemoteBackend) AskAssistant(question string) (AssistantAnswer, error) {
	payload, _ := json.Marshal(map[string]string{"question": question})
	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/api/ask", bytes.NewReader(payload))
	if err != nil {
		return AssistantAnswer{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out AssistantAnswer
	if err := b.do(req, &out); err != nil {
		return AssistantAnswer{}, err
	}
	return out, nil
}

// documentFileResponse embrulha o corpo da resposta HTTP para
// implementar io.ReadCloser sem copiar os bytes todos para memória —
// fechar isto fecha a ligação HTTP subjacente.
func (b *RemoteBackend) DocumentFile(id string) (io.ReadCloser, string, error) {
	req, err := http.NewRequest(http.MethodGet, b.baseURL+"/documents/"+url.PathEscape(id)+"/file", nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := b.client.Do(b.authed(req))
	if err != nil {
		return nil, "", fmt.Errorf("remote backend: %w", err)
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, "", fmt.Errorf("remote backend: GET %s devolveu %d: %s", req.URL.Path, resp.StatusCode, string(body))
	}

	_, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	return resp.Body, params["filename"], nil
}

func (b *RemoteBackend) Close() error {
	b.client.CloseIdleConnections()
	return nil
}
