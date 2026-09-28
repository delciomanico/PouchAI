package backend

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
)

// DefaultDBPath, DefaultFilesRoot, DefaultAIWorkerPath e
// DefaultSearchServicePath resolvem os caminhos por omissão partilhados
// pelos dois binários (app.go, o build desktop, e cmd/server, o build
// headless) — ver ROADMAP.md, Fase 6, "dois builds do Go".

func DefaultDBPath() (string, error) {
	appDir, err := defaultAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "app.db"), nil
}

func DefaultFilesRoot() (string, error) {
	appDir, err := defaultAppDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(appDir, "files")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return root, nil
}

func defaultAppDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "GestaoDocumental")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", err
	}
	return appDir, nil
}

// DefaultAIWorkerPath e DefaultSearchServicePath resolvem os binários
// C++ relativos à raiz do projecto — só funcionam a correr a partir daí
// em desenvolvimento (a Fase 8 substitui isto por binários embutidos
// via go:embed). Devolvem "" com um erro descritivo se não encontrarem o
// binário — quem chama decide como degradar (normalmente só logar e
// continuar, ver LocalBackend.NewLocal), mas mantém a mensagem útil que
// já existia desde a Fase 3/5.
func DefaultAIWorkerPath() (string, error) {
	return resolveDevBinary("ai-worker", "ai-worker")
}

func DefaultSearchServicePath() (string, error) {
	return resolveDevBinary("search-service", "search-service")
}

func DefaultLLMServicePath() (string, error) {
	return resolveDevBinary("llm-service", "llm-service")
}

// DefaultAIChatModelPath e DefaultAIEmbedModelPath resolvem os dois GGUF
// dentro da pasta de dados da app (nunca no repositório) — descarregados
// manualmente via scripts/download-ai-model.ps1, nunca pelo build. Como
// os binários C++, devolvem "" com erro se não existirem; quem chama
// decide degradar (LocalBackend.NewLocal não arranca o llm-service).
func DefaultAIChatModelPath() (string, error) {
	return resolveModelFile("chat.gguf")
}

func DefaultAIEmbedModelPath() (string, error) {
	return resolveModelFile("embed.gguf")
}

func resolveModelFile(fileName string) (string, error) {
	appDir, err := defaultAppDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(appDir, "models", fileName)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("modelo não encontrado em %q (correr scripts/download-ai-model.ps1): %w", path, err)
	}
	return path, nil
}

func resolveDevBinary(dir, name string) (string, error) {
	if goruntime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, "build", "bin", name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("binário não encontrado em %q (compilar com `cmake --build %s/build`): %w", path, dir, err)
	}
	return path, nil
}

// DefaultMaxWorkers devolve min(NumCPU, 4) — o mesmo limite usado desde
// a Fase 3.
func DefaultMaxWorkers() int {
	if n := goruntime.NumCPU(); n < 4 {
		return n
	}
	return 4
}

// GenerateToken cria um token aleatório para modo host quando não foi
// dado um explicitamente — suficiente para uma rede local de confiança
// (não há gestão de utilizadores nesta fase, só "conhece o token da
// equipa ou não entra"). Partilhado entre o build desktop (runmode.go)
// e o build servidor (cmd/server) para não duplicar a mesma lógica em
// dois main() diferentes.
func GenerateToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err) // fonte de aleatoriedade do SO indisponível: falhar alto é o correcto aqui
	}
	return hex.EncodeToString(buf)
}
