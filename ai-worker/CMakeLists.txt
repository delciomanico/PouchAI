cmake_minimum_required(VERSION 3.21)
project(ai-worker VERSION 0.1.0 LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_C_STANDARD 11)

find_package(Threads REQUIRED)

# pdfio (extração de texto real de PDFs com camada de texto) + zlib (de
# que o pdfio depende para descomprimir streams FlateDecode — ver
# shared/vendor/README.md). Todo o .c do núcleo de cada um entra
# directamente no executável, tal como o sqlite3.c já fazia — nenhum dos
# dois precisa de um alvo CMake próprio.
file(GLOB PDFIO_SOURCES ../shared/vendor/pdfio/*.c)
file(GLOB ZLIB_SOURCES ../shared/vendor/zlib/*.c)

add_executable(ai-worker
    src/main.cpp
    ../shared/vendor/sqlite3/sqlite3.c
    ${PDFIO_SOURCES}
    ${ZLIB_SOURCES}
)

target_include_directories(ai-worker PRIVATE
    ../shared/vendor/sqlite3
    ../shared/vendor/httplib
    ../shared/vendor/pdfio
    ../shared/vendor/zlib
)

target_compile_definitions(ai-worker PRIVATE
    SQLITE_THREADSAFE=1
)

target_link_libraries(ai-worker PRIVATE Threads::Threads)

if(WIN32)
    # cpp-httplib usa APIs (ex. CreateFile2) que só existem a partir do
    # Windows 10 — sem isto o SDK do MinGW assume um Windows mais antigo
    # por omissão e a compilação falha.
    target_compile_definitions(ai-worker PRIVATE _WIN32_WINNT=0x0A00 WINVER=0x0A00)
    target_link_libraries(ai-worker PRIVATE ws2_32)

    if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
        # Liga estaticamente o runtime do MinGW (libstdc++/libgcc/
        # libwinpthread): sem isto o executável depende de encontrar
        # essas DLLs via PATH em tempo de execução — e diferentes
        # instalações do MinGW na mesma máquina (ex. a do Git para
        # Windows vs. a do WinGet) têm versões incompatíveis entre si,
        # o que causa falhas silenciosas (STATUS_ENTRYPOINT_NOT_FOUND)
        # mal o PATH resolva a DLL errada primeiro — visto em primeira
        # mão ao lançar o worker a partir do processo Go do Wails.
        target_link_options(ai-worker PRIVATE -static-libgcc -static-libstdc++ -static)
    endif()
endif()

set_target_properties(ai-worker PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)
