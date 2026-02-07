// Package gnome fornece helpers compartilhados para modulos de extensoes GNOME Shell.
// Elimina duplicacao de logica de guard, check, install e detect entre modulos.
package gnome

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ale/dotfiles/internal/module"
)

// DconfEntry representa uma chave/valor no dconf.
type DconfEntry struct {
	Path  string
	Value string
}

// extensionInfo representa a resposta da API do extensions.gnome.org.
type extensionInfo struct {
	DownloadURL string `json:"download_url"`
}

// ShouldRunGuard verifica as condicoes necessarias para executar um modulo GNOME:
// nao estar em container, ter sessao grafica, e ter gnome-extensions disponivel.
func ShouldRunGuard(sys module.System) (bool, string) {
	if sys.IsContainer() {
		return false, "dentro de container"
	}
	if sys.Env("WAYLAND_DISPLAY") == "" && sys.Env("DISPLAY") == "" {
		return false, "sem sessao grafica (faca login no desktop primeiro)"
	}
	if !sys.CommandExists("gnome-extensions") {
		return false, "gnome-extensions nao disponivel (requer GNOME Shell)"
	}
	return true, ""
}

// CheckExtension verifica o estado de uma extensao GNOME Shell pelo UUID.
// Retorna Installed, Missing ou Partial conforme o estado real.
func CheckExtension(ctx context.Context, sys module.System, uuid, displayName string) (module.Status, error) {
	out, err := sys.Exec(ctx, "gnome-extensions", "show", uuid)
	if err != nil {
		return module.Status{Kind: module.Missing, Message: displayName + " nao instalado"}, nil
	}

	if !strings.Contains(out, "Enabled: Yes") {
		return module.Status{Kind: module.Partial, Message: displayName + " instalado mas desativado"}, nil
	}

	if strings.Contains(out, "OUT OF DATE") {
		return module.Status{Kind: module.Partial, Message: displayName + " desatualizado (faca logout/login ou atualize a extensao)"}, nil
	}

	if strings.Contains(out, "ERROR") {
		return module.Status{Kind: module.Partial, Message: displayName + " com erro (verifique logs: journalctl -f -o cat /usr/bin/gnome-shell)"}, nil
	}

	return module.Status{Kind: module.Installed, Message: displayName + " instalado e ativo"}, nil
}

// DetectVersion retorna a versao major do GNOME Shell (ex: "46").
func DetectVersion(ctx context.Context, sys module.System) (string, error) {
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

// InstallFromGnomeExtensions baixa e instala uma extensao do extensions.gnome.org.
func InstallFromGnomeExtensions(ctx context.Context, sys module.System, uuid, gnomeVer, displayName string) error {
	apiURL := fmt.Sprintf(
		"https://extensions.gnome.org/extension-info/?uuid=%s&shell_version=%s",
		uuid, gnomeVer,
	)

	jsonOut, err := sys.Exec(ctx, "curl", "-sfL", apiURL)
	if err != nil {
		return fmt.Errorf("erro ao consultar extensions.gnome.org (verifique sua conexao com a internet): %w", err)
	}

	var info extensionInfo
	if err := json.Unmarshal([]byte(jsonOut), &info); err != nil {
		return fmt.Errorf("resposta inesperada da API: %w", err)
	}
	if info.DownloadURL == "" {
		return fmt.Errorf("%s nao disponivel para GNOME Shell %s — verifique se ha uma versao compativel em https://extensions.gnome.org", displayName, gnomeVer)
	}

	downloadURL := "https://extensions.gnome.org" + info.DownloadURL
	zipPath := fmt.Sprintf("/tmp/%s.zip", uuid)

	if _, err := sys.Exec(ctx, "curl", "-sfL", "-o", zipPath, downloadURL); err != nil {
		return fmt.Errorf("erro ao baixar %s: %w", displayName, err)
	}

	if _, err := sys.Exec(ctx, "gnome-extensions", "install", "--force", zipPath); err != nil {
		return fmt.Errorf("erro ao instalar extensao: %w", err)
	}

	return nil
}

// ApplyDconf escreve uma lista de chaves dconf.
func ApplyDconf(ctx context.Context, sys module.System, entries []DconfEntry) error {
	for _, e := range entries {
		if _, err := sys.Exec(ctx, "dconf", "write", e.Path, e.Value); err != nil {
			return fmt.Errorf("dconf write %s: %w", e.Path, err)
		}
	}
	return nil
}
