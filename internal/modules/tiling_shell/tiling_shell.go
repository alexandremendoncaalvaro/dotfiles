// Package tiling_shell instala e configura a extensao Tiling Shell para auto-tiling no GNOME.
// Tiling Shell gerencia automaticamente os atalhos conflitantes do GNOME via overridden-settings.
package tiling_shell

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/blueprint/internal/gnome"
	"github.com/ale/blueprint/internal/module"
)

const tilingShellUUID = "tilingshell@ferrarodomenico.com"
const forgeUUID = "forge@jmmaranan.com"

var gapSettings = []gnome.DconfEntry{
	{Path: "/org/gnome/shell/extensions/tilingshell/inner-gaps", Value: "uint32 4"},
	{Path: "/org/gnome/shell/extensions/tilingshell/outer-gaps", Value: "uint32 4"},
}

// Module implementa auto-tiling com Tiling Shell.
type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string        { return "tiling-shell" }
func (m *Module) Description() string { return "Auto-tiling Tiling Shell (snap + layouts)" }
func (m *Module) Tags() []string      { return []string{"desktop"} }

func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	return gnome.ShouldRunGuard(sys)
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	return gnome.CheckExtension(ctx, sys, tilingShellUUID, "Tiling Shell")
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

	// 2. Desabilitar Forge se presente
	reporter.Step(2, total, "Verificando Forge...")
	out, _ := sys.Exec(ctx, "gnome-extensions", "show", forgeUUID)
	if strings.Contains(out, forgeUUID) {
		if _, err := sys.Exec(ctx, "gnome-extensions", "disable", forgeUUID); err != nil {
			reporter.Warn("Nao foi possivel desabilitar Forge: " + err.Error())
		} else {
			reporter.Success("Forge desabilitado")
		}
	} else {
		reporter.Info("Forge nao encontrado (ok)")
	}

	// 3. Instalar Tiling Shell se necessario + ativar
	reporter.Step(3, total, "Verificando Tiling Shell...")
	out, _ = sys.Exec(ctx, "gnome-extensions", "show", tilingShellUUID)
	if !strings.Contains(out, tilingShellUUID) {
		reporter.Info("Baixando Tiling Shell de extensions.gnome.org...")
		if err := gnome.InstallFromGnomeExtensions(ctx, sys, tilingShellUUID, gnomeVer, "Tiling Shell"); err != nil {
			return fmt.Errorf("erro ao instalar Tiling Shell: %w", err)
		}
		reporter.Success("Tiling Shell instalado")
	} else {
		reporter.Info("Tiling Shell ja instalado")
	}

	if _, err := sys.Exec(ctx, "gnome-extensions", "enable", tilingShellUUID); err != nil {
		reporter.Warn("Tiling Shell sera ativado apos re-login")
	} else {
		reporter.Success("Tiling Shell ativado")
	}

	// 4. Configurar gaps
	reporter.Step(4, total, "Configurando gaps...")
	if err := gnome.ApplyDconf(ctx, sys, gapSettings); err != nil {
		return fmt.Errorf("erro ao configurar gaps: %w", err)
	}
	reporter.Success("Gaps: inner=4, outer=4")

	reporter.Info("Faca logout e login se o Tiling Shell nao aparecer imediatamente")
	return nil
}
