import {useEffect, useState} from 'react'
import {DownloadDocumentFile, GetDocumentPreview, OpenDocumentFile} from '../wailsjs/go/main/App'
import type {main} from '../wailsjs/go/models'

export default function DocumentPreviewPage({documentId, onBack}: {documentId: string; onBack: () => void}) {
    const [preview, setPreview] = useState<main.DocumentPreview | null>(null)
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        setPreview(null)
        setError(null)
        setLoading(true)
        GetDocumentPreview(documentId)
            .then((result) => setPreview(result))
            .catch((err) => setError(String(err)))
            .finally(() => setLoading(false))
    }, [documentId])

    const handleOpenInSystem = () => {
        setError(null)
        OpenDocumentFile(documentId).catch((err) => setError(String(err)))
    }

    const handleDownload = () => {
        setError(null)
        DownloadDocumentFile(documentId).catch((err) => setError(String(err)))
    }

    return (
        <div className="preview-page">
            <div className="preview-header">
                <button className="btn-secondary" onClick={onBack}>
                    ← Voltar
                </button>
                {preview && <span className="document-name">{preview.fileName}</span>}
                <div className="preview-header-actions">
                    <button className="btn-secondary" onClick={handleOpenInSystem}>
                        Abrir no leitor do sistema
                    </button>
                    <button className="btn-secondary" onClick={handleDownload}>
                        Descarregar
                    </button>
                </div>
            </div>

            {error && (
                <div className="error-banner" role="alert">
                    {error}
                    <button onClick={() => setError(null)}>×</button>
                </div>
            )}

            <div className="preview-body">
                {loading && <p className="muted">A carregar pré-visualização…</p>}

                {!loading && preview?.previewable && preview.mimeType.startsWith('image/') && (
                    <img className="preview-image" src={preview.dataUrl} alt={preview.fileName} />
                )}

                {!loading && preview?.previewable && preview.mimeType === 'application/pdf' && (
                    <iframe className="preview-pdf" src={preview.dataUrl} title={preview.fileName} />
                )}

                {!loading && preview && !preview.previewable && (
                    <p className="muted">
                        Este tipo de ficheiro ({preview.mimeType}) não pode ser pré-visualizado aqui — usa "Abrir no
                        leitor do sistema" ou "Descarregar".
                    </p>
                )}
            </div>
        </div>
    )
}
