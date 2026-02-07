package gnome_forge

import (
	"context"
	"fmt"
	"testing"

	"github.com/ale/dotfiles/internal/module"
	"github.com/ale/dotfiles/internal/module/moduletest"
	"github.com/ale/dotfiles/internal/system"
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
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
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
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Name: Forge\n  Enabled: Yes\n  State: ACTIVE\n",
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

func TestCheck_OutOfDate(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Enabled: Yes\n  State: OUT OF DATE\n",
	}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para OUT OF DATE, obteve %s", status.Kind)
	}
}

func TestCheck_Disabled(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Enabled: No\n  State: INACTIVE\n",
	}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para desativado, obteve %s", status.Kind)
	}
}

func TestApply_AlreadyInstalled(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 46.2"}
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Enabled: Yes\n  State: ACTIVE\n",
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica que dconf write foi chamado (keybindings)
	hasDconf := false
	for _, cmd := range mock.ExecLog {
		if len(cmd) > 10 && cmd[:10] == "dconf writ" {
			hasDconf = true
			break
		}
	}
	if !hasDconf {
		t.Error("esperava chamadas ao dconf para configurar atalhos")
	}
}

func TestApply_GnomeVersionFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Err: fmt.Errorf("not found")}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando gnome-shell nao esta disponivel")
	}
}

func TestApply_DconfFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["gnome-shell"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 46.2"}
	mock.ExecResults["gnome-extensions show forge@jmmaranan.com"] = system.ExecResult{
		Output: "forge@jmmaranan.com\n  Enabled: Yes\n  State: ACTIVE\n",
	}
	// Primeira chamada dconf falha
	mock.ExecResults["dconf write /org/gnome/desktop/wm/keybindings/maximize @as []"] = system.ExecResult{
		Err: fmt.Errorf("dconf error"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando dconf write falha")
	}
}
