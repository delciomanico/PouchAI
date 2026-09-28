# Gestão Documental — DocumentApp

Aplicação desktop nativa (Qt 6 + QML + C++) para gestão documental com
classificação automática por OCR/IA. Este repositório contém a camada de
UI completa, ligada a modelos C++ reais através de uma interface de
backend mockada — pronta para receber o motor local (SQLite, filesystem,
pipeline OCR/IA, servidor HTTP embutido) sem alterar nenhum ficheiro QML.

## Compilar e correr

Requisitos: Qt 6.5+ (módulos Quick, Qml, Sql), CMake 3.21+, um compilador
C++20.

```bash
cmake -B build -S .
cmake --build build
./build/DocumentApp        # Linux/macOS
build\DocumentApp.exe      # Windows
```

## Arquitetura

```
ui/
├── Main.qml              # App Shell: janela, AppTopBar, navegação por página
├── theme/Theme.qml        # Design system (singleton): cores, tipografia, spacing…
├── components/            # Peças reutilizáveis (botões 3D, cartões, estados…)
├── dialogs/                # Diálogos modais
└── pages/                  # Uma página por ecrã

src/
├── main.cpp                  # Bootstrap do QQmlApplicationEngine
├── core/
│   ├── BackendInterface.h    # Única fronteira entre UI e dados
│   └── MockBackend.*         # Implementação em memória, dados realistas
└── models/                   # QAbstractListModel reais (Document/Folder/Team)
```

`BackendInterface` é o único sítio por onde a UI fala com dados. Hoje
`main.cpp` instancia `MockBackend`; quando o núcleo em C++ estiver pronto,
troca-se essa linha por `LocalBackend` (modo standalone/host) e
`RemoteBackend` (modo cliente LAN) — nenhum ficheiro QML precisa de mudar,
porque todas as páginas só conhecem `backend` e os `QAbstractListModel*`
que ele expõe.

## Páginas implementadas (fiéis às frames)

| Página | Frame de origem |
|---|---|
| `WelcomePage` | Login.dc.html |
| `OnboardingPage` | Onboarding / Onboarding-Equipa / Onboarding-Pronto |
| `LibraryPage` | Main.dc.html (painel + chat IA) |
| `DocumentsPage` | Pasta.dc.html |
| `UploadPage` | NovoDocumento.dc.html / NovoDocumento-Massa.dc.html |
| `SearchPage` | TodosDocumentos.dc.html |
| `SettingsPage` | Definicoes.dc.html (+ separador **Rede**, ver abaixo) |
| `ProfilePage` | Perfil.dc.html |

## Adaptações face às frames

- **Sem barra de título falsa**: as frames de Login/Onboarding simulavam
  uma janela nativa (semáforo macOS) porque eram mockups HTML. Aqui a
  janela É nativa a sério — o SO já fornece essa moldura, por isso não
  foi reproduzida.
- **"Rede"**: adicionei este separador a `SettingsPage` (não existia
  frame dedicada) para cobrir a partilha de biblioteca na LAN e a ligação
  a bibliotecas descobertas, usando o texto de exemplo do pedido original
  e o mesmo design system. Vale a pena desenhar uma frame própria antes
  de refinar mais.
- **`DocumentPage` (visualizador)**: não existe frame para este ecrã.
  Implementei a estrutura (conteúdo + metadados/IA/versões em separadores)
  para já poder ligar um motor de renderização real, mas o layout é
  provisório — recomendo desenhar a frame antes de o polir.
- **"Nova pasta"**: nas frames era um cartão tracejado dentro da grelha;
  aqui ficou como botão junto ao seletor de vista, para manter a grelha
  puramente alimentada por `folderModel` (sem misturar uma linha
  sintética num modelo de dados real — ver notas de performance abaixo).

## Performance

`DocumentModel`/`FolderModel`/`TeamModel` são `QAbstractListModel` reais
(não arrays JS), para que `ListView`/`GridView` façam virtualização.
`DocumentsPage`/`SearchPage`/`UploadPage` hoje filtram em JS por
simplicidade (biblioteca mockada, pequena); ao ligar SQLite, substituir
esses filtros por um `QSortFilterProxyModel` em C++ ou, melhor ainda,
por queries filtradas na origem.

## Por fazer

- `LocalBackend` (SQLite + filesystem + pipeline OCR/IA em C++) e
  `RemoteBackend` (cliente HTTP/mDNS), implementando `BackendInterface`.
- Motor de renderização de documentos (PDF/imagem) para `DocumentPage`.
- Ligar `AIChatPanel`, `SmartSearchBar` e a "Classificação sugerida pela
  IA" do `UploadPage` ao motor de IA real (hoje mockados/estáticos).
- Ícone e instalador por plataforma (`assets/` está pronto a receber).
