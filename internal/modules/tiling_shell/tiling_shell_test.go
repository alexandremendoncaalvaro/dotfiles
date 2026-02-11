package tiling_shell

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ale/blueprint/internal/module"
	"github.com/ale/blueprint/internal/module/moduletest"
	"github.com/ale/blueprint/internal/system"
)

func TestShouldRun_SkipInContainer(t *testing.T) {
	mock := system.NewMock()
	mock.Container = true

	mod := New()
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular em container")
	}
}

func TestShouldRun_SkipWithoutDisplay(t *testing.T) {
	mock := system.NewMock()
	// Sem DISPLAY e sem WAYLAND_DISPLAY

	mod := New()
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular sem sessao grafica")
	}
}

func TestShouldRun_RunOnGnomeDesktop(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"
	mock.Commands["gnome-extensions"] = true

	mod := New()
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if !ok {
		t.Error("deveria rodar em desktop GNOME")
	}
}

func TestCheck_Missing(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing, obteve %s", status.Kind)
	}
}

func TestCheck_Installed(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Output: "tilingshell@ferrarodomenico.com\n  Name: Tiling Shell\n  Enabled: Yes\n  State: ACTIVE\n",
	}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestShouldRun_SkipWithoutGnomeExtensions(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"
	// gnome-extensions nao disponivel

	mod := New()
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular sem gnome-extensions")
	}
}

func TestApply_AlreadyInstalled(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 47.2"}
	// Forge nao presente
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}
	// Tiling Shell ja instalado
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Output: "tilingshell@ferrarodomenico.com\n  Name: Tiling Shell\n  Enabled: Yes\n  State: ACTIVE\n",
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica que dconf write foi chamado (gaps)
	hasDconf := false
	for _, cmd := range mock.ExecLog {
		if strings.Contains(cmd, "dconf write") && strings.Contains(cmd, "tilingshell") {
			hasDconf = true
			break
		}
	}
	if !hasDconf {
		t.Error("esperava chamadas ao dconf para configurar gaps")
	}

	// Verifica que Forge disable NAO foi chamado
	for _, cmd := range mock.ExecLog {
		if strings.Contains(cmd, "disable") && strings.Contains(cmd, "forge") {
			t.Error("nao deveria tentar desabilitar Forge quando nao presente")
		}
	}
}

func TestApply_DisablesForge(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 47.2"}
	// Forge presente
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Name: Forge\n  Enabled: Yes\n  State: ACTIVE\n",
	}
	// Tiling Shell ja instalado
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Output: "tilingshell@ferrarodomenico.com\n  Name: Tiling Shell\n  Enabled: Yes\n  State: ACTIVE\n",
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica que Forge disable foi chamado
	forgeDisabled := false
	for _, cmd := range mock.ExecLog {
		if strings.Contains(cmd, "disable") && strings.Contains(cmd, "forge@jmmaranan.com") {
			forgeDisabled = true
			break
		}
	}
	if !forgeDisabled {
		t.Error("esperava que Forge fosse desabilitado")
	}
}

func TestApply_ForgeDisableFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 47.2"}
	// Forge presente
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Name: Forge\n  Enabled: Yes\n  State: ACTIVE\n",
	}
	// Forge disable falha
	mock.ExecResults["gnome-extensions disable forge@jmmaranan.com"] = system.ExecResult{
		Err: fmt.Errorf("extension error"),
	}
	// Tiling Shell ja instalado
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Output: "tilingshell@ferrarodomenico.com\n  Name: Tiling Shell\n  Enabled: Yes\n  State: ACTIVE\n",
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	// Nao deve falhar — Forge disable e warn, nao fatal
	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: Forge disable nao deveria ser fatal, obteve: %v", err)
	}
}

func TestApply_InstallFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 47.2"}
	// Forge nao presente
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}
	// Tiling Shell nao instalado
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}
	// curl falha (simula falha de instalacao)
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=tilingshell@ferrarodomenico.com&shell_version=47"] = system.ExecResult{
		Err: fmt.Errorf("connection refused"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando instalacao falha")
	}
}

func TestApply_DconfFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 47.2"}
	// Forge nao presente
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}
	// Tiling Shell ja instalado
	mock.ExecResults["gnome-extensions show tilingshell@ferrarodomenico.com"] = system.ExecResult{
		Output: "tilingshell@ferrarodomenico.com\n  Name: Tiling Shell\n  Enabled: Yes\n  State: ACTIVE\n",
	}
	// dconf write falha
	mock.ExecResults["dconf write /org/gnome/shell/extensions/tilingshell/inner-gaps uint32 4"] = system.ExecResult{
		Err: fmt.Errorf("dconf error"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando dconf write falha")
	}
}
