<#
.SYNOPSIS
    Descarrega os dois modelos GGUF do motor de IA local (chat/resumo +
    embeddings/classificação) para a pasta de dados da app.

.DESCRIPTION
    Passo manual único — nunca é disparado pelo build (ver Makefile) nem
    pelo arranque da app. Sem isto, o llm-service simplesmente não
    arranca e a app degrada graciosamente (sem resumo/classificação/chat
    reais), como já acontece hoje se o ai-worker/search-service faltarem.

    Modelos escolhidos (ambos licença permissiva, ambos pequenos o
    suficiente para correr em CPU sem GPU):
      - chat.gguf:  Qwen2.5-0.5B-Instruct, quantização Q4_K_M
                    (Qwen/Qwen2.5-0.5B-Instruct-GGUF, Apache-2.0)
                    — só para chat e resumo (tarefas genuinamente
                    geradoras, ver plano "Dois modelos, dois papéis").
                    TROCADO de 1.5B para 0.5B em 2026-09-28: o 1.5B
                    demorava 65-90s só a carregar no arranque (disco
                    frio) neste hardware CPU-only. O 0.5B carrega em
                    15-20s — ganho grande no arranque. NOTA: o tempo de
                    GERAÇÃO por pedido não melhorou na mesma proporção
                    (ambos batem perto do kGenerationDeadline de 90s em
                    main.cpp para um resumo real) — o gargalo aqui
                    parece ser a CPU em si, não o tamanho do modelo; ver
                    memory motor-ia-local para detalhe.
      - embed.gguf: paraphrase-multilingual-mpnet-base-v2, quantização
                    Q8_0 (keisuke-miyako/paraphrase-multilingual-mpnet-base-v2-gguf-q8_0,
                    base sentence-transformers/paraphrase-multilingual-mpnet-base-v2,
                    Apache-2.0) — suporta português explicitamente; usado
                    só para classificação zero-shot por comparação de
                    cosseno, nunca para gerar texto.
                    TROCADO de multilingual-e5-small em 2026-09-28 (mesmo
                    dia, segunda troca): o e5 é um modelo de RECUPERAÇÃO
                    (retrieval), treinado com prefixos "query: "/
                    "passage: " para tarefas assimétricas pergunta→
                    documento — mau ajuste para "comparar um documento
                    contra descrições fixas de categoria", que é uma
                    tarefa de SIMILARIDADE SEMÂNTICA direta. Medido com 4
                    documentos reais: a margem de confiança do e5 entre
                    1º e 2º candidato nunca passou de 0.006 (um cartão de
                    visita chegou a ser classificado com CONFIANÇA MAIOR
                    no rótulo ERRADO do que um currículo no rótulo
                    certo). Com o mpnet (modelo de similaridade direta,
                    sem prefixo, mesma configuração de pooling por
                    média), a margem em casos claros sobe para 0.18 —
                    quase 60x mais sinal. Custo: ficheiro ~2.3x maior
                    (126MB → 303MB), continua perfeitamente viável em
                    CPU. Ver kCategoryMargin em llm-service/src/main.cpp
                    para a calibração final.

    ATENÇÃO — testada e REJEITADA a conversão em
    cstr/multilingual-e5-small-GGUF: falha a carregar no llama.cpp
    vendored aqui com "bert model needs to define token type count"
    (falta metadata BERT nessa conversão específica; graceful degradation
    da app confirmada — llm-service não arranca, resto da app continua
    normal). Conversões do keisuke-miyako (usadas para os dois modelos
    de embeddings testados até agora, e5-small e mpnet-base) carregam
    sempre bem. Se algum dia for preciso trocar de repositório outra
    vez, testar sempre o carregamento antes de pinar — conversões GGUF
    de modelos BERT/XLM-RoBERTa nem sempre incluem toda a metadata que
    versões recentes do llama.cpp exigem.

    CALIBRAÇÃO DOS LIMIARES DE CLASSIFICAÇÃO — ver comentário completo
    junto a kCategoryAbsoluteFloor em llm-service/src/main.cpp. Resumo
    honesto (mesmo com o mpnet, melhor que o e5): só os casos muito
    claros (ex. um currículo bem escrito) se distinguem do ruído com
    confiança — casos ambíguos (uma fatura genérica, um cartão de
    visita) continuam com margens parecidas às de texto sem sentido
    nenhum, e por isso caem em "Outro" por decisão (preferir não
    inventar a arriscar uma etiqueta errada). Um bug real também foi
    corrigido nesta linha de trabalho: tags candidatas de uma palavra só
    colapsavam quase todas para o mesmo vector de embedding (o prefixo
    do e5 dominava a média de pooling) — corrigido trocando para frases
    descritivas curtas, mantido mesmo depois de trocar de modelo.

    Os sha256 abaixo foram calculados a partir de um download real destes
    dois ficheiros (ver memory motor-ia-local — sessão de 2026-09-28).
    Se a Hugging Face alguma vez re-quantizar/substituir estes ficheiros,
    o hash deixa de bater e o script avisa em vez de aceitar
    silenciosamente um ficheiro diferente do testado.

.EXAMPLE
    powershell -File scripts\download-ai-model.ps1
#>

$ErrorActionPreference = 'Stop'

$models = @(
    @{
        Name   = 'chat.gguf'
        Url    = 'https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf'
        Sha256 = '74A4DA8C9FDBCD15BD1F6D01D621410D31C6FC00986F5EB687824E7B93D7A9DB'
    },
    @{
        Name   = 'embed.gguf'
        Url    = 'https://huggingface.co/keisuke-miyako/paraphrase-multilingual-mpnet-base-v2-gguf-q8_0/resolve/main/paraphrase-multilingual-mpnet-base-v2-Q8_0.gguf'
        Sha256 = '6FF8B90F0CE3A7AAA53DEB7AC3AA1DF40C924E14204EDB8DA31327D20B2D4FD6'
    }
)

# Mesma pasta de dados que internal/backend/paths.go resolve via
# os.UserConfigDir() + "GestaoDocumental" — manter os dois em sincronia.
$appDir = Join-Path $env:APPDATA 'GestaoDocumental'
$modelsDir = Join-Path $appDir 'models'
New-Item -ItemType Directory -Force -Path $modelsDir | Out-Null

foreach ($model in $models) {
    $destPath = Join-Path $modelsDir $model.Name

    if (Test-Path $destPath) {
        $existingHash = (Get-FileHash -Path $destPath -Algorithm SHA256).Hash
        if ($existingHash -eq $model.Sha256) {
            Write-Host "[$($model.Name)] já presente e com sha256 correto, a saltar."
            continue
        }
        Write-Host "[$($model.Name)] já presente mas com sha256 diferente do esperado — a re-descarregar."
    }

    Write-Host "[$($model.Name)] a descarregar de $($model.Url) ..."
    $tmpPath = "$destPath.download"
    Invoke-WebRequest -Uri $model.Url -OutFile $tmpPath

    $actualHash = (Get-FileHash -Path $tmpPath -Algorithm SHA256).Hash
    if ($actualHash -ne $model.Sha256) {
        Remove-Item -Force $tmpPath
        throw "[$($model.Name)] sha256 não bate: esperado $($model.Sha256), obtido $actualHash. Ficheiro descartado — não confiar num modelo que não bate com o testado."
    }

    Move-Item -Force $tmpPath $destPath
    Write-Host "[$($model.Name)] descarregado e verificado em $destPath"
}

Write-Host ""
Write-Host "Modelos prontos em $modelsDir — a app (LocalBackend.NewLocal) vai encontrá-los automaticamente no próximo arranque."
