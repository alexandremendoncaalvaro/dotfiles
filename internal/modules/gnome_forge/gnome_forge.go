// Package gnome_forge instala e configura a extensão Forge para auto-tiling no GNOME.
// Configura SUPER+setas para controlar foco, mover e redimensionar janelas.
package gnome_forge

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/dotfiles/internal/gnome"
	"github.com/ale/dotfiles/internal/module"
)

const forgeUUID = "forge@jmmaranan.com"

// Keybindings: Super+setas (foco), Super+Shift (mover), Super+Ctrl (resize).
// Primeiro limpa os defaults do GNOME que conflitam.
var forgeKeybindings = []gnome.DconfEntry{
	// Desabilita defaults do GNOME que conflitam com Forge
	{"/org/gnome/desktop/wm/keybindings/maximize", "@as []"},
	{"/org/gnome/desktop/wm/keybindings/unmaximize", "@as []"},
	{"/org/gnome/mutter/keybindings/toggle-tiled-left", "@as []"},
	{"/org/gnome/mutter/keybindings/toggle-tiled-right", "@as []"},

	// Foco: Super + setas
	{"/org/gnome/shell/extensions/forge/keybindings/window-focus-left", "['<Super>Left']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-focus-right", "['<Super>Right']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-focus-up", "['<Super>Up']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-focus-down", "['<Super>Down']"},

	// Mover/swap: Super + Shift + setas
	{"/org/gnome/shell/extensions/forge/keybindings/window-swap-left", "['<Super><Shift>Left']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-swap-right", "['<Super><Shift>Right']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-swap-up", "['<Super><Shift>Up']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-swap-down", "['<Super><Shift>Down']"},

	// Resize: Super + Ctrl + setas
	{"/org/gnome/shell/extensions/forge/keybindings/window-resize-left", "['<Super><Control>Left']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-resize-right", "['<Super><Control>Right']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-resize-up", "['<Super><Control>Up']"},
	{"/org/gnome/shell/extensions/forge/keybindings/window-resize-down", "['<Super><Control>Down']"},

	// Split toggle: Super + g
	{"/org/gnome/shell/extensions/forge/keybindings/con-split-layout-toggle", "['<Super>g']"},

	// Toggle tiling: Super + y
	{"/org/gnome/shell/extensions/forge/keybindings/prefs-tiling-toggle", "['<Super>y']"},
}

// Module implementa auto-tiling com Forge.
type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string        { return "gnome-forge" }
func (m *Module) Description() string { return "Auto-tiling Forge (Super+setas para janelas e foco)" }
func (m *Module) Tags() []string      { return []string{"desktop"} }

func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	return gnome.ShouldRunGuard(sys)
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	return gnome.CheckExtension(ctx, sys, forgeUUID, "Forge")
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	total := 4

	// 1. Detectar versao do GNOME
	reporter.Step(1, total, "Detectando versao do GNOME...")
	gnomeVer, err := gnome.DetectVersion(ctx, sys)
	if err != nil {
		return err
	}
	reporter.Info(fmt.Sprintf("GNOME Shell %s", gnomeVer))

	// 2. Instalar Forge se necessario
	reporter.Step(2, total, "Verificando Forge...")
	out, _ := sys.Exec(ctx, "gnome-extensions", "show", forgeUUID)
	if !strings.Contains(out, forgeUUID) {
		reporter.Info("Baixando Forge de extensions.gnome.org...")
		if err := gnome.InstallFromGnomeExtensions(ctx, sys, forgeUUID, gnomeVer, "Forge"); err != nil {
			return fmt.Errorf("erro ao instalar Forge: %w", err)
		}
		reporter.Success("Forge instalado")
	} else {
		reporter.Info("Forge ja instalado")
	}

	// 3. Ativar
	reporter.Step(3, total, "Ativando Forge...")
	if _, err := sys.Exec(ctx, "gnome-extensions", "enable", forgeUUID); err != nil {
		reporter.Warn("Forge sera ativado apos re-login")
	} else {
		reporter.Success("Forge ativado")
	}

	// 4. Configurar atalhos
	reporter.Step(4, total, "Configurando atalhos...")
	if err := gnome.ApplyDconf(ctx, sys, forgeKeybindings); err != nil {
		return fmt.Errorf("erro ao configurar atalhos: %w", err)
	}
	reporter.Success("Super+setas: foco | Super+Shift: mover | Super+Ctrl: resize")

	reporter.Info("Faca logout e login se o Forge nao aparecer imediatamente")
	return nil
}
