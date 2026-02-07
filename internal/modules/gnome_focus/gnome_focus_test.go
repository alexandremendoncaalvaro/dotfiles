package gnome_focus

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
	mock.ExecResults["gnome-extensions show focus-mode@blueprint"] = system.ExecResult{
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
	mock.ExecResults["gnome-extensions show focus-mode@blueprint"] = system.ExecResult{
		Output: "focus-mode@blueprint\n  Enabled: Yes\n  State: ACTIVE\n",
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

func TestCheck_OutOfDate(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show focus-mode@blueprint"] = system.ExecResult{
		Output: "focus-mode@blueprint\n  Enabled: Yes\n  State: OUT OF DATE\n",
	}

	mod := New("/configs/focus-mode")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para OUT OF DATE, obteve %s", status.Kind)
	}
}

func TestCheck_Error(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show focus-mode@blueprint"] = system.ExecResult{
		Output: "focus-mode@blueprint\n  Enabled: Yes\n  State: ERROR\n",
	}

	mod := New("/configs/focus-mode")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para ERROR, obteve %s", status.Kind)
	}
}

func TestCheck_Disabled(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show focus-mode@blueprint"] = system.ExecResult{
		Output: "focus-mode@blueprint\n  Enabled: No\n  State: INACTIVE\n",
	}

	mod := New("/configs/focus-mode")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para desativado, obteve %s", status.Kind)
	}
}

func TestCheck_DynamicWorkspacesDisabled(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show focus-mode@blueprint"] = system.ExecResult{
		Output: "focus-mode@blueprint\n  Enabled: Yes\n  State: ACTIVE\n",
	}
	mock.ExecResults["dconf read /org/gnome/mutter/dynamic-workspaces"] = system.ExecResult{
		Output: "false",
	}

	mod := New("/configs/focus-mode")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para dynamic-workspaces=false, obteve %s", status.Kind)
	}
}

func TestApply_InstallsAndEnables(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["dconf"] = true

	mod := New("/repo/configs/gnome-extensions/focus-mode@blueprint")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica que os comandos corretos foram executados
	expectedCmds := []string{
		"dconf write /org/gnome/mutter/dynamic-workspaces true",
	}
	for _, expected := range expectedCmds {
		found := false
		for _, cmd := range mock.ExecLog {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("comando esperado nao executado: %s", expected)
		}
	}

	// Verifica que zip e gnome-extensions install foram chamados
	foundZip := false
	foundInstall := false
	for _, cmd := range mock.ExecLog {
		if strings.HasPrefix(cmd, "zip -j") {
			foundZip = true
		}
		if strings.HasPrefix(cmd, "gnome-extensions install --force") {
			foundInstall = true
		}
	}
	if !foundZip {
		t.Error("zip nao foi chamado")
	}
	if !foundInstall {
		t.Error("gnome-extensions install nao foi chamado")
	}
}

func TestApply_DconfFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["dconf write /org/gnome/mutter/dynamic-workspaces true"] = system.ExecResult{
		Err: fmt.Errorf("dconf error"),
	}

	mod := New("/repo/configs/focus-mode")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando dconf write falha")
	}
}

func TestApply_ZipFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["dconf"] = true
	// zip falha para qualquer chamada que comece com "zip"
	mock.ExecResults["zip -j /tmp/focus-mode@blueprint.zip /repo/configs/focus-mode/metadata.json /repo/configs/focus-mode/extension.js"] = system.ExecResult{
		Err: fmt.Errorf("zip not found"),
	}

	mod := New("/repo/configs/focus-mode")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando zip falha")
	}
}

func TestApply_InstallFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["gnome-extensions"] = true
	mock.Commands["dconf"] = true
	mock.ExecResults["gnome-extensions install --force /tmp/focus-mode@blueprint.zip"] = system.ExecResult{
		Err: fmt.Errorf("install error"),
	}

	mod := New("/repo/configs/focus-mode")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando gnome-extensions install falha")
	}
}

func TestShouldRun_SkipWithoutGnomeExtensions(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"
	// gnome-extensions nao disponivel

	mod := New("/configs/focus-mode")
	ok, _ := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular sem gnome-extensions")
	}
}
