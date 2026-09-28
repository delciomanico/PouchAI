import {useEffect, useRef, useState} from 'react'
import type {FormEvent} from 'react'
import './theme.css'
import './App.css'
import {
    ApproveDocument,
    AskAssistant,
    CreateFolder,
    DeleteDocument,
    DeleteFolder,
    DownloadDocumentFile,
    IngestFile,
    ListDocuments,
    ListFolders,
    RejectDocument,
    Search,
    SelectDocumentFile,
} from '../wailsjs/go/main/App'
import type {backend, documents, folders} from '../wailsjs/go/models'
import {EventsOn, OnFileDrop, OnFileDropOff} from '../wailsjs/runtime/runtime'
import {FileTypeIcon, FolderIcon} from './Icons'
import ActivityBar from './ActivityBar'
import type {PanelKey} from './ActivityBar'
import ChatSidebar from './ChatSidebar'
import DocumentPreviewPage from './DocumentPreviewPage'
import IngestFlowPage from './IngestFlowPage'
import {MascotLoading} from './Mascot'
import PlaceholderPanel from './PlaceholderPanel'
import SettingsPanel from './SettingsPanel'
import TitleBar from './TitleBar'

const GRADIENT_PRESETS: Record<string, string> = {
    teal: 'linear-gradient(135deg, #2E7D6B, #123F35)',
    terracotta: 'linear-gradient(135deg, #C97B4A, #6B3A1F)',
    blue: 'linear-gradient(135deg, #3E6FB0, #1B3A63)',
    purple: 'linear-gradient(135deg, #7C5CC4, #3A2766)',
    green: 'linear-gradient(135deg, #4E9B6B, #1E4A30)',
}

const STATUS_LABELS: Record<string, string> = {
    uploading: 'A carregar',
    processing: 'A processar',
    pending_review: 'Por rever',
    ready: 'Pronto',
    failed: 'Falhou',
    rejected: 'Rejeitado',
}

function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    const units = ['KB', 'MB', 'GB']
    let value = bytes / 1024
    let unitIndex = 0
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024
        unitIndex += 1
    }
    return `${value.toFixed(1)} ${units[unitIndex]}`
}

function formatDate(iso: string): string {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return iso
    return d.toLocaleString('pt-PT')
}

