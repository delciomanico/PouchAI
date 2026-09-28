import logo from './assets/pouch-logo.jpg'

type MascotVariant = 'bounce' | 'pulse' | 'wiggle' | 'static'

// O logo (pouch-logo.jpg) é um raster, não um SVG vetorial — em vez de
// tentar recriar o desenho à mão (arriscado, não é nosso para adivinhar
// traço a traço), anima-se a imagem inteira com transform/opacity em
// CSS. Dá para animações de "salto", "pulsar" e "entrada" sem precisar
// de vetorizar nada.
export default function Mascot({
    variant = 'bounce',
    size = 40,
    label,
}: {
    variant?: MascotVariant
    size?: number
    label?: string
}) {
    return (
        <span
            className={`mascot mascot-${variant}`}
            style={{width: size, height: size}}
            role="img"
            aria-label={label ?? 'PouchIA'}
        >
            <img src={logo} alt="" className="mascot-img" />
        </span>
    )
}

// Combinação pronta a usar para estados de espera (ingest a classificar,
// pastas/documentos a carregar, assistente "a pensar") — substitui o
// texto simples "A carregar…" que existia antes em vários sítios.
export function MascotLoading({text}: {text: string}) {
    return (
        <div className="mascot-loading">
            <Mascot variant="bounce" size={26} />
            <span>{text}</span>
        </div>
    )
}
