package bluefin_update

import (
	"context"
	"fmt"
	"strings"
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

func TestShouldRun_RunOutsideContainer(t *testing.T) {
	mock := system.NewMock()
	mod := New()

	ok, _ := mod.ShouldRun(context.Background(), mock)
	if !ok {
		t.Error("deveria rodar fora de container")
	}
}

func TestCheck_SystemUpToDate(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	// rpm-ostree upgrade --check retorna erro (exit 77) = sem updates
	mock.ExecResults["rpm-ostree upgrade --check"] = system.ExecResult{Err: fmt.Errorf("exit status 77")}
	// flatpak remote-ls --updates retorna vazio = sem updates
	mock.ExecResults["flatpak remote-ls --updates"] = system.ExecResult{Output: ""}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestCheck_RpmOstreeUpdateAvailable(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	// rpm-ostree upgrade --check retorna sucesso (exit 0) = update disponivel
	mock.ExecResults["rpm-ostree upgrade --check"] = system.ExecResult{Output: "AvailableUpdate:\n  Version: 40.20240310"}
	// flatpak sem updates
	mock.ExecResults["flatpak remote-ls --updates"] = system.ExecResult{Output: ""}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing (update disponivel), obteve %s", status.Kind)
	}
	if !strings.Contains(status.Message, "rpm-ostree") {
		t.Errorf("mensagem deveria mencionar rpm-ostree: %s", status.Message)
	}
}

func TestCheck_FlatpakUpdateAvailable(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	// rpm-ostree sem updates
	mock.ExecResults["rpm-ostree upgrade --check"] = system.ExecResult{Err: fmt.Errorf("exit status 77")}
	// flatpak com updates
	mock.ExecResults["flatpak remote-ls --updates"] = system.ExecResult{Output: "org.mozilla.Firefox\tstable\tflathub"}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing (flatpak update disponivel), obteve %s", status.Kind)
	}
	if !strings.Contains(status.Message, "Flatpak") {
		t.Errorf("mensagem deveria mencionar Flatpak: %s", status.Message)
	}
}

func TestCheck_BothUpdatesAvailable(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	mock.ExecResults["rpm-ostree upgrade --check"] = system.ExecResult{Output: "AvailableUpdate:\n  Version: 40.20240310"}
	mock.ExecResults["flatpak remote-ls --updates"] = system.ExecResult{Output: "org.mozilla.Firefox\tstable\tflathub"}

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Missing {
		t.Errorf("esperava Missing (ambos com updates), obteve %s", status.Kind)
	}
	if !strings.Contains(status.Message, "sistema") || !strings.Contains(status.Message, "Flatpak") {
		t.Errorf("mensagem deveria mencionar sistema e Flatpak: %s", status.Message)
	}
}

func TestCheck_CommandsNotAvailable(t *testing.T) {
	mock := system.NewMock()
	// Nenhum comando disponivel — nao e Bluefin

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed (sem comandos para verificar), obteve %s", status.Kind)
	}
}

func TestApply_AllCommandsAvailable(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	mock.Commands["fwupdmgr"] = true
	mock.Commands["distrobox"] = true

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica que todos os comandos foram executados
	if len(mock.ExecLog) != 5 {
		t.Errorf("esperava 5 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}
}

func TestApply_SkipOptionalMissing(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	// fwupdmgr e distrobox nao disponiveis

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Apenas rpm-ostree e flatpak devem ter sido executados
	if len(mock.ExecLog) != 2 {
		t.Errorf("esperava 2 comandos executados, obteve %d: %v", len(mock.ExecLog), mock.ExecLog)
	}
}

func TestApply_FailOnMissingRequired(t *testing.T) {
	mock := system.NewMock()
	// rpm-ostree nao disponivel (obrigatorio)
	mock.Commands["flatpak"] = true

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando comando obrigatorio esta ausente")
	}
}

func TestApply_OptionalFailureContinues(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	mock.Commands["fwupdmgr"] = true
	mock.ExecResults["fwupdmgr refresh"] = system.ExecResult{Err: fmt.Errorf("falha")}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("nao deveria falhar com erro opcional: %v", err)
	}
}

func TestApply_RequiredCommandFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["rpm-ostree"] = true
	mock.Commands["flatpak"] = true
	// rpm-ostree upgrade falha durante execucao
	mock.ExecResults["rpm-ostree upgrade"] = system.ExecResult{Err: fmt.Errorf("upgrade failed")}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando comando obrigatorio falha durante execucao")
	}
}
