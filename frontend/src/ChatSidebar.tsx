import {useState} from 'react'
import type {FormEvent} from 'react'
import {AskAssistant} from '../wailsjs/go/main/App'
import type {backend} from '../wailsjs/go/models'
import {FileTypeIcon} from './Icons'
import {MascotLoading} from './Mascot'

type ChatMessage = {
    role: 'user' | 'assistant'
    text: string
    sources?: backend.SearchHit[]
    error?: boolean
}

// Sidebar de chat com IA local (RAG sobre os documentos já indexados,
// ver backend.AskAssistant). Conversa totalmente efémera por desenho: o
// componente não recebe nem eleva o histórico de mensagens em lado
// nenhum — App.tsx desmonta-o (não só esconde) sempre que a sidebar
// fecha ou a vista muda, para o próximo mount começar sempre vazio.
export default function ChatSidebar({onOpenSource, onClose}: {
    onOpenSource: (hit: backend.SearchHit) => void
    onClose: () => void
}) {
    const [messages, setMessages] = useState<ChatMessage[]>([])
    const [question, setQuestion] = useState('')
    const [isAsking, setIsAsking] = useState(false)

    const handleAsk = (e: FormEvent) => {
        e.preventDefault()
        const q = question.trim()
        if (!q || isAsking) return

        setMessages((prev) => [...prev, {role: 'user', text: q}])
        setQuestion('')
        setIsAsking(true)

        AskAssistant(q)
            .then((result) => {
                setMessages((prev) => [...prev, {role: 'assistant', text: result.answer, sources: result.sources}])
            })
            .catch((err) => {
                setMessages((prev) => [...prev, {role: 'assistant', text: String(err), error: true}])
            })
            .finally(() => setIsAsking(false))
    }

    return (
        <aside className="chat-sidebar">
            <div className="sidebar-header">
                <h1>Assistente</h1>
                <button className="btn-icon" onClick={onClose} title="Fechar assistente">
                    ×
                </button>
            </div>

            <div className="chat-messages">
                {messages.length === 0 && (
                    <p className="muted">Pergunta alguma coisa sobre os teus documentos.</p>
                )}
                {messages.map((m, i) => (
                    <div
                        key={i}
                        className={`chat-message chat-message-${m.role}${m.error ? ' chat-message-error' : ''}`}
                    >
                        <p>{m.text}</p>
                        {m.sources && m.sources.length > 0 && (
                            <div className="chat-sources">
                                {m.sources.map((hit) => (
                                    <button
                                        key={hit.documentId}
                                        type="button"
                                        className="chat-source-chip"
                                        onClick={() => onOpenSource(hit)}
                                    >
                                        <FileTypeIcon fileName={hit.fileName} />
                                        <span>{hit.fileName}</span>
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>
                ))}
                {isAsking && <MascotLoading text="A pensar…" />}
            </div>

            <form className="chat-input-form" onSubmit={handleAsk}>
                <input
                    autoFocus
                    placeholder="Pergunta aos teus documentos…"
                    value={question}
                    onChange={(e) => setQuestion(e.target.value)}
                    disabled={isAsking}
                />
                <button type="submit" className="btn-primary" disabled={isAsking || !question.trim()}>
                    Enviar
                </button>
            </form>
        </aside>
    )
}
