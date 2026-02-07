package gnome

import (
	"context"
	"fmt"
	"testing"

	"github.com/ale/dotfiles/internal/module"
	"github.com/ale/dotfiles/internal/system"
)

func TestShouldRunGuard_Container(t *testing.T) {
	mock := system.NewMock()
	mock.Container = true
	ok, reason := ShouldRunGuard(mock)
	if ok {
		t.Error("deveria retornar false em container")
	}
	if reason == "" {
		t.Error("deveria ter motivo")
	}
}

func TestShouldRunGuard_NoDisplay(t *testing.T) {
	mock := system.NewMock()
	ok, _ := ShouldRunGuard(mock)
	if ok {
		t.Error("deveria retornar false sem sessao grafica")
	}
}

func TestShouldRunGuard_NoGnomeExtensions(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"
	ok, _ := ShouldRunGuard(mock)
	if ok {
		t.Error("deveria retornar false sem gnome-extensions")
	}
}

func TestShouldRunGuard_OK(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"
	mock.Commands["gnome-extensions"] = true
	ok, reason := ShouldRunGuard(mock)
	if !ok {
		t.Error("deveria retornar true em desktop GNOME")
	}
	if reason != "" {
		t.Errorf("nao deveria ter motivo: %s", reason)
	}
}

func TestShouldRunGuard_XDisplay(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["DISPLAY"] = ":0"
	mock.Commands["gnome-extensions"] = true
	ok, _ := ShouldRunGuard(mock)
	if !ok {
		t.Error("deveria aceitar DISPLAY (X11)")
	}
}

func TestCheckExtension_Missing(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show test@ext"] = system.ExecResult{
		Err: fmt.Errorf("not found"),
	}
	status, err := CheckExtension(context.Background(), mock, "test@ext", "Test")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing, obteve %s", status.Kind)
	}
}

func TestCheckExtension_Installed(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show test@ext"] = system.ExecResult{
		Output: "test@ext\n  Enabled: Yes\n  State: ACTIVE\n",
	}
	status, err := CheckExtension(context.Background(), mock, "test@ext", "Test")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestCheckExtension_Disabled(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show test@ext"] = system.ExecResult{
		Output: "test@ext\n  Enabled: No\n  State: INACTIVE\n",
	}
	status, _ := CheckExtension(context.Background(), mock, "test@ext", "Test")
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para desativado, obteve %s", status.Kind)
	}
}

func TestCheckExtension_OutOfDate(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show test@ext"] = system.ExecResult{
		Output: "test@ext\n  Enabled: Yes\n  State: OUT OF DATE\n",
	}
	status, _ := CheckExtension(context.Background(), mock, "test@ext", "Test")
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para OUT OF DATE, obteve %s", status.Kind)
	}
}

func TestCheckExtension_Error(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-extensions show test@ext"] = system.ExecResult{
		Output: "test@ext\n  Enabled: Yes\n  State: ERROR\n",
	}
	status, _ := CheckExtension(context.Background(), mock, "test@ext", "Test")
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial para ERROR, obteve %s", status.Kind)
	}
}

func TestDetectVersion(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 46.2"}
	ver, err := DetectVersion(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if ver != "46" {
		t.Errorf("esperava 46, obteve %s", ver)
	}
}

func TestDetectVersion_NoDecimal(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "GNOME Shell 49"}
	ver, err := DetectVersion(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if ver != "49" {
		t.Errorf("esperava 49, obteve %s", ver)
	}
}

func TestDetectVersion_NotFound(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Err: fmt.Errorf("not found")}
	_, err := DetectVersion(context.Background(), mock)
	if err == nil {
		t.Error("esperava erro quando gnome-shell nao existe")
	}
}

func TestDetectVersion_MalformedOutput(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["gnome-shell --version"] = system.ExecResult{Output: "invalid"}
	_, err := DetectVersion(context.Background(), mock)
	if err == nil {
		t.Error("esperava erro para saida malformada")
	}
}

