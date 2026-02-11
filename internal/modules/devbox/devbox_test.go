package devbox

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

	mod := New("/repo/configs/devbox/setup-dev.sh")
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
	mod := New("/repo/configs/devbox/setup-dev.sh")

	ok, _ := mod.ShouldRun(context.Background(), mock)
	if !ok {
		t.Error("deveria rodar fora de container")
	}
}

func TestCheck_Missing(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["distrobox list"] = system.ExecResult{Output: "ID | NAME | STATUS | IMAGE\n"}

	mod := New("/repo/configs/devbox/setup-dev.sh")
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
	mock.ExecResults["distrobox list"] = system.ExecResult{
		Output: "ID | NAME | STATUS | IMAGE\nabc123 | devbox | running | quay.io/toolbx/ubuntu-toolbox:24.04\n",
	}

	mod := New("/repo/configs/devbox/setup-dev.sh")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestCheck_DistroboxNotAvailable(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["distrobox list"] = system.ExecResult{Err: fmt.Errorf("command not found")}

	mod := New("/repo/configs/devbox/setup-dev.sh")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing, obteve %s", status.Kind)
	}
}

func TestCheck_NoFalsePositive(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["distrobox list"] = system.ExecResult{
		Output: "ID | NAME | STATUS | IMAGE\nabc123 | my-devbox | running | ubuntu:24.04\n",
	}

	mod := New("/repo/configs/devbox/setup-dev.sh")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing para 'my-devbox', obteve %s", status.Kind)
	}
}

func TestApply_Success(t *testing.T) {
	mock := system.NewMock()

	mod := New("/repo/configs/devbox/setup-dev.sh")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Sem .vscode-server existente, sao 2 comandos (create + provision)
	if len(mock.ExecLog) != 2 {
		t.Errorf("esperava 2 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}

	expected := []string{
		"distrobox create --name devbox --image quay.io/toolbx/ubuntu-toolbox:24.04 --home /home/test/.distrobox/devbox --yes",
		"distrobox enter devbox -- bash /repo/configs/devbox/setup-dev.sh",
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

func TestApply_ChownsVscodeServer(t *testing.T) {
	mock := system.NewMock()
	// Simula .vscode-server existente (de sessao anterior como root)
	mock.Files["/home/test/.distrobox/devbox/.vscode-server"] = []byte{}

	mod := New("/repo/configs/devbox/setup-dev.sh")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// 3 comandos: create + provision + chown
	if len(mock.ExecLog) != 3 {
		t.Fatalf("esperava 3 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}

	chownCmd := mock.ExecLog[2]
	expected := "distrobox enter devbox -- sudo chown -R ale:ale /home/test/.distrobox/devbox/.vscode-server"
	if chownCmd != expected {
		t.Errorf("comando chown: esperava %q, obteve %q", expected, chownCmd)
	}
}

func TestApply_CreateFailsContinues(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["distrobox create --name devbox --image quay.io/toolbx/ubuntu-toolbox:24.04 --home /home/test/.distrobox/devbox --yes"] = system.ExecResult{
		Err: fmt.Errorf("container already exists"),
	}

	mod := New("/repo/configs/devbox/setup-dev.sh")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("nao deveria falhar quando create falha (container ja existe): %v", err)
	}

	// Deve continuar e executar o provisionamento
	if len(mock.ExecLog) != 2 {
		t.Errorf("esperava 2 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}
}

func TestApply_SetupFails(t *testing.T) {
	mock := system.NewMock()
	mock.ExecResults["distrobox enter devbox -- bash /repo/configs/devbox/setup-dev.sh"] = system.ExecResult{
		Err: fmt.Errorf("setup failed"),
	}

	mod := New("/repo/configs/devbox/setup-dev.sh")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando setup falha")
	}
}
