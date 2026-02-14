package profile

import (
	"testing"

	"github.com/ale/blueprint/internal/system"
)

func TestDetect_Container(t *testing.T) {
	mock := system.NewMock()
	mock.Container = true
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"

	got := Detect(mock)
	if got.Name != "minimal" {
		t.Errorf("container deveria retornar minimal, obteve %s", got.Name)
	}
}

func TestDetect_WSL(t *testing.T) {
	mock := system.NewMock()
	mock.WSL = true
	// Mesmo com display vazio (que seria server), WSL deve ganhar
	mock.EnvVars["DISPLAY"] = ""

	got := Detect(mock)
	if got.Name != "wsl" {
		t.Errorf("WSL deveria retornar wsl, obteve %s", got.Name)
	}
}

func TestDetect_NoDisplay(t *testing.T) {
	mock := system.NewMock()
	// Sem DISPLAY e sem WAYLAND_DISPLAY

	got := Detect(mock)
	if got.Name != "server" {
		t.Errorf("sem display deveria retornar server, obteve %s", got.Name)
	}
}

func TestDetect_Wayland(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["WAYLAND_DISPLAY"] = "wayland-0"

	got := Detect(mock)
	if got.Name != "full" {
		t.Errorf("wayland deveria retornar full, obteve %s", got.Name)
	}
}

func TestDetect_X11(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["DISPLAY"] = ":0"

	got := Detect(mock)
	if got.Name != "full" {
		t.Errorf("X11 deveria retornar full, obteve %s", got.Name)
	}
}
