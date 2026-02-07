// Package gnome_focus configura o modo foco no GNOME:
// F11 (fullscreen) move a janela para um workspace exclusivo (estilo macOS).
// Instala uma extensão GNOME Shell leve que gerencia isso automaticamente.
package gnome_focus

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ale/dotfiles/internal/gnome"
	"github.com/ale/dotfiles/internal/module"
)

const extensionUUID = "focus-mode@dotfiles"

// Module implementa o modo foco via extensão GNOME Shell.
type Module struct {
	// ExtensionSource é o caminho para configs/gnome-extensions/focus-mode@dotfiles/
	ExtensionSource string
}

func New(extensionSource string) *Module {
	return &Module{ExtensionSource: extensionSource}
}

func (m *Module) Name() string        { return "gnome-focus-mode" }
func (m *Module) Description() string { return "Modo foco: F11 = fullscreen + workspace exclusivo" }
func (m *Module) Tags() []string      { return []string{"desktop"} }

func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	return gnome.ShouldRunGuard(sys)
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	// Usa o checker compartilhado para extensao
	status, err := gnome.CheckExtension(ctx, sys, extensionUUID, "Focus-mode")
	if err != nil || status.Kind != module.Installed {
		return status, err
	}

	// Check adicional: dynamic workspaces
	dynWs, _ := sys.Exec(ctx, "dconf", "read", "/org/gnome/mutter/dynamic-workspaces")
	if strings.TrimSpace(dynWs) == "false" {
		return module.Status{Kind: module.Partial, Message: "Workspaces dinamicos desativados"}, nil
	}

	return status, nil
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	total := 3

	// 1. Ativar workspaces dinamicos
	reporter.Step(1, total, "Ativando workspaces dinamicos...")
	if _, err := sys.Exec(ctx, "dconf", "write", "/org/gnome/mutter/dynamic-workspaces", "true"); err != nil {
		return fmt.Errorf("erro ao ativar workspaces dinamicos: %w", err)
	}
	reporter.Success("Workspaces dinamicos ativados")

	// 2. Instalar extensao via symlink
	reporter.Step(2, total, "Instalando extensao focus-mode...")
	extDir := filepath.Join(sys.HomeDir(), ".local", "share", "gnome-shell", "extensions")
	if err := sys.MkdirAll(extDir, 0o755); err != nil {
		return fmt.Errorf("erro ao criar diretorio de extensoes: %w", err)
	}

	dest := filepath.Join(extDir, extensionUUID)
	if err := sys.Symlink(m.ExtensionSource, dest); err != nil {
		return fmt.Errorf("erro ao criar symlink da extensao: %w", err)
	}
	reporter.Success("Extensao instalada via symlink")

	// 3. Ativar
	reporter.Step(3, total, "Ativando extensao...")
	_, err := sys.Exec(ctx, "gnome-extensions", "enable", extensionUUID)
	if err != nil {
		reporter.Warn("Extensao sera ativada apos re-login")
	} else {
		reporter.Success("Focus mode ativo")
	}

	reporter.Info("F11 agora envia a janela para um workspace exclusivo")
	reporter.Info("Faca logout e login se nao funcionar imediatamente")

	return nil
}
