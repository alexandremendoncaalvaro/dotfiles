package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Real implementa System usando chamadas reais ao SO.
type Real struct{}

// NewReal cria uma implementacao real do System.
func NewReal() *Real {
	return &Real{}
}

func (r *Real) Exec(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (r *Real) ExecStream(ctx context.Context, callback func(line string), name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("erro ao criar pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar comando: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		callback(scanner.Text())
	}

	return cmd.Wait()
}

func (r *Real) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *Real) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (r *Real) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (r *Real) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *Real) Symlink(oldname, newname string) error {
	// Remove link existente se houver
	if _, err := os.Lstat(newname); err == nil {
		if err := os.Remove(newname); err != nil {
			return fmt.Errorf("erro ao remover link existente: %w", err)
		}
	}
	return os.Symlink(oldname, newname)
}

func (r *Real) HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}

func (r *Real) IsContainer() bool {
	// Verifica indicadores comuns de container
	for _, f := range []string{"/run/.containerenv", "/.dockerenv"} {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	// Verifica variaveis de ambiente de devcontainer
	if os.Getenv("REMOTE_CONTAINERS") != "" || os.Getenv("CODESPACES") != "" {
		return true
	}
	return false
}

func (r *Real) IsWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "microsoft")
}

func (r *Real) Env(key string) string {
	return os.Getenv(key)
}

func (r *Real) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (r *Real) AppendToFileIfMissing(path, line string) (bool, error) {
	// Garante que o diretorio pai existe
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("erro ao criar diretorio %s: %w", dir, err)
	}

	// Le conteudo existente
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("erro ao ler %s: %w", path, err)
	}

	// Verifica se a linha ja existe
	if strings.Contains(string(data), line) {
		return false, nil
	}

	// Abre arquivo para append (cria se nao existir)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, fmt.Errorf("erro ao abrir %s: %w", path, err)
	}
	defer f.Close()

	// Adiciona newline antes se o arquivo nao termina com um
	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return false, err
		}
	}

	if _, err := f.WriteString(line + "\n"); err != nil {
		return false, err
	}

	return true, nil
}
