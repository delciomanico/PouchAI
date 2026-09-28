cmake_minimum_required(VERSION 3.21)
project(llm-service VERSION 0.1.0 LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(Threads REQUIRED)

# Flags do vendored llama.cpp/ggml (ver shared/vendor/README.md) — têm de
# ser fixadas ANTES do add_subdirectory, porque são `option()` lá dentro
# (só pegam no valor de cache se ainda não tiver sido definido).
#
#   GGML_NATIVE=OFF   — sem isto o build fica -march=native, optimizado
#                        só para o CPU da máquina que compila; corre bem
#                        aqui e crasha com instrução ilegal na máquina de
#                        um utilizador com um CPU diferente.
#   GGML_OPENMP=OFF   — o link -static já usado nos outros dois binários
#                        (ver comentário mais abaixo) não combina bem com
#                        libgomp estático no MinGW (pode nem existir uma
#                        libgomp.a estática); o ggml cai para o seu
#                        thread-pool interno em pthreads, que já ligamos.
#   GGML_CUDA/VULKAN/METAL=OFF, LLAMA_CURL=OFF, LLAMA_BUILD_*=OFF
#                     — nada disto é preciso (inferência só em CPU,
#                        escrevemos o nosso próprio main.cpp com httplib
#                        como o search-service já faz); o código-fonte
#                        destes backends nem sequer foi vendored (ver
#                        shared/vendor/README.md), isto é só uma segunda
#                        camada de segurança caso uma actualização futura
#                        volte a trazer esses directórios.
set(GGML_NATIVE OFF CACHE BOOL "" FORCE)
set(GGML_OPENMP OFF CACHE BOOL "" FORCE)
set(GGML_CUDA OFF CACHE BOOL "" FORCE)
set(GGML_VULKAN OFF CACHE BOOL "" FORCE)
set(GGML_METAL OFF CACHE BOOL "" FORCE)
set(LLAMA_CURL OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_COMMON OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_TESTS OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_TOOLS OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
set(LLAMA_BUILD_SERVER OFF CACHE BOOL "" FORCE)
set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)

add_subdirectory(../shared/vendor/llama.cpp llama.cpp)

add_executable(llm-service
    src/main.cpp
)

target_include_directories(llm-service PRIVATE
    ../shared/vendor/httplib
)

target_link_libraries(llm-service PRIVATE Threads::Threads llama)

if(WIN32)
    # Ver ai-worker/CMakeLists.txt para a explicação destas duas flags —
    # o mesmo cpp-httplib, os mesmos dois problemas (API só existe a
    # partir do Windows 10; DLLs do MinGW incompatíveis entre
    # instalações na mesma máquina). Aqui importa ainda mais manter o
    # link 100% estático: é o único binário dos três que carrega um
    # modelo de vários megabytes, e não queremos que isso dependa de
    # nenhuma DLL externa (incluindo as do próprio llama.cpp) andar a
    # ser encontrada em tempo de execução.
    target_compile_definitions(llm-service PRIVATE _WIN32_WINNT=0x0A00 WINVER=0x0A00)
    target_link_libraries(llm-service PRIVATE ws2_32)

    if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
        target_link_options(llm-service PRIVATE -static-libgcc -static-libstdc++ -static)
    endif()
endif()

set_target_properties(llm-service PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)
