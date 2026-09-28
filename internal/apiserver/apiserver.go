// Package apiserver expõe um backend.LocalBackend por HTTP para
// RemoteBackend (internal/backend/remote.go) — é o lado "host" do modo
// equipa (ROADMAP.md, Fase 6). Nunca fala com um Backend genérico: só um
// host (que tem sempre um LocalBackend próprio) serve esta API — um
// cliente nunca reexpõe o que recebe de outro host.
//
// Autenticação: um único token partilhado (Authorization: Bearer
// <token>), verificado em todas as rotas — não há gestão de
// utilizadores nesta fase, só "conhece o token da equipa ou não entra".
package apiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"DocumentApp/internal/backend"
)

type Server struct {
	backend *backend.LocalBackend
	token   string
	mux     *http.ServeMux
}

func New(b *backend.LocalBackend, token string) *Server {
	s := &Server{backend: b, token: token, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Serve arranca a escutar em addr (ex. "0.0.0.0:8790") e bloqueia —
// chamar em goroutine própria. addr com porto 0 deixa o SO escolher.
func (s *Server) Serve(addr string) error {
	log.Printf("apiserver: a servir em %s", addr)
	return http.ListenAndServe(addr, s.withAuth(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/folders", s.handleListFolders)
	s.mux.HandleFunc("POST /api/folders", s.handleCreateFolder)
	s.mux.HandleFunc("DELETE /api/folders/{id}", s.handleDeleteFolder)
	s.mux.HandleFunc("GET /api/documents", s.handleListDocuments)
	s.mux.HandleFunc("POST /api/documents", s.handleIngestFile)
	s.mux.HandleFunc("POST /api/documents/{id}/approve", s.handleApprove)
	s.mux.HandleFunc("POST /api/documents/{id}/reject", s.handleReject)
	s.mux.HandleFunc("POST /api/documents/{id}/classification", s.handleUpdateClassification)
	s.mux.HandleFunc("DELETE /api/documents/{id}", s.handleDeleteDocument)
	s.mux.HandleFunc("POST /api/search", s.handleSearch)
	s.mux.HandleFunc("POST /api/ask", s.handleAsk)
	s.mux.HandleFunc("GET /documents/{id}/file", s.handleDocumentFile)
}

// withAuth exige "Authorization: Bearer <token>" em todos os pedidos —
// aplicado a todo o mux, não rota a rota, para nunca ser possível
// esquecer uma rota nova sem protecção.
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" || got != s.token {
			http.Error(w, "token inválido ou em falta", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	http.Error(w, err.Error(), status)
}

func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {
	list, err := s.backend.ListFolders()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type createFolderRequest struct {
	Name           string `json:"name"`
	AccentGradient string `json:"accentGradient"`
	IsShared       bool   `json:"isShared"`
}

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("payload inválido: %w", err))
		return
	}
	folder, err := s.backend.CreateFolder(req.Name, req.AccentGradient, req.IsShared)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, folder)
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.DeleteFolder(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder_id")
	if folderID == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("falta o parâmetro folder_id"))
		return
	}
	list, err := s.backend.ListDocuments(folderID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleIngestFile recebe o conteúdo do ficheiro em bruto no corpo do
// pedido — não multipart, para o cliente poder fazer streaming directo
// do ficheiro sem o carregar todo para memória primeiro (ver
// RemoteBackend.IngestFile).
func (s *Server) handleIngestFile(w http.ResponseWriter, r *http.Request) {
	folderID := r.Header.Get("X-Folder-Id")
	fileName := r.Header.Get("X-File-Name")
	if folderID == "" || fileName == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("faltam os cabeçalhos X-Folder-Id e/ou X-File-Name"))
		return
	}

	doc, err := s.backend.IngestFile(folderID, fileName, r.ContentLength, r.Body)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.ApproveDocument(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.RejectDocument(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type updateClassificationRequest struct {
	DocumentType string   `json:"documentType"`
	Tags         []string `json:"tags"`
}

func (s *Server) handleUpdateClassification(w http.ResponseWriter, r *http.Request) {
	var req updateClassificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("payload inválido: %w", err))
		return
	}
	if err := s.backend.UpdateDocumentClassification(r.PathValue("id"), req.DocumentType, req.Tags); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.DeleteDocument(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type searchRequest struct {
	Query string `json:"query"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("payload inválido: %w", err))
		return
	}
	result, err := s.backend.Search(req.Query)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type askRequest struct {
	Question string `json:"question"`
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("payload inválido: %w", err))
		return
	}
	result, err := s.backend.AskAssistant(req.Question)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDocumentFile(w http.ResponseWriter, r *http.Request) {
	content, fileName, err := s.backend.DocumentFile(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	defer content.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, fileName))
	io.Copy(w, content)
}
