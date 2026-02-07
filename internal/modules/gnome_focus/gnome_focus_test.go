package gnome_focus

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

	mod := New("/configs/focus-mode")
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular em container")
	}
}

func TestShouldRun_SkipWithoutDisplay(t *testing.T) {
	mock := system.NewMock()

	mod := New("/configs/focus-mode")
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular sem sessao grafica")
	}
}

func TestShouldRun_RunOnGnomeDesktop(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"
	mock.Commands["gnome-extensions"] = true

	mod := New("/configs/focus-mode")
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if !ok {
		t.Error("deveria rodar em desktop GNOME")
	}
}

func TestCheck_Missing(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show focus-mode@dotfiles"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}

	mod := New("/configs/focus-mode")
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
	mock.ExecResults["gnome-extensions show focus-mode@dotfiles"] = system.ExecResult{
		Output: "focus-mode@dotfiles\n  State: ENABLED\n",
	}
	mock.ExecResults["dconf read /org/gnome/mutter/dynamic-workspaces"] = system.ExecResult{
		Output: "true",
	}

	mod := New("/configs/focus-mode")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestApply_InstallsAndEnables(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["dconf"] = true

	mod := New("/repo/configs/gnome-extensions/focus-mode@dotfiles")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica symlink criado
	dest := "/home/test/.local/share/gnome-shell/extensions/focus-mode@dotfiles"
	if target, ok := mock.Symlinks[dest]; !ok {
		t.Error("symlink nao foi criado")
	} else if target != "/repo/configs/gnome-extensions/focus-mode@dotfiles" {
		t.Errorf("symlink aponta para %s", target)
	}

	// Verifica que dconf write foi chamado para dynamic-workspaces
	found := false
	for _, cmd := range mock.ExecLog {
		if cmd == "dconf write /org/gnome/mutter/dynamic-workspaces true" {
			found = true
			break
		}
	}
	if !found {
		t.Error("dconf write para dynamic-workspaces nao foi chamado")
	}
}
