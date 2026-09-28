import {useEffect, useRef, useState} from 'react'
import {ApproveDocument, IngestFile, RejectDocument, UpdateDocumentClassification} from '../wailsjs/go/main/App'
import type {documents} from '../wailsjs/go/models'
import {EventsOn} from '../wailsjs/runtime/runtime'
import Mascot, {MascotLoading} from './Mascot'

function parseTags(tagsJson: string): string[] {
    try {
        const tags = JSON.parse(tagsJson || '[]')
        return Array.isArray(tags) ? tags.filter((t) => typeof t === 'string') : []
    } catch {
        return []
    }
}

function tagsFromText(text: string): string[] {
    return text
        .split(',')
        .map((t) => t.trim())
        .filter((t) => t.length > 0)
}

const STEPS = [
    {key: 'uploading', label: 'A enviar'},
    {key: 'processing', label: 'A extrair e classificar'},
    {key: 'pending_review', label: 'Pronto para rever'},
] as const

function stepIndex(status: string): number {
    const idx = STEPS.findIndex((s) => s.key === status)
    if (idx >= 0) return idx
    if (status === 'ready') return STEPS.length - 1 // já foi aprovado por outra via
    return -1
}

export default function IngestFlowPage({
    folderId,
    filePath,
    onDone,
}: {
    folderId: string
    filePath: string
    onDone: () => void
}) {
    const [doc, setDoc] = useState<documents.Document | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [isSubmitting, setIsSubmitting] = useState(false)
    const [editType, setEditType] = useState('')
    const [editTagsText, setEditTagsText] = useState('')
    const [isSavingEdit, setIsSavingEdit] = useState(false)
    const docIdRef = useRef<string | null>(null)
    const hasStartedRef = useRef(false)
    const editSeededRef = useRef(false)

    useEffect(() => {
        // IngestFile não é idempotente — cria mesmo um documento novo
        // no backend. O React.StrictMode (ver main.tsx) monta, desmonta
        // e volta a montar componentes de propósito em desenvolvimento
        // para apanhar efeitos exactamente assim; sem esta guarda, cada
        // escolha de ficheiro criava DOIS documentos (visto em primeira
        // mão: um ficava "pronto" — o que a página mostrava e que se
        // aprovou — e o outro "a rever", órfão, criado pela segunda
        // invocação do efeito).
        if (hasStartedRef.current) return
        hasStartedRef.current = true

        IngestFile(folderId, filePath)
            .then((created) => {
                docIdRef.current = created.id
                setDoc(created)
            })
            .catch((err) => setError(String(err)))
    }, [folderId, filePath])

    useEffect(() => {
        return EventsOn('document:updated', (updated: documents.Document) => {
            if (updated.id !== docIdRef.current) return
            setDoc(updated)
        })
    }, [])

    // As sugestões do modelo (documentType/tags) só ficam definitivas
    // quando classificationPending passa a false (ver
    // callbackserver.enrichAsync) — semear os campos editáveis antes
    // disso mostraria o fallback (tipo derivado da extensão, sem tags) a
    // ser editado como se já fosse a sugestão real. Semeia só uma vez,
    // para não apagar o que o utilizador já tiver escrito se chegar
    // outro evento "document:updated" por outro motivo.
    //
    // Quando não há texto extraído (ocrExcerpt vazio — o extrator só lê
    // a camada de texto do PDF, não faz OCR de imagem: um PDF digitalizado
    // ou só com desenho vetorial dá sempre 0 caracteres, visto em
    // primeira mão), o enriquecimento nunca chega a correr (ver
    // callbackserver.handleCallback — só arranca com OCRExcerpt não
    // vazio) e classificationPending fica sempre false, sem nunca ter
    // havido uma sugestão real. Nesse caso o "documentType" ainda é só o
    // fallback derivado da extensão (ex. "PDF") — não faz sentido
    // pré-preencher a edição com isso como se fosse uma sugestão da IA.
    useEffect(() => {
        if (!doc || editSeededRef.current) return
        if (doc.status !== 'pending_review' || doc.classificationPending) return
        const hadRealClassification = !!doc.ocrExcerpt
        setEditType(hadRealClassification ? doc.documentType || '' : '')
        setEditTagsText(hadRealClassification ? parseTags(doc.tagsJson).join(', ') : '')
        editSeededRef.current = true
    }, [doc])

    const noExtractableText = doc?.status === 'pending_review' && !doc.classificationPending && !doc.ocrExcerpt

    const isDirty =
        editSeededRef.current &&
        doc !== null &&
        (editType.trim() !== (doc.documentType || '') ||
            tagsFromText(editTagsText).join(',') !== parseTags(doc.tagsJson).join(','))

    const saveEdit = () => {
        if (!doc) return Promise.resolve()
        return UpdateDocumentClassification(doc.id, editType.trim(), tagsFromText(editTagsText))
    }

    const handleSaveEdit = () => {
        setIsSavingEdit(true)
        setError(null)
        saveEdit()
            .then(() => setIsSavingEdit(false))
            .catch((err) => {
                setError(String(err))
                setIsSavingEdit(false)
            })
    }

    const handleApprove = () => {
        if (!doc) return
        setIsSubmitting(true)
        setError(null)
        const proceed = isDirty ? saveEdit() : Promise.resolve()
        proceed
            .then(() => ApproveDocument(doc.id))
            .then(onDone)
            .catch((err) => {
                setError(String(err))
                setIsSubmitting(false)
            })
    }

    const handleReject = () => {
        if (!doc) return
        setIsSubmitting(true)
        setError(null)
        RejectDocument(doc.id)
            .then(onDone)
            .catch((err) => {
                setError(String(err))
                setIsSubmitting(false)
            })
    }

    const currentStep = doc ? stepIndex(doc.status) : -1
    const failed = doc?.status === 'failed'
    const stillClassifying = doc?.status === 'pending_review' && doc.classificationPending

    return (
        <div className="ingest-page">
            <div className="preview-header">
                <button className="btn-secondary" onClick={onDone}>
                    ← Voltar à pasta
                </button>
                {doc && <span className="document-name">{doc.fileName}</span>}
            </div>

            {error && (
                <div className="error-banner" role="alert">
                    {error}
                    <button onClick={() => setError(null)}>×</button>
                </div>
            )}

            {!doc && !error && <MascotLoading text="A carregar o ficheiro…" />}

            {doc && !failed && (
                <ol className="ingest-stepper">
                    {STEPS.map((step, idx) => (
                        <li
                            key={step.key}
                            className={`ingest-step ${idx < currentStep ? 'done' : ''} ${idx === currentStep ? 'active' : ''}`}
                        >
                            <span className="ingest-step-dot">{idx < currentStep ? '✓' : idx + 1}</span>
                            <span>{step.label}</span>
                        </li>
                    ))}
                </ol>
            )}

            {doc && failed && (
                <div className="ingest-failed">
                    <p>O processamento deste documento falhou.</p>
                    <button className="btn-secondary" onClick={onDone}>
                        Voltar à pasta
                    </button>
                </div>
            )}

            {doc && (doc.status === 'pending_review' || doc.status === 'ready') && (
                <div className="ingest-review">
                    <h3>Dados extraídos</h3>

                    {stillClassifying && (
                        <p className="ingest-classifying">
                            <Mascot variant="bounce" size={22} /> A gerar resumo e classificação com IA…
                        </p>
                    )}

                    {noExtractableText && (
                        <p className="ingest-no-text-warning">
                            Não foi possível extrair texto deste documento (pode ser uma imagem digitalizada) — a
                            classificação automática não está disponível. Revê e preenche manualmente antes de
                            aprovar.
                        </p>
                    )}

                    <dl className="document-details">
                        <dt>Tipo de documento</dt>
                        <dd>
                            {stillClassifying || doc.status === 'ready' ? (
                                doc.documentType || '—'
                            ) : (
                                <input
                                    className="ingest-edit-input"
                                    type="text"
                                    value={editType}
                                    onChange={(e) => setEditType(e.target.value)}
                                    placeholder="ex.: Fatura"
                                />
                            )}
                        </dd>
                        <dt>Resumo</dt>
                        <dd>{stillClassifying ? 'a calcular…' : doc.summary || '—'}</dd>
                        <dt>Tags</dt>
                        <dd>
                            {stillClassifying || doc.status === 'ready' ? (
                                (() => {
                                    const tags = parseTags(doc.tagsJson)
                                    return tags.length > 0 ? tags.join(', ') : '—'
                                })()
                            ) : (
                                <input
                                    className="ingest-edit-input"
                                    type="text"
                                    value={editTagsText}
                                    onChange={(e) => setEditTagsText(e.target.value)}
                                    placeholder="tags separadas por vírgula"
                                />
                            )}
                        </dd>
                        <dt>Confiança OCR</dt>
                        <dd>{doc.ocrConfidence ? `${(doc.ocrConfidence * 100).toFixed(0)}%` : '—'}</dd>
                        <dt>Conteúdo extraído</dt>
                        <dd>{doc.ocrExcerpt || '—'}</dd>
                    </dl>

                    {doc.status === 'pending_review' && !stillClassifying && (
                        <div className="ingest-review-actions">
                            <button className="btn-primary" disabled={isSubmitting} onClick={handleApprove}>
                                Aprovar e registar definitivamente
                            </button>
                            <button className="btn-danger" disabled={isSubmitting} onClick={handleReject}>
                                Rejeitar
                            </button>
                            {isDirty && (
                                <button className="btn-secondary" disabled={isSubmitting || isSavingEdit} onClick={handleSaveEdit}>
                                    Guardar alterações
                                </button>
                            )}
                        </div>
                    )}
                    {doc.status === 'pending_review' && stillClassifying && (
                        <div className="ingest-review-actions">
                            <button className="btn-danger" disabled={isSubmitting} onClick={handleReject}>
                                Rejeitar
                            </button>
                        </div>
                    )}
                    {doc.status === 'ready' && <p className="muted">Já aprovado.</p>}
                </div>
            )}
        </div>
    )
}
