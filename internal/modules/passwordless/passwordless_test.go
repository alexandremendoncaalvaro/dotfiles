package passwordless

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
	mock.EnvVars["USER"] = "ale"
	// sudo -n true falha
	mock.ExecResults["sudo -n true"] = system.ExecResult{Err: fmt.Errorf("senha necessaria")}
	// GDM nao configurado
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\n")

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
	mock.EnvVars["USER"] = "ale"
	// sudo -n true sucesso (default do mock: sem resultado = sem erro)
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\nAutomaticLoginEnable=True\nAutomaticLogin=ale\n")

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestCheck_Partial_SudoOnly(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	// sudo OK (default)
	// GDM sem login automatico
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\n")

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
	if !strings.Contains(status.Message, "login automatico ausente") {
		t.Errorf("mensagem inesperada: %s", status.Message)
	}
}

func TestCheck_Partial_GDMOnly(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	// sudo falha
	mock.ExecResults["sudo -n true"] = system.ExecResult{Err: fmt.Errorf("senha necessaria")}
	// GDM configurado
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\nAutomaticLoginEnable=True\nAutomaticLogin=ale\n")

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
	if !strings.Contains(status.Message, "sudo com senha") {
		t.Errorf("mensagem inesperada: %s", status.Message)
	}
}

func TestCheck_GDMFileNotFound(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	// sudo OK, GDM file missing

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// sudo OK mas GDM nao encontrado = Partial
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
}

func TestCheck_UserEmpty(t *testing.T) {
	mock := system.NewMock()
	// USER nao definido, sudo OK
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\nAutomaticLoginEnable=True\nAutomaticLogin=ale\n")

	mod := New()
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// sudo OK mas USER vazio = GDM check falha = Partial
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
}

func TestApply_ConfiguresBoth(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\n#AutomaticLoginEnable=False\n\n[security]\n")

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica sudoers temporario
	sudoersData, ok := mock.Files["/home/test/.cache/blueprint-nopasswd"]
	if !ok {
		t.Fatal("arquivo temporario de sudoers nao criado")
	}
	if string(sudoersData) != "ale ALL=(ALL) NOPASSWD: ALL\n" {
		t.Errorf("conteudo do sudoers inesperado: %q", string(sudoersData))
	}

	// Verifica GDM temporario
	gdmData, ok := mock.Files["/home/test/.cache/blueprint-gdm-custom.conf"]
	if !ok {
		t.Fatal("arquivo temporario do GDM nao criado")
	}
	gdmContent := string(gdmData)
	if !strings.Contains(gdmContent, "AutomaticLoginEnable=True") {
		t.Error("AutomaticLoginEnable nao configurado")
	}
	if !strings.Contains(gdmContent, "AutomaticLogin=ale") {
		t.Error("AutomaticLogin nao configurado")
	}

	// Verifica comandos executados
	expectedCmds := []string{
		"sudo visudo -c -f /home/test/.cache/blueprint-nopasswd",
		"sudo cp /home/test/.cache/blueprint-nopasswd /etc/sudoers.d/nopasswd-ale",
		"sudo chmod 0440 /etc/sudoers.d/nopasswd-ale",
		"sudo cp /home/test/.cache/blueprint-gdm-custom.conf /etc/gdm/custom.conf",
	}
	for _, cmd := range expectedCmds {
		found := false
		for _, logged := range mock.ExecLog {
			if logged == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("comando esperado nao executado: %s", cmd)
		}
	}
}

func TestApply_UserEmpty(t *testing.T) {
	mock := system.NewMock()
	// USER nao definido

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando USER nao definido")
	}
	if !strings.Contains(err.Error(), "USER") {
		t.Errorf("mensagem de erro deveria mencionar USER: %v", err)
	}
}

func TestApply_SudoersWriteFails(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	mock.WriteFileErr = fmt.Errorf("disco cheio")

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando WriteFile falha")
	}
}

func TestApply_VisudoValidationFails(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	mock.ExecResults["sudo visudo -c -f /home/test/.cache/blueprint-nopasswd"] = system.ExecResult{
		Err: fmt.Errorf("syntax error"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando visudo falha")
	}
	if !strings.Contains(err.Error(), "validacao") {
		t.Errorf("mensagem de erro deveria mencionar validacao: %v", err)
	}
}

func TestApply_SudoCpFails(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	mock.ExecResults["sudo cp /home/test/.cache/blueprint-nopasswd /etc/sudoers.d/nopasswd-ale"] = system.ExecResult{
		Err: fmt.Errorf("permissao negada"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando sudo cp falha")
	}
}

func TestApply_GDMReadFails(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	// GDM file nao existe — ReadFile vai falhar

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando leitura do GDM falha")
	}
}

func TestApply_GDMCpFails(t *testing.T) {
	mock := system.NewMock()
	mock.EnvVars["USER"] = "ale"
	mock.Files["/etc/gdm/custom.conf"] = []byte("[daemon]\n")
	mock.ExecResults["sudo cp /home/test/.cache/blueprint-gdm-custom.conf /etc/gdm/custom.conf"] = system.ExecResult{
		Err: fmt.Errorf("permissao negada"),
	}

	mod := New()
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando sudo cp do GDM falha")
	}
}

func TestSetGDMAutoLogin_AddsToEmptyDaemon(t *testing.T) {
	input := "[daemon]\n\n[security]\n"
	result := setGDMAutoLogin(input, "ale")

	if !strings.Contains(result, "AutomaticLoginEnable=True") {
		t.Error("AutomaticLoginEnable nao adicionado")
	}
	if !strings.Contains(result, "AutomaticLogin=ale") {
		t.Error("AutomaticLogin nao adicionado")
	}
}

func TestSetGDMAutoLogin_UpdatesExisting(t *testing.T) {
	input := "[daemon]\nAutomaticLoginEnable=False\nAutomaticLogin=outro\n"
	result := setGDMAutoLogin(input, "ale")

	if !strings.Contains(result, "AutomaticLoginEnable=True") {
		t.Error("AutomaticLoginEnable nao atualizado")
	}
	if !strings.Contains(result, "AutomaticLogin=ale") {
		t.Error("AutomaticLogin nao atualizado")
	}
	if strings.Contains(result, "False") {
		t.Error("valor antigo nao removido")
	}
	if strings.Contains(result, "outro") {
		t.Error("usuario antigo nao removido")
	}
}

func TestSetGDMAutoLogin_DaemonIsLastSection(t *testing.T) {
	input := "[security]\nkey=val\n\n[daemon]\n"
	result := setGDMAutoLogin(input, "ale")

	if !strings.Contains(result, "AutomaticLoginEnable=True") {
		t.Error("AutomaticLoginEnable nao adicionado")
	}
	if !strings.Contains(result, "AutomaticLogin=ale") {
		t.Error("AutomaticLogin nao adicionado")
	}
}
