cmake_minimum_required(VERSION 3.21)
project(search-service VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

find_package(Threads REQUIRED)

add_executable(search-service
    src/main.cpp
)

target_include_directories(search-service PRIVATE
    ../shared/vendor/httplib
)

target_link_libraries(search-service PRIVATE Threads::Threads)

if(WIN32)
    # Ver ai-worker/CMakeLists.txt para a explicação destas duas flags —
    # o mesmo cpp-httplib, os mesmos dois problemas (API só existe a
    # partir do Windows 10; DLLs do MinGW incompatíveis entre
    # instalações na mesma máquina).
    target_compile_definitions(search-service PRIVATE _WIN32_WINNT=0x0A00 WINVER=0x0A00)
    target_link_libraries(search-service PRIVATE ws2_32)

    if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
        target_link_options(search-service PRIVATE -static-libgcc -static-libstdc++ -static)
    endif()
endif()

set_target_properties(search-service PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)
