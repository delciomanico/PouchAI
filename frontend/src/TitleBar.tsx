import {useEffect, useState} from 'react'
import {
    Quit,
    WindowFullscreen,
    WindowIsFullscreen,
    WindowMinimise,
    WindowToggleMaximise,
    WindowUnfullscreen,
} from '../wailsjs/runtime/runtime'
import logo from './assets/pouch-logo.jpg'
import {ChatIcon} from './Icons'

// Barra de controlo da janela — a janela já tem a moldura nativa do SO
// (não corre em modo Frameless), mas o pedido foi para teres estes
// botões sempre visíveis dentro da própria app também. O botão do
// assistente vive aqui (não dentro do próprio ChatSidebar) para continuar
// visível mesmo com a sidebar fechada — é o único sítio sempre presente
// em qualquer vista.
export default function TitleBar({isChatOpen, onToggleChat}: {isChatOpen: boolean; onToggleChat: () => void}) {
    const [isFullscreen, setIsFullscreen] = useState(false)

    useEffect(() => {
        WindowIsFullscreen().then(setIsFullscreen).catch(() => {})
    }, [])

    const handleToggleFullscreen = () => {
        if (isFullscreen) {
            WindowUnfullscreen()
            setIsFullscreen(false)
        } else {
            WindowFullscreen()
            setIsFullscreen(true)
        }
    }

    return (
        <div className="titlebar">
            <span className="titlebar-brand">
                <img alt="" className="titlebar-logo" src={logo} />
                <span className="titlebar-title">PouchIA</span>
            </span>
            <div className="titlebar-actions">
                <button
                    className={`titlebar-btn ${isChatOpen ? 'titlebar-btn-active' : ''}`}
                    title={isChatOpen ? 'Fechar assistente' : 'Abrir assistente'}
                    onClick={onToggleChat}
                >
                    <ChatIcon />
                </button>
                <button className="titlebar-btn" title="Minimizar" onClick={() => WindowMinimise()}>
                    &#x2212;
                </button>
                <button className="titlebar-btn" title="Maximizar / restaurar" onClick={() => WindowToggleMaximise()}>
                    &#x25A1;
                </button>
                <button
                    className="titlebar-btn"
                    title={isFullscreen ? 'Sair de ecrã inteiro' : 'Ecrã inteiro'}
                    onClick={handleToggleFullscreen}
                >
                    &#x26F6;
                </button>
                <button className="titlebar-btn titlebar-btn-close" title="Fechar" onClick={() => Quit()}>
                    &#x2715;
                </button>
            </div>
        </div>
    )
}
