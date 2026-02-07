// Package cedilla corrige a cedilha no Bluefin (Wayland/GNOME).
// Configura ~/.XCompose com regras de Compose para ' + c => ç.
package cedilla

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/dotfiles/internal/module"
)

const (
	beginMarker = "# BEGIN BLUEFIN CEDILLA"
	endMarker   = "# END BLUEFIN CEDILLA"

	composeRules = `
# BEGIN BLUEFIN CEDILLA
<dead_acute> <c> : "ç"
<dead_acute> <C> : "Ç"
# END BLUEFIN CEDILLA`
)

// Module implementa o fix de cedilha.
type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string        { return "cedilla-fix" }
func (m *Module) Description() string { return "Correcao de cedilha para Bluefin (Wayland/GNOME)" }
func (m *Module) Tags() []string      { return []string{"desktop"} }

// ShouldRun retorna false dentro de containers (configuracao de desktop).
func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	if sys.IsContainer() {
		return false, "dentro de container (configuracao de desktop)"
	}
	return true, ""
}

func (m *Module) Check(_ context.Context, sys module.System) (module.Status, error) {
	xcompose := sys.HomeDir() + "/.XCompose"

	if !sys.FileExists(xcompose) {
		return module.Status{Kind: module.Missing, Message: "~/.XCompose nao existe"}, nil
	}

	data, err := sys.ReadFile(xcompose)
	if err != nil {
		return module.Status{}, fmt.Errorf("erro ao ler ~/.XCompose: %w", err)
	}

	content := string(data)
	if strings.Contains(content, beginMarker) && strings.Contains(content, endMarker) {
		return module.Status{Kind: module.Installed, Message: "Regras de cedilha configuradas"}, nil
	}

	return module.Status{Kind: module.Missing, Message: "Regras de cedilha ausentes"}, nil
}

func (m *Module) Apply(_ context.Context, sys module.System, reporter module.Reporter) error {
	xcompose := sys.HomeDir() + "/.XCompose"

	// 1. Verificar sessao e layout (informativo, nao bloqueia)
	reporter.Step(1, 3, "Verificando ambiente...")
	if session := sys.Env("XDG_SESSION_TYPE"); session == "wayland" {
		reporter.Info("Sessao Wayland detectada")
	} else {
		reporter.Warn(fmt.Sprintf("Sessao nao e Wayland (tipo: %s)", session))
	}

	// 2. Preparar ~/.XCompose
	reporter.Step(2, 3, "Preparando ~/.XCompose...")
	var content string
	if sys.FileExists(xcompose) {
		data, err := sys.ReadFile(xcompose)
		if err != nil {
			return fmt.Errorf("erro ao ler ~/.XCompose: %w", err)
		}
		content = string(data)

		// Remove bloco antigo se existir
		content = removeBlock(content, beginMarker, endMarker)
	} else {
		// Cria com include padrao
		content = `include "%L"` + "\n"
		reporter.Info("Criando ~/.XCompose")
	}

	// Garante include "%L" no inicio
	if !strings.Contains(content, `include "%L"`) {
		content = `include "%L"` + "\n\n" + content
	}

	// 3. Adicionar regras
	reporter.Step(3, 3, "Adicionando regras de cedilha...")
	content = strings.TrimRight(content, "\n") + "\n" + composeRules + "\n"

	if err := sys.WriteFile(xcompose, []byte(content), 0o644); err != nil {
		return fmt.Errorf("erro ao escrever ~/.XCompose: %w", err)
	}

	reporter.Success("Regras de cedilha configuradas")
	reporter.Info("Faca logout e login para aplicar as mudancas")

	return nil
}

// removeBlock remove um bloco marcado com BEGIN/END do conteudo.
func removeBlock(content, begin, end string) string {
	startIdx := strings.Index(content, begin)
	if startIdx == -1 {
		return content
	}

	endIdx := strings.Index(content, end)
	if endIdx == -1 {
		return content
	}

	endIdx += len(end)
	// Remove newline apos o end marker
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	return content[:startIdx] + content[endIdx:]
}
