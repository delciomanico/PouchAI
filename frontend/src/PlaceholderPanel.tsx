import Mascot from './Mascot'

// Usado por painéis da ActivityBar que ainda não têm nenhuma
// funcionalidade real por trás (Grupos, Perfil) — antes de existir
// dados/backend para eles, mostrar algo a fingir seria pior do que ser
// direto sobre o que ainda não existe.
export default function PlaceholderPanel({title, description}: {title: string; description: string}) {
    return (
        <div className="placeholder-panel">
            <Mascot variant="wiggle" size={56} />
            <h2>{title}</h2>
            <p className="muted">{description}</p>
        </div>
    )
}
