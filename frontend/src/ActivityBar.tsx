import type {ReactElement} from 'react'
import {GroupsIcon, ProfileIcon, RailFolderIcon, SettingsIcon} from './Icons'

export type PanelKey = 'folders' | 'settings' | 'groups' | 'profile'

const ITEMS: {key: PanelKey; label: string; icon: () => ReactElement}[] = [
    {key: 'folders', label: 'Pastas', icon: RailFolderIcon},
    {key: 'groups', label: 'Grupos', icon: GroupsIcon},
    {key: 'settings', label: 'Definições', icon: SettingsIcon},
    {key: 'profile', label: 'Perfil', icon: ProfileIcon},
]

// Rail de ícones à VSCode: cada botão é uma área funcional (pastas,
// grupos, definições, perfil); clicar na já ativa recolhe o painel ao
// lado (ver isSidePanelCollapsed em App.tsx), clicar noutra troca e
// reabre.
export default function ActivityBar({active, onSelect}: {active: PanelKey; onSelect: (key: PanelKey) => void}) {
    return (
        <nav className="activity-bar">
            {ITEMS.map(({key, label, icon: Icon}) => (
                <button
                    key={key}
                    className={`activity-bar-btn ${active === key ? 'active' : ''}`}
                    title={label}
                    onClick={() => onSelect(key)}
                >
                    <Icon />
                </button>
            ))}
        </nav>
    )
}
