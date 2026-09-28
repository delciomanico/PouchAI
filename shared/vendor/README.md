# Dependências C++ vendored

Vendored (copiadas para o repositório) em vez de geridas por um gestor de
pacotes, para que `ai-worker` e `search-service` compilem com só CMake +
um compilador C++20 — sem depender de vcpkg/Conan/pacman estarem
instalados na máquina de quem compila (ver Fase 8, empacotamento).

| Pasta | Origem | Versão | Licença |
|---|---|---|---|
| `sqlite3/` | [sqlite.org, amalgamation](https://www.sqlite.org/download.html) | 3.53.4 (2026/sqlite-amalgamation-3530400.zip) | Public domain |
| `httplib/` | [yhirose/cpp-httplib](https://github.com/yhirose/cpp-httplib), `httplib.h` | commit da branch `master` em 2026-09-27 | MIT |
| `llama.cpp/` | [ggml-org/llama.cpp](https://github.com/ggml-org/llama.cpp) | tag `v0.5.0`, commit `7fe450e19305b828c199d602c23a8337aaa1f03b` (2026-09-27) | MIT |
| `pdfio/` | [michaelrsweet/pdfio](https://github.com/michaelrsweet/pdfio) | tag `v1.6.5`, commit `5102eddff28d9b605f3e6783d50ad380f61f692e` | Apache-2.0 |
| `zlib/` | [madler/zlib](https://github.com/madler/zlib) | tag `v1.3.2` | zlib |

`llama.cpp/` é um recorte, não o repositório inteiro: só `ggml/`, `src/`,
`include/`, `cmake/`, `vendor/` (exigido sem condição pelo `CMakeLists.txt`
de topo deles, "mtmd needs these even when common is not built" — dentro
tirámos só o `vendor/cpp-httplib/`, que só é usado quando
`LLAMA_BUILD_COMMON` está ligado, e nós desligamos), `CMakeLists.txt` e
`LICENSE` — sem `examples/`,
`tools/`, `tests/`, `common/` (código dos exemplos/CLI deles, que não
usamos, ver `llm-service/`) nem os ficheiros Python de conversão de
modelos. Dentro de `ggml/src/`, também foram removidos os backends que a
app não usa (`ggml-cuda`, `ggml-vulkan`, `ggml-metal`, `ggml-sycl`,
`ggml-opencl`, `ggml-hip`, `ggml-cann`, `ggml-musa`, `ggml-hexagon`,
`ggml-openvino`, `ggml-virtgpu`, `ggml-webgpu`, `ggml-zdnn`, `ggml-zendnn`,
`ggml-et`, `ggml-rpc`, `ggml-blas`) — fica só `ggml-cpu`, inferência 100%
em CPU, sem GPU nem serviços externos (ver `llm-service/CMakeLists.txt`
para as flags `GGML_*=OFF` que tornam isto seguro mesmo que o código
tivesse ficado). O ficheiro `.git` do clone original nunca foi copiado —
isto é uma cópia de ficheiros, não um submódulo.

`pdfio/` (usado pelo `ai-worker` para extrair texto real de PDFs com
camada de texto, ver `ai-worker/src/main.cpp`) é todos os `.c`/`.h` do
núcleo da biblioteca, sem `testpdfio.c`/`testttf.c`/`test.h` (o arnês de
testes deles, não precisamos). Depende de `zlib/` (inclui
`<zlib.h>` sem guarda condicional em `pdfio-private.h` — é uma dependência
obrigatória, não opcional) para descomprimir streams `FlateDecode`; o
suporte a PNG (`HAVE_LIBPNG`) fica desligado (não definimos essa macro),
o que só desactiva funções de *escrever* imagens PNG num PDF novo —
não usamos essas funções, só lemos PDFs existentes, por isso não faz
falta `libpng` nenhuma.

`zlib/` é o código-fonte principal (ficheiros `.c`/`.h` de topo, sem `contrib/`,
`test/`, `examples/`) tal como está nos outros vendors — nada de MinGW
específico a fazer aqui, zlib compila e liga estaticamente sem problemas
conhecidos.

Para atualizar: descarregar de novo a partir da origem e substituir os
ficheiros — não editar à mão. Para o `llama.cpp/`, repetir o mesmo recorte
(sparse-checkout de `ggml src include cmake vendor` + apagar os backends
não usados de `ggml/src/` + apagar `vendor/cpp-httplib/`) em vez de copiar
a árvore inteira.
