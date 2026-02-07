// Package gnome_forge instala e configura a extensão Forge para auto-tiling no GNOME.
// Configura SUPER+setas para controlar foco, mover e redimensionar janelas.
package gnome_forge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ale/dotfiles/internal/module"
)

const forgeUUID = "forge@jmmaranan.com"

// dconfEntry representa uma chave/valor no dconf.
type dconfEntry struct {
	path  string
	value string
}

// Keybindings: Super+setas (foco), Super+Shift (mover), Super+Ctrl (resize).
// Primeiro limpa os defaults do GNOME que conflitam.
var forgeKeybindings = []dconfEntry{
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
	if sys.IsContainer() {
		return false, "dentro de container"
	}
	if sys.Env("WAYLAND_DISPLAY") == "" && sys.Env("DISPLAY") == "" {
		return false, "sem sessao grafica"
	}
	if !sys.CommandExists("gnome-extensions") {
		return false, "gnome-extensions nao disponivel"
	}
	return true, ""
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	out, err := sys.Exec(ctx, "gnome-extensions", "show", forgeUUID)
	if err != nil {
		return module.Status{Kind: module.Missing, Message: "Forge nao instalado"}, nil
	}
	if strings.Contains(out, "ENABLED") {
		return module.Status{Kind: module.Installed, Message: "Forge instalado e ativo"}, nil
	}
	return module.Status{Kind: module.Partial, Message: "Forge instalado mas desativado"}, nil
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	total := 4

	// 1. Detectar versao do GNOME
	reporter.Step(1, total, "Detectando versao do GNOME...")
	gnomeVer, err := detectGnomeVersion(ctx, sys)
	if err != nil {
		return err
	}
	reporter.Info(fmt.Sprintf("GNOME Shell %s", gnomeVer))

	// 2. Instalar Forge se necessario
	reporter.Step(2, total, "Verificando Forge...")
	out, _ := sys.Exec(ctx, "gnome-extensions", "show", forgeUUID)
	if !strings.Contains(out, forgeUUID) {
		reporter.Info("Baixando Forge de extensions.gnome.org...")
		if err := m.installFromGnomeExtensions(ctx, sys, gnomeVer); err != nil {
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
	if err := applyDconf(ctx, sys, forgeKeybindings); err != nil {
		return fmt.Errorf("erro ao configurar atalhos: %w", err)
	}
	reporter.Success("Super+setas: foco | Super+Shift: mover | Super+Ctrl: resize")

	reporter.Info("Faca logout e login se o Forge nao aparecer imediatamente")
	return nil
}

// extensionInfo representa a resposta da API do extensions.gnome.org.
type extensionInfo struct {
	DownloadURL string `json:"download_url"`
}

func (m *Module) installFromGnomeExtensions(ctx context.Context, sys module.System, gnomeVer string) error {
	apiURL := fmt.Sprintf(
		"https://extensions.gnome.org/extension-info/?uuid=%s&shell_version=%s",
		forgeUUID, gnomeVer,
	)

	jsonOut, err := sys.Exec(ctx, "curl", "-sfL", apiURL)
	if err != nil {
		return fmt.Errorf("erro ao consultar extensions.gnome.org: %w", err)
	}

	var info extensionInfo
	if err := json.Unmarshal([]byte(jsonOut), &info); err != nil {
		return fmt.Errorf("resposta inesperada da API: %w", err)
	}
	if info.DownloadURL == "" {
		return fmt.Errorf("Forge nao disponivel para GNOME Shell %s", gnomeVer)
	}

	downloadURL := "https://extensions.gnome.org" + info.DownloadURL
	zipPath := "/tmp/forge-extension.zip"

	if _, err := sys.Exec(ctx, "curl", "-sfL", "-o", zipPath, downloadURL); err != nil {
		return fmt.Errorf("erro ao baixar Forge: %w", err)
	}

	if _, err := sys.Exec(ctx, "gnome-extensions", "install", "--force", zipPath); err != nil {
		return fmt.Errorf("erro ao instalar extensao: %w", err)
	}

	return nil
}

func detectGnomeVersion(ctx context.Context, sys module.System) (string, error) {
	out, err := sys.Exec(ctx, "gnome-shell", "--version")
	if err != nil {
		return "", fmt.Errorf("gnome-shell nao encontrado: %w", err)
	}
	// "GNOME Shell 46.2" → "46"
	parts := strings.Fields(out)
	if len(parts) < 3 {
		return "", fmt.Errorf("saida inesperada: %s", out)
	}
	ver := parts[2]
	if dot := strings.Index(ver, "."); dot > 0 {
		ver = ver[:dot]
	}
	return ver, nil
}

func applyDconf(ctx context.Context, sys module.System, entries []dconfEntry) error {
	for _, e := range entries {
		if _, err := sys.Exec(ctx, "dconf", "write", e.path, e.value); err != nil {
			return fmt.Errorf("dconf write %s: %w", e.path, err)
		}
	}
	return nil
}
