// Ícones SVG desenhados à mão, sem biblioteca externa — só um punhado
// de glifos precisos (pasta, tipos de ficheiro), não vale a pena puxar
// uma dependência inteira (react-icons/lucide/etc.) só por isto.
import type {ReactElement, ReactNode} from 'react'

export function FolderIcon({color}: {color: string}) {
    return (
        <svg width="20" height="16" viewBox="0 0 20 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
                d="M1 3C1 1.89543 1.89543 1 3 1H7.17157C7.70201 1 8.21071 1.21071 8.58579 1.58579L9.41421 2.41421C9.78929 2.78929 10.298 3 10.8284 3H17C18.1046 3 19 3.89543 19 5V13C19 14.1046 18.1046 15 17 15H3C1.89543 15 1 14.1046 1 13V3Z"
                fill={color}
                fillOpacity="0.18"
                stroke={color}
                strokeWidth="1.3"
            />
        </svg>
    )
}

function FileBase({children, tint}: {children?: ReactNode; tint: string}) {
    return (
        <svg width="18" height="20" viewBox="0 0 18 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
                d="M2 2C2 0.895431 2.89543 0 4 0H10.1716C10.702 0 11.2107 0.210714 11.5858 0.585786L16.4142 5.41421C16.7893 5.78929 17 6.29799 17 6.82843V18C17 19.1046 16.1046 20 15 20H4C2.89543 20 2 19.1046 2 18V2Z"
                fill={tint}
                fillOpacity="0.15"
                stroke={tint}
                strokeWidth="1.2"
            />
            <path d="M10.5 0.5V5.5C10.5 6.05228 10.9477 6.5 11.5 6.5H16.3" stroke={tint} strokeWidth="1.2" />
            {children}
        </svg>
    )
}

function FilePdf() {
    return (
        <FileBase tint="#C24B3F">
            <text x="3.2" y="15.5" fontSize="6" fontWeight="700" fill="#C24B3F" fontFamily="sans-serif">
                PDF
            </text>
        </FileBase>
    )
}

function FileWord() {
    return (
        <FileBase tint="#3E6FB0">
            <text x="2.6" y="15.5" fontSize="6" fontWeight="700" fill="#3E6FB0" fontFamily="sans-serif">
                DOC
            </text>
        </FileBase>
    )
}

function FileImage() {
    return (
        <FileBase tint="#4E9B6B">
            <circle cx="6" cy="11.5" r="1.3" fill="#4E9B6B" />
            <path d="M3.5 16L7.2 12.3L9.5 14.6L12 11L14.5 16H3.5Z" fill="#4E9B6B" fillOpacity="0.7" />
        </FileBase>
    )
}

function FileGeneric() {
    return <FileBase tint="#8C8378" />
}

const EXTENSION_ICONS: Record<string, () => ReactElement> = {
    pdf: FilePdf,
    doc: FileWord,
    docx: FileWord,
    png: FileImage,
    jpg: FileImage,
    jpeg: FileImage,
    gif: FileImage,
    webp: FileImage,
}

export function FileTypeIcon({fileName}: {fileName: string}) {
    const ext = fileName.split('.').pop()?.toLowerCase() ?? ''
    const Icon = EXTENSION_ICONS[ext] ?? FileGeneric
    return <Icon />
}

export function ChatIcon() {
    return (
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="1" y="1.5" width="14" height="9.5" rx="2.5" stroke="currentColor" strokeWidth="1.3" />
            <path d="M4.5 11L4.5 14L7.5 11" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round" />
        </svg>
    )
}

// Ícones da barra de atividades (ActivityBar) — sempre monocromáticos
// (currentColor), mesma convenção do ChatIcon acima, para herdarem a cor
// do botão (normal/hover/ativo) só por CSS.

export function RailFolderIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
                d="M2 5C2 3.89543 2.89543 3 4 3H8.17157C8.70201 3 9.21071 3.21071 9.58579 3.58579L10.4142 4.41421C10.7893 4.78929 11.298 5 11.8284 5H16C17.1046 5 18 5.89543 18 7V15C18 16.1046 17.1046 17 16 17H4C2.89543 17 2 16.1046 2 15V5Z"
                stroke="currentColor"
                strokeWidth="1.3"
            />
        </svg>
    )
}

export function SettingsIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="10" cy="10" r="2.6" stroke="currentColor" strokeWidth="1.3" />
            <path
                d="M10 2.5V4.2M10 15.8V17.5M17.5 10H15.8M4.2 10H2.5M15.16 4.84L13.95 6.05M6.05 13.95L4.84 15.16M15.16 15.16L13.95 13.95M6.05 6.05L4.84 4.84"
                stroke="currentColor"
                strokeWidth="1.3"
                strokeLinecap="round"
            />
        </svg>
    )
}

export function GroupsIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="7" cy="6.5" r="2.3" stroke="currentColor" strokeWidth="1.3" />
            <circle cx="14" cy="7.5" r="1.9" stroke="currentColor" strokeWidth="1.3" opacity="0.7" />
            <path
                d="M2.5 16.5C2.5 13.5 4.5 11.5 7 11.5C9.5 11.5 11.5 13.5 11.5 16.5"
                stroke="currentColor"
                strokeWidth="1.3"
                strokeLinecap="round"
            />
            <path
                d="M12.5 12C14.7 12 16.3 13.6 16.3 16.3"
                stroke="currentColor"
                strokeWidth="1.3"
                strokeLinecap="round"
                opacity="0.7"
            />
        </svg>
    )
}

export function ProfileIcon() {
    return (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="10" cy="6.8" r="3.3" stroke="currentColor" strokeWidth="1.3" />
            <path
                d="M3.5 17C3.5 13.5 6.4 11.5 10 11.5C13.6 11.5 16.5 13.5 16.5 17"
                stroke="currentColor"
                strokeWidth="1.3"
                strokeLinecap="round"
            />
        </svg>
    )
}