func TestInstallFromGnomeExtensions_CurlFails(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=test@ext&shell_version=46"] = system.ExecResult{
		Err: fmt.Errorf("connection refused"),
	}
	err := InstallFromGnomeExtensions(context.Background(), mock, "test@ext", "46", "Test")
	if err == nil {
		t.Error("esperava erro quando curl falha")
	}
}

func TestInstallFromGnomeExtensions_BadJSON(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=test@ext&shell_version=46"] = system.ExecResult{
		Output: "not json",
	}
	err := InstallFromGnomeExtensions(context.Background(), mock, "test@ext", "46", "Test")
	if err == nil {
		t.Error("esperava erro para JSON invalido")
	}
}

func TestInstallFromGnomeExtensions_EmptyDownloadURL(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=test@ext&shell_version=46"] = system.ExecResult{
		Output: `{"download_url": ""}`,
	}
	err := InstallFromGnomeExtensions(context.Background(), mock, "test@ext", "46", "Test")
	if err == nil {
		t.Error("esperava erro para download_url vazio")
	}
}

func TestInstallFromGnomeExtensions_DownloadFails(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=test@ext&shell_version=46"] = system.ExecResult{
		Output: `{"download_url": "/download/extension/test.zip"}`,
	}
	mock.ExecResults["curl -sfL -o /tmp/test@ext.zip https://extensions.gnome.org/download/extension/test.zip"] = system.ExecResult{
		Err: fmt.Errorf("download failed"),
	}
	err := InstallFromGnomeExtensions(context.Background(), mock, "test@ext", "46", "Test")
	if err == nil {
		t.Error("esperava erro quando download falha")
	}
}

func TestInstallFromGnomeExtensions_InstallFails(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=test@ext&shell_version=46"] = system.ExecResult{
		Output: `{"download_url": "/download/extension/test.zip"}`,
	}
	mock.ExecResults["curl -sfL -o /tmp/test@ext.zip https://extensions.gnome.org/download/extension/test.zip"] = system.ExecResult{}
	mock.ExecResults["gnome-extensions install --force /tmp/test@ext.zip"] = system.ExecResult{
		Err: fmt.Errorf("install failed"),
	}
	err := InstallFromGnomeExtensions(context.Background(), mock, "test@ext", "46", "Test")
	if err == nil {
		t.Error("esperava erro quando install falha")
	}
}

func TestInstallFromGnomeExtensions_Success(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["curl -sfL https://extensions.gnome.org/extension-info/?uuid=test@ext&shell_version=46"] = system.ExecResult{
		Output: `{"download_url": "/download/extension/test.zip"}`,
	}
	mock.ExecResults["curl -sfL -o /tmp/test@ext.zip https://extensions.gnome.org/download/extension/test.zip"] = system.ExecResult{}
	mock.ExecResults["gnome-extensions install --force /tmp/test@ext.zip"] = system.ExecResult{}
	err := InstallFromGnomeExtensions(context.Background(), mock, "test@ext", "46", "Test")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestApplyDconf_Success(t *testing.T) {
	mock := system.NewMock()
	entries := []DconfEntry{
		{"/org/gnome/test/key1", "value1"},
		{"/org/gnome/test/key2", "value2"},
	}
	err := ApplyDconf(context.Background(), mock, entries)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(mock.ExecLog) != 2 {
		t.Errorf("esperava 2 chamadas dconf, obteve %d", len(mock.ExecLog))
	}
}

func TestApplyDconf_Failure(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["dconf write /org/gnome/test/key1 value1"] = system.ExecResult{
		Err: fmt.Errorf("dconf error"),
	}
	entries := []DconfEntry{
		{"/org/gnome/test/key1", "value1"},
		{"/org/gnome/test/key2", "value2"},
	}
	err := ApplyDconf(context.Background(), mock, entries)
	if err == nil {
		t.Error("esperava erro quando dconf falha")
	}
}