// accentGradient é um "linear-gradient(...)" completo (ver GRADIENT_PRESETS)
// — o FolderIcon só precisa de uma cor sólida, por isso extrai a primeira.
function extractFolderColor(accentGradient: string): string {
    const match = accentGradient.match(/#[0-9a-fA-F]{3,8}/)
    return match ? match[0] : '#8C8378'
}

function App() {
    const [folderList, setFolderList] = useState<folders.Folder[]>([])
    const [selectedFolderId, setSelectedFolderId] = useState<string | null>(null)
    const [documentList, setDocumentList] = useState<documents.Document[]>([])
    const [error, setError] = useState<string | null>(null)
    const [loadingFolders, setLoadingFolders] = useState(false)
    const [loadingDocuments, setLoadingDocuments] = useState(false)
    const [isCreatingFolder, setIsCreatingFolder] = useState(false)
    const [newFolderName, setNewFolderName] = useState('')
    const [newFolderGradient, setNewFolderGradient] = useState('teal')
    const [newFolderShared, setNewFolderShared] = useState(false)

    const [searchQuery, setSearchQuery] = useState('')
    const [searchResult, setSearchResult] = useState<backend.SearchResponse | null>(null)
    const [isSearching, setIsSearching] = useState(false)
    // Resposta gerada pelo llm-service para o tipo "answer" da pesquisa —
    // null usa o fallback (resumo já guardado) quando o llm-service está
    // indisponível ou a chamada falha, ver handleSearch.
    const [assistantAnswer, setAssistantAnswer] = useState<string | null>(null)

    const [expandedDocId, setExpandedDocId] = useState<string | null>(null)

    // Pedido explícito: o assistente fica aberto por omissão (era fechado).
    const [isChatOpen, setIsChatOpen] = useState(true)

    const [activePanel, setActivePanel] = useState<PanelKey>('folders')
    const [isSidePanelCollapsed, setIsSidePanelCollapsed] = useState(false)

    const handleSelectPanel = (key: PanelKey) => {
        if (key === activePanel) {
            setIsSidePanelCollapsed((v) => !v)
            return
        }
        setActivePanel(key)
        setIsSidePanelCollapsed(false)
    }

    const [view, setView] = useState<'library' | 'preview' | 'ingest'>('library')
    const [previewDocId, setPreviewDocId] = useState<string | null>(null)
    const [ingestDraft, setIngestDraft] = useState<{folderId: string; filePath: string} | null>(null)

    const refreshFolders = () => {
        setLoadingFolders(true)
        ListFolders()
            .then((result) => setFolderList(result ?? []))
            .catch((err) => setError(String(err)))
            .finally(() => setLoadingFolders(false))
    }

    const refreshDocuments = (folderId: string) => {
        setLoadingDocuments(true)
        ListDocuments(folderId)
            .then((result) => setDocumentList(result ?? []))
            .catch((err) => setError(String(err)))
            .finally(() => setLoadingDocuments(false))
    }

    useEffect(refreshFolders, [])

    useEffect(() => {
        if (selectedFolderId) {
            refreshDocuments(selectedFolderId)
        } else {
            setDocumentList([])
        }
    }, [selectedFolderId])

    // O ai-worker (C++) muda o estado do documento diretamente na SQLite
    // e avisa o Go, que emite este evento — assim a UI reflete
    // uploading -> processing -> pending_review sozinha, sem recarregar.
    const selectedFolderIdRef = useRef(selectedFolderId)
    useEffect(() => {
        selectedFolderIdRef.current = selectedFolderId
    }, [selectedFolderId])

    useEffect(() => {
        return EventsOn('document:updated', (doc: documents.Document) => {
            if (doc.folderId !== selectedFolderIdRef.current) return
            setDocumentList((prev) => {
                const idx = prev.findIndex((d) => d.id === doc.id)
                if (idx === -1) return prev
                const next = [...prev]
                next[idx] = doc
                return next
            })
        })
    }, [])

    // Arrastar e largar ficheiros nativos: EnableFileDrop (main.go) dá-nos
    // caminhos absolutos reais (ao contrário do DataTransfer.files do
    // browser), por isso cada caminho vai directo para IngestFile, tal
    // como o SelectDocumentFile via diálogo.
    useEffect(() => {
        OnFileDrop((_x, _y, paths) => {
            const folderId = selectedFolderIdRef.current
            if (!folderId || paths.length === 0) return
            setError(null)
            Promise.all(paths.map((path) => IngestFile(folderId, path)))
                .then(() => refreshDocuments(folderId))
                .catch((err) => setError(String(err)))
        }, true)
        return () => OnFileDropOff()
    }, [])

    const handleCreateFolder = (e: FormEvent) => {
        e.preventDefault()
        if (!newFolderName.trim()) return
        setError(null)
        CreateFolder(newFolderName.trim(), GRADIENT_PRESETS[newFolderGradient], newFolderShared)
            .then((created) => {
                setNewFolderName('')
                setNewFolderShared(false)
                setIsCreatingFolder(false)
                refreshFolders()
                setSelectedFolderId(created.id)
            })
            .catch((err) => setError(String(err)))
    }

    const handleAddDocument = () => {
        if (!selectedFolderId) return
        setError(null)
        SelectDocumentFile()
            .then((path) => {
                if (!path) return // utilizador cancelou o diálogo
                setIngestDraft({folderId: selectedFolderId, filePath: path})
                setView('ingest')
            })
            .catch((err) => setError(String(err)))
    }

    const handleIngestDone = () => {
        setView('library')
        setIngestDraft(null)
        if (selectedFolderId) refreshDocuments(selectedFolderId)
    }

    const handleApprove = (id: string) => {
        setError(null)
        ApproveDocument(id)
            .then(() => selectedFolderId && refreshDocuments(selectedFolderId))
            .catch((err) => setError(String(err)))
    }

    const handleReject = (id: string) => {
        setError(null)
        RejectDocument(id)
            .then(() => selectedFolderId && refreshDocuments(selectedFolderId))
            .catch((err) => setError(String(err)))
    }

    const handleDownload = (id: string) => {
        setError(null)
        DownloadDocumentFile(id).catch((err) => setError(String(err)))
    }

    const handleOpen = (id: string) => {
        setPreviewDocId(id)
        setView('preview')
    }

    const handleDeleteDocument = (id: string, fileName: string) => {
        if (!window.confirm(`Apagar definitivamente "${fileName}"? Esta ação não pode ser desfeita.`)) return
        setError(null)
        DeleteDocument(id)
            .then(() => selectedFolderId && refreshDocuments(selectedFolderId))
            .catch((err) => setError(String(err)))
    }

    const handleDeleteFolder = (id: string, name: string) => {
        if (!window.confirm(`Apagar a pasta "${name}" e todos os documentos dentro dela? Esta ação não pode ser desfeita.`)) return
        setError(null)
        DeleteFolder(id)
            .then(() => {
                if (selectedFolderId === id) setSelectedFolderId(null)
                refreshFolders()
            })
            .catch((err) => setError(String(err)))
    }

    const toggleDetails = (id: string) => {
        setExpandedDocId((prev) => (prev === id ? null : id))
    }

    const handleSearch = (e: FormEvent) => {
        e.preventDefault()
        const query = searchQuery.trim()
        if (!query) return
        setError(null)
        setAssistantAnswer(null)
        setIsSearching(true)
        Search(query)
            .then((result) => {
                setSearchResult(result)
                if (result.type !== 'answer' || result.results.length === 0) return
                // O tipo "answer" já tinha um resumo guardado como
                // "resposta mais provável" (search-answer-card); agora
                // tenta substituir por uma resposta real gerada pelo
                // llm-service — se falhar (indisponível, degradação
                // graciosa), o resumo guardado continua a aparecer.
                AskAssistant(query)
                    .then((answer) => setAssistantAnswer(answer.answer))
                    .catch(() => setAssistantAnswer(null))
            })
            .catch((err) => {
                setError(String(err))
                setSearchResult(null)
            })
            .finally(() => setIsSearching(false))
    }

    const handleClearSearch = () => {
        setSearchQuery('')
        setSearchResult(null)
        setAssistantAnswer(null)
    }

    const handleOpenSearchHit = (hit: backend.SearchHit) => {
        setSelectedFolderId(hit.folderId)
        handleClearSearch()
        setPreviewDocId(hit.documentId)
        setView('preview')
    }

    // Fontes citadas pelo assistente (ChatSidebar) navegam da mesma forma
    // que um resultado de pesquisa, mas sem mexer no estado da pesquisa
    // em si — o chat é um percurso independente.
    const handleOpenChatSource = (hit: backend.SearchHit) => {
        setSelectedFolderId(hit.folderId)
        setPreviewDocId(hit.documentId)
        setView('preview')
    }

    const selectedFolder = folderList.find((f) => f.id === selectedFolderId) ?? null

    return (
        <div id="app-shell">
            <TitleBar isChatOpen={isChatOpen} onToggleChat={() => setIsChatOpen((v) => !v)} />
            <div id="app">
            <ActivityBar active={activePanel} onSelect={handleSelectPanel} />

            <aside className={`sidebar ${isSidePanelCollapsed ? 'sidebar-collapsed' : ''}`}>
                <div className="sidebar-panel-inner">
                {activePanel === 'folders' && (
                    <>
                        <div className="sidebar-header">
                            <h1>Pastas</h1>
                            <button className="btn-icon" onClick={() => setIsCreatingFolder((v) => !v)} title="Nova pasta">
                                +
                            </button>
                        </div>

                        {isCreatingFolder && (
                            <form className="new-folder-form" onSubmit={handleCreateFolder}>
                                <input
                                    autoFocus
                                    placeholder="Nome da pasta"
                                    value={newFolderName}
                                    onChange={(e) => setNewFolderName(e.target.value)}
                                />
                                <div className="gradient-picker">
                                    {Object.entries(GRADIENT_PRESETS).map(([key, gradient]) => (
                                        <button
                                            key={key}
                                            type="button"
                                            className={`gradient-swatch ${newFolderGradient === key ? 'selected' : ''}`}
                                            style={{background: gradient}}
                                            onClick={() => setNewFolderGradient(key)}
                                            aria-label={key}
                                        />
                                    ))}
                                </div>
                                <label className="checkbox-row">
                                    <input
                                        type="checkbox"
                                        checked={newFolderShared}
                                        onChange={(e) => setNewFolderShared(e.target.checked)}
                                    />
                                    Partilhada com a equipa
                                </label>
                                <div className="form-actions">
                                    <button type="submit" className="btn-primary">Criar</button>
                                    <button type="button" className="btn-secondary" onClick={() => setIsCreatingFolder(false)}>
                                        Cancelar
                                    </button>
                                </div>
                            </form>
                        )}

                        {loadingFolders && <MascotLoading text="A carregar pastas…" />}
                        {!loadingFolders && folderList.length === 0 && <p className="muted">Sem pastas ainda.</p>}

                        <ul className="folder-list">
                            {folderList.map((folder) => (
                                <li key={folder.id} className="folder-list-item">
                                    <button
                                        className={`folder-item ${folder.id === selectedFolderId ? 'selected' : ''}`}
                                        onClick={() => setSelectedFolderId(folder.id)}
                                    >
                                        <FolderIcon color={extractFolderColor(folder.accentGradient)} />
                                        <span className="folder-name">{folder.name}</span>
                                        {folder.isShared && <span className="badge">Equipa</span>}
                                    </button>
                                    <button
                                        className="btn-icon-ghost"
                                        title="Apagar pasta"
                                        onClick={() => handleDeleteFolder(folder.id, folder.name)}
                                    >
                                        ×
                                    </button>
                                </li>
                            ))}
                        </ul>
                    </>
                )}

                {activePanel === 'settings' && <SettingsPanel />}
                {activePanel === 'groups' && (
                    <PlaceholderPanel
                        title="Grupos"
                        description="Partilha de pastas e colaboração em equipa chegam em breve."
                    />
                )}
                {activePanel === 'profile' && (
                    <PlaceholderPanel title="Perfil" description="Contas e perfis de utilizador chegam em breve." />
                )}
                </div>
            </aside>

            <main className="main-panel">
              <div key={view} className="page-transition">
                {view === 'preview' && previewDocId && (
                    <DocumentPreviewPage documentId={previewDocId} onBack={() => setView('library')} />
                )}

                {view === 'ingest' && ingestDraft && (
                    <IngestFlowPage
                        folderId={ingestDraft.folderId}
                        filePath={ingestDraft.filePath}
                        onDone={handleIngestDone}
                    />
                )}

                {view === 'library' && (
                <>
                {error && (
                    <div className="error-banner" role="alert">
                        {error}
                        <button onClick={() => setError(null)}>×</button>
                    </div>
                )}

                <form className="search-bar" onSubmit={handleSearch}>
                    <input
                        type="search"
                        placeholder="Pesquisar documentos… (ex.: uma pergunta ou um assunto)"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                    />
                    <button type="submit" className="btn-primary" disabled={isSearching || !searchQuery.trim()}>
                        {isSearching ? 'A pesquisar…' : 'Pesquisar'}
                    </button>
                    {searchResult && (
                        <button type="button" className="btn-secondary" onClick={handleClearSearch}>
                            Limpar
                        </button>
                    )}
                </form>

                {searchResult && (
                    <div className="search-results">
                        {searchResult.type === 'none' && (
                            <p className="muted">Nenhum documento encontrado para essa pesquisa.</p>
                        )}

                        {searchResult.type === 'answer' && searchResult.results.length > 0 && (
                            <div className="search-answer-card" onClick={() => handleOpenSearchHit(searchResult.results[0])}>
                                <span className="search-answer-label">Resposta mais provável</span>
                                <div className="document-info">
                                    <FileTypeIcon fileName={searchResult.results[0].fileName} />
                                    <span className="document-name">{searchResult.results[0].fileName}</span>
                                </div>
                                <p className="search-summary">{assistantAnswer ?? searchResult.results[0].summary}</p>
                                <span className="muted">relevância {searchResult.results[0].score.toFixed(2)}</span>
                            </div>
                        )}

                        {searchResult.type === 'list' && (
                            <ul className="document-list">
                                {searchResult.results.map((hit) => (
                                    <li key={hit.documentId} className="document-row search-hit-row" onClick={() => handleOpenSearchHit(hit)}>
                                        <div className="document-info">
                                            <FileTypeIcon fileName={hit.fileName} />
                                            <div>
                                                <span className="document-name">{hit.fileName}</span>
                                                <p className="search-summary">{hit.summary}</p>
                                            </div>
                                        </div>
                                        <span className="muted">relevância {hit.score.toFixed(2)}</span>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                )}

                {!searchResult && !selectedFolder && <p className="muted">Escolhe uma pasta para ver os documentos.</p>}

                {!searchResult && selectedFolder && (
                    <>
                        <div className="main-header">
                            <h2>{selectedFolder.name}</h2>
                            <button className="btn-primary" onClick={handleAddDocument}>
                                Adicionar documento
                            </button>
                        </div>

                        {loadingDocuments && <MascotLoading text="A carregar documentos…" />}
                        {!loadingDocuments && documentList.length === 0 && (
                            <p className="muted drop-hint">
                                Esta pasta ainda não tem documentos. Arrasta ficheiros para aqui, ou usa "Adicionar documento".
                            </p>
                        )}

                        <ul className="document-list document-drop-zone">
                            {documentList.map((doc) => (
                                <li key={doc.id} className="document-row-wrapper">
                                    <div className="document-row">
                                        <button className="document-info document-info-btn" onClick={() => toggleDetails(doc.id)}>
                                            <FileTypeIcon fileName={doc.fileName} />
                                            <span className="document-name">{doc.fileName}</span>
                                            <span className={`status-badge status-${doc.status}`}>
                                                {STATUS_LABELS[doc.status] ?? doc.status}
                                            </span>
                                        </button>
                                        <div className="document-actions">
                                            <button className="btn-secondary" onClick={() => handleOpen(doc.id)}>
                                                Abrir
                                            </button>
                                            <button className="btn-secondary" onClick={() => handleDownload(doc.id)}>
                                                Descarregar
                                            </button>
                                            <button className="btn-secondary" onClick={() => handleApprove(doc.id)}>
                                                Aprovar
                                            </button>
                                            <button className="btn-danger" onClick={() => handleReject(doc.id)}>
                                                Rejeitar
                                            </button>
                                            <button className="btn-danger" onClick={() => handleDeleteDocument(doc.id, doc.fileName)}>
                                                Apagar
                                            </button>
                                        </div>
                                    </div>
                                    {expandedDocId === doc.id && (
                                        <dl className="document-details">
                                            <dt>Autor</dt>
                                            <dd>{doc.author || '—'}</dd>
                                            <dt>Tamanho</dt>
                                            <dd>{formatBytes(doc.sizeBytes)}</dd>
                                            <dt>Confiança OCR</dt>
                                            <dd>{doc.ocrConfidence ? `${(doc.ocrConfidence * 100).toFixed(0)}%` : '—'}</dd>
                                            <dt>Tags</dt>
                                            <dd>{(() => {
                                                try {
                                                    const tags = JSON.parse(doc.tagsJson || '[]')
                                                    return Array.isArray(tags) && tags.length > 0 ? tags.join(', ') : '—'
                                                } catch {
                                                    return '—'
                                                }
                                            })()}</dd>
                                            <dt>Adicionado em</dt>
                                            <dd>{formatDate(doc.createdAt)}</dd>
                                        </dl>
                                    )}
                                </li>
                            ))}
                        </ul>
                    </>
                )}
                </>
                )}
              </div>
            </main>

            {isChatOpen && (
                <ChatSidebar
                    key={view}
                    onOpenSource={handleOpenChatSource}
                    onClose={() => setIsChatOpen(false)}
                />
            )}
            </div>
        </div>
    )
}

export default App
