import {useEffect, useState} from 'react'
import {GetAppInfo} from '../wailsjs/go/main/App'
import type {main} from '../wailsjs/go/models'

const MODE_LABELS: Record<string, string> = {
    solo: 'Individual (sem rede)',
    host: 'Anfitrião de equipa',
    client: 'Cliente de equipa',
}

// Conteúdo real (não inventado): o único dado de configuração que a app
// já expõe hoje é o modo de execução (ver runmode.go). Mais opções
// ficam para quando existirem — preferível a fingir controlos que não
// fazem nada.
export default function SettingsPanel() {
    const [info, setInfo] = useState<main.AppInfo | null>(null)

    useEffect(() => {
        GetAppInfo().then(setInfo).catch(() => {})
    }, [])

    return (
        <div className="settings-panel">
            <h2>Definições</h2>
            <dl className="document-details">
                <dt>Aplicação</dt>
                <dd>{info?.name ?? '—'}</dd>
                <dt>Modo</dt>
                <dd>{info ? MODE_LABELS[info.mode] ?? info.mode : '—'}</dd>
            </dl>
            <p className="muted">Mais opções de configuração chegam em breve.</p>
        </div>
    )
}
