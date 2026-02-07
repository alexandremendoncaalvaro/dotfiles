package starship

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ale/dotfiles/internal/module"
	"github.com/ale/dotfiles/internal/module/moduletest"
	"github.com/ale/dotfiles/internal/system"
)

func TestCheck_Missing(t *testing.T) {
	mock := system.NewMock()
	mod := New("/repo/configs/starship.toml")

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
	mock.Commands["starship"] = true
	mock.Files["/home/test/.config/starship.toml"] = []byte("config")
	mock.Files["/home/test/.bashrc"] = []byte(`eval "$(starship init bash)"`)

	mod := New("/repo/configs/starship.toml")

	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if status.Kind != module.Installed {
		t.Errorf("esperava Installed, obteve %s", status.Kind)
	}
}

func TestCheck_Partial(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["starship"] = true
	// Sem config e sem bashrc init

	mod := New("/repo/configs/starship.toml")

	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if status.Kind != module.Partial {
		t.Errorf("esperava Partial, obteve %s", status.Kind)
	}
}

func TestApply_InstallAndConfigure(t *testing.T) {
	mock := system.NewMock()
	mock.Files["/home/test/.bashrc"] = []byte("# meu bashrc\n")

	mod := New("/repo/configs/starship.toml")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Verifica que o comando de instalacao foi chamado
	if len(mock.ExecLog) == 0 {
		t.Error("esperava que o comando de instalacao fosse executado")
	}

	// Verifica symlink
	if dest, ok := mock.Symlinks["/home/test/.config/starship.toml"]; !ok {
		t.Error("symlink nao foi criado")
	} else if dest != "/repo/configs/starship.toml" {
		t.Errorf("symlink aponta para %s, esperava /repo/configs/starship.toml", dest)
	}

	// Verifica bashrc
	bashrc := string(mock.Files["/home/test/.bashrc"])
	if !strings.Contains(bashrc, `eval "$(starship init bash)"`) {
		t.Error("init do starship nao foi adicionado ao .bashrc")
	}
}

func TestApply_SkipZshrcIfNotExists(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["starship"] = true
	mock.Files["/home/test/.bashrc"] = []byte("# bashrc\n")
	// .zshrc nao existe

	mod := New("/repo/configs/starship.toml")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// .zshrc nao deve ter sido criado
	if _, ok := mock.Files["/home/test/.zshrc"]; ok {
		t.Error(".zshrc nao deveria ter sido criado")
	}
}

func TestApply_AddToZshrcIfExists(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["starship"] = true
	mock.Files["/home/test/.bashrc"] = []byte("# bashrc\n")
	mock.Files["/home/test/.zshrc"] = []byte("# zshrc\n")

	mod := New("/repo/configs/starship.toml")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	zshrc := string(mock.Files["/home/test/.zshrc"])
	if !strings.Contains(zshrc, `eval "$(starship init zsh)"`) {
		t.Error("init do starship nao foi adicionado ao .zshrc")
	}
}

func TestApply_InstallFails(t *testing.T) {
	mock := system.NewMock()
	mock.Files["/home/test/.bashrc"] = []byte("")
	mock.ExecResults["sh -c curl -sS https://starship.rs/install.sh | sh -s -- -y"] = system.ExecResult{
		Err: fmt.Errorf("curl failed"),
	}

	mod := New("/repo/configs/starship.toml")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando instalacao falha")
	}
}

func TestApply_MkdirAllFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["starship"] = true
	mock.MkdirAllErr = fmt.Errorf("permissao negada")

	mod := New("/repo/configs/starship.toml")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando MkdirAll falha")
	}
}

func TestApply_SymlinkFails(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["starship"] = true
	mock.SymlinkErr = fmt.Errorf("link error")

	mod := New("/repo/configs/starship.toml")
	reporter := moduletest.NoopReporter()

	err := mod.Apply(context.Background(), mock, reporter)
	if err == nil {
		t.Error("esperava erro quando Symlink falha")
	}
}

func TestCheck_ReadFileError(t *testing.T) {
	mock := system.NewMock()
	mock.Commands["starship"] = true
	mock.Files["/home/test/.config/starship.toml"] = []byte("config")
	// .bashrc nao existe, ReadFile vai retornar erro
	// Nesse caso hasBashInit fica false (por design)

	mod := New("/repo/configs/starship.toml")
	status, err := mod.Check(context.Background(), mock)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if status.Kind != module.Partial {
		t.Errorf("esperava Partial sem .bashrc, obteve %s", status.Kind)
	}
}
