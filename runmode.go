package main

import "flag"

// Os três modos do build desktop (ROADMAP.md, Fase 6). O binário
// separado cmd/server (headless, "build servidor") é sempre
// equivalente a "host" — não tem os outros dois modos porque não tem
// janela nenhuma para mostrar dados locais sozinho.
const (
	runModeSolo   = "solo"   // omissão: LocalBackend, sem rede nenhuma — comportamento de sempre (Fases 1-5)
	runModeHost   = "host"   // LocalBackend + expõe a API HTTP para clientes (internal/apiserver)
	runModeClient = "client" // RemoteBackend — todos os dados vêm de outra instância pela rede
)

type runMode struct {
	kind       string
	listenAddr string // modo host
	hostURL    string // modo cliente
	token      string // host: se vazio, gera-se um; cliente: obrigatório
}

// parseRunMode lê as flags de arranque — usadas para testar modo
// equipa localmente com `wails dev -appargs "--mode=host ..."` (ver
// memória da Fase 6), e para quem instalar a app manualmente escolher
// o papel de cada instância.
func parseRunMode() runMode {
	mode := flag.String("mode", runModeSolo, "solo | host | client")
	listen := flag.String("listen", "0.0.0.0:8790", "endereço a escutar em modo host, ex. 0.0.0.0:8790")
	hostURL := flag.String("host", "", "URL do host em modo cliente, ex. http://192.168.1.50:8790")
	token := flag.String("token", "", "token partilhado da equipa (obrigatório em modo cliente; opcional em modo host — gera-se um se vazio)")
	flag.Parse()

	return runMode{kind: *mode, listenAddr: *listen, hostURL: *hostURL, token: *token}
}
