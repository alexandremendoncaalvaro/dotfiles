// Package clipboard_indicator instala e ativa a extensão Clipboard Indicator (Tudmotu) no GNOME.
// Gerenciador de clipboard com histórico, busca e atalhos.
package clipboard_indicator

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/dotfiles/internal/gnome"
	"github.com/ale/dotfiles/internal/module"
)

const extensionUUID = "clipboard-indicator@tudmotu.com"

// Module implementa a instalação do Clipboard Indicator.
type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string        { return "clipboard-indicator" }
func (m *Module) Description() string { return "Clipboard Indicator (historico de clipboard no GNOME)" }
func (m *Module) Tags() []string      { return []string{"desktop"} }

func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	return gnome.ShouldRunGuard(sys)
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	return gnome.CheckExtension(ctx, sys, extensionUUID, "Clipboard Indicator")
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	total := 3

	// 1. Detectar versão do GNOME
	reporter.Step(1, total, "Detectando versao do GNOME...")
	gnomeVer, err := gnome.DetectVersion(ctx, sys)
	if err != nil {
		return err
	}
	reporter.Info(fmt.Sprintf("GNOME Shell %s", gnomeVer))

	// 2. Instalar se necessário
	reporter.Step(2, total, "Verificando Clipboard Indicator...")
	out, _ := sys.Exec(ctx, "gnome-extensions", "show", extensionUUID)
	if !strings.Contains(out, extensionUUID) {
		reporter.Info("Baixando Clipboard Indicator de extensions.gnome.org...")
		if err := gnome.InstallFromGnomeExtensions(ctx, sys, extensionUUID, gnomeVer, "Clipboard Indicator"); err != nil {
			return fmt.Errorf("erro ao instalar Clipboard Indicator: %w", err)
		}
		reporter.Success("Clipboard Indicator instalado")
	} else {
		reporter.Info("Clipboard Indicator ja instalado")
	}

	// 3. Ativar
	reporter.Step(3, total, "Ativando Clipboard Indicator...")
	if _, err := sys.Exec(ctx, "gnome-extensions", "enable", extensionUUID); err != nil {
		reporter.Warn("Clipboard Indicator sera ativado apos re-login")
	} else {
		reporter.Success("Clipboard Indicator ativo")
	}

	reporter.Info("Faca logout e login se nao aparecer imediatamente")
	return nil
}
