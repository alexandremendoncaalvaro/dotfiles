package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Mock implementa System para testes.
// Permite configurar respostas para comandos e estado do filesystem.
type Mock struct {
	// ExecResults mapeia "comando arg1 arg2" -> (saida, erro)
	ExecResults map[string]ExecResult

	// Files simula o filesystem em memoria
	Files map[string][]byte

	// Symlinks registra links criados (newname -> oldname)
	Symlinks map[string]string

	// Home define o diretorio home simulado
	Home string

	// Container define se esta em container
	Container bool

	// EnvVars simula variaveis de ambiente
	EnvVars map[string]string

	// Commands define comandos disponiveis
	Commands map[string]bool

	// ExecLog registra todos os comandos executados
	ExecLog []string

	// WriteFileErr faz WriteFile retornar esse erro (se nao nil)
	WriteFileErr error

	// MkdirAllErr faz MkdirAll retornar esse erro (se nao nil)
	MkdirAllErr error

	// SymlinkErr faz Symlink retornar esse erro (se nao nil)
	SymlinkErr error
}

// ExecResult armazena o resultado simulado de um comando.
type ExecResult struct {
	Output string
	Err    error
}

// NewMock cria um Mock com valores padrao.
func NewMock() *Mock {
	return &Mock{
		ExecResults: make(map[string]ExecResult),
		Files:       make(map[string][]byte),
		Symlinks:    make(map[string]string),
		Home:        "/home/test",
		EnvVars:     make(map[string]string),
		Commands:    make(map[string]bool),
	}
}

func (m *Mock) Exec(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	key = strings.TrimSpace(key)
	m.ExecLog = append(m.ExecLog, key)

	if result, ok := m.ExecResults[key]; ok {
		return result.Output, result.Err
	}
	return "", nil
}

func (m *Mock) ExecStream(_ context.Context, callback func(line string), name string, args ...string) error {
	key := name + " " + strings.Join(args, " ")
	key = strings.TrimSpace(key)
	m.ExecLog = append(m.ExecLog, key)

	if result, ok := m.ExecResults[key]; ok {
		if result.Output != "" {
			for _, line := range strings.Split(result.Output, "\n") {
				callback(line)
			}
		}
		return result.Err
	}
	return nil
}

func (m *Mock) FileExists(path string) bool {
	if _, ok := m.Files[path]; ok {
		return true
	}
	if _, ok := m.Symlinks[path]; ok {
		return true
	}
	return false
}

func (m *Mock) ReadFile(path string) ([]byte, error) {
	data, ok := m.Files[path]
	if !ok {
		return nil, fmt.Errorf("arquivo nao encontrado: %s", path)
	}
	return data, nil
}

func (m *Mock) WriteFile(path string, data []byte, _ os.FileMode) error {
	if m.WriteFileErr != nil {
		return m.WriteFileErr
	}
	m.Files[path] = data
	return nil
}

func (m *Mock) MkdirAll(_ string, _ os.FileMode) error {
	if m.MkdirAllErr != nil {
		return m.MkdirAllErr
	}
	return nil
}

func (m *Mock) Symlink(oldname, newname string) error {
	if m.SymlinkErr != nil {
		return m.SymlinkErr
	}
	m.Symlinks[newname] = oldname
	return nil
}

func (m *Mock) HomeDir() string {
	return m.Home
}

func (m *Mock) IsContainer() bool {
	return m.Container
}

func (m *Mock) Env(key string) string {
	return m.EnvVars[key]
}

func (m *Mock) CommandExists(name string) bool {
	return m.Commands[name]
}

func (m *Mock) AppendToFileIfMissing(path, line string) (bool, error) {
	data, ok := m.Files[path]
	if ok && strings.Contains(string(data), line) {
		return false, nil
	}

	// Garante diretorio pai
	dir := filepath.Dir(path)
	_ = m.MkdirAll(dir, 0o755)

	existing := string(data)
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		existing += "\n"
	}
	existing += line + "\n"
	m.Files[path] = []byte(existing)
	return true, nil
}
