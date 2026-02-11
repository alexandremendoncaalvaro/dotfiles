package devcontainers

import (
	"context"
	"fmt"
	"testing"

	"github.com/ale/blueprint/internal/module"
	"github.com/ale/blueprint/internal/module/moduletest"
	"github.com/ale/blueprint/internal/system"
)

func TestShouldRun_SkipInContainer(t *testing.T) {
	mock := system.NewMock()
	mock.Container = true

	mod := New()
	ok, reason := mod.ShouldRun(context.Background(), mock)
	if ok {
		t.Error("deveria pular em container")
	}
	if reason == "" {
		t.Error("deveria ter motivo")
	}
}

func TestShouldRun_RunOutsideContainer(t *testing.T) {
	mock := system.NewMock()
	mod := New()

	ok, _ := mod.ShouldRun(context.Background(), mock)
	if !ok {
		t.Error("deveria rodar fora de container")
	}
}

func TestCheck_Missing(t *testing.T) {
	mock := system.NewMock()
	// dev mode inativo (ujust devmode falha)
	mock.ExecResults["ujust devmode"] = system.ExecResult{Err: fmt.Errorf("not enabled")}
	// docker-ce presente
	mock.ExecResults["rpm -q docker-ce"] = system.ExecResult{Output: "docker-ce-24.0.0"}
	// podman-docker ausente
	mock.ExecResults["rpm -q podman-docker"] = system.ExecResult{Err: fmt.Errorf("not installed")}

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
	// dev mode ativo
	mock.ExecResults["ujust devmode"] = system.ExecResult{Output: "enabled"}
	// docker-ce ausente
	mock.ExecResults["rpm -q docker-ce"] = system.ExecResult{Err: fmt.Errorf("not installed")}
	// podman-docker presente
	mock.ExecResults["rpm -q podman-docker"] = system.ExecResult{Output: "podman-docker-4.0.0"}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestCheck_Partial_OnlyDevMode(t *testing.T) {
	mock := system.NewMock()
	// dev mode ativo
	mock.ExecResults["ujust devmode"] = system.ExecResult{Output: "enabled"}
	// docker-ce ainda presente
	mock.ExecResults["rpm -q docker-ce"] = system.ExecResult{Output: "docker-ce-24.0.0"}
	// podman-docker ausente
	mock.ExecResults["rpm -q podman-docker"] = system.ExecResult{Err: fmt.Errorf("not installed")}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
}

func TestCheck_Partial_OnlyPodman(t *testing.T) {
	mock := system.NewMock()
	// dev mode inativo
	mock.ExecResults["ujust devmode"] = system.ExecResult{Err: fmt.Errorf("not enabled")}
	// docker-ce ausente
	mock.ExecResults["rpm -q docker-ce"] = system.ExecResult{Err: fmt.Errorf("not installed")}
	// podman-docker presente
	mock.ExecResults["rpm -q podman-docker"] = system.ExecResult{Output: "podman-docker-4.0.0"}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
}

func TestApply_Success(t *testing.T) {
	mock := system.NewMock()

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(mock.ExecLog) != 3 {
		t.Errorf("esperava 3 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}

	expected := []string{
		"ujust devmode-enable",
		"rpm-ostree override remove docker-ce docker-ce-cli docker-ce-rootless-extras",
		"rpm-ostree install podman-docker",
	}
	for i, cmd := range expected {
		if i >= len(mock.ExecLog) {
			t.Errorf("comando %d ausente: %s", i, cmd)
			continue
		}
		if mock.ExecLog[i] != cmd {
			t.Errorf("comando %d: esperava %q, obteve %q", i, cmd, mock.ExecLog[i])
		}
	}
}

func TestApply_OverrideRemoveFails(t *testing.T) {
	mock := system.NewMock()
	// override remove falha (docker-ce ja removido)
	mock.ExecResults["rpm-ostree override remove docker-ce docker-ce-cli docker-ce-rootless-extras"] = system.ExecResult{
		Err: fmt.Errorf("package docker-ce not found"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("nao deveria falhar quando override remove falha: %v", err)
	}

	// Deve continuar e executar o install
	if len(mock.ExecLog) != 3 {
		t.Errorf("esperava 3 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}
}

func TestApply_DevModeEnableFails(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["ujust devmode-enable"] = system.ExecResult{Err: fmt.Errorf("failed")}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando devmode-enable falha")
	}
}

func TestApply_PodmanDockerInstallFails(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["rpm-ostree install podman-docker"] = system.ExecResult{Err: fmt.Errorf("failed")}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando install podman-docker falha")
	}
}
