package system

import (
	"testing"
)

func TestMock_FileExists_Files(t *testing.T) {
	mock := NewMock()
	mock.Files["/home/test/.config/starship.toml"] = []byte("test")

	if !mock.FileExists("/home/test/.config/starship.toml") {
		t.Error("deveria encontrar arquivo em Files")
	}
	if mock.FileExists("/nao/existe") {
		t.Error("nao deveria encontrar arquivo inexistente")
	}
}

func TestMock_FileExists_Symlinks(t *testing.T) {
	mock := NewMock()
	mock.Symlinks["/home/test/.local/share/gnome-shell/extensions/focus-mode@dotfiles"] = "/repo/configs/focus"

	if !mock.FileExists("/home/test/.local/share/gnome-shell/extensions/focus-mode@dotfiles") {
		t.Error("deveria encontrar symlink em Symlinks")
	}
}

func TestMock_FileExists_AfterSymlink(t *testing.T) {
	mock := NewMock()

	dest := "/home/test/link"
	if mock.FileExists(dest) {
		t.Error("nao deveria existir antes de Symlink")
	}

	_ = mock.Symlink("/source", dest)

	if !mock.FileExists(dest) {
		t.Error("deveria existir apos Symlink")
	}
}

func TestMock_AppendToFileIfMissing_NewFile(t *testing.T) {
	mock := NewMock()

	added, err := mock.AppendToFileIfMissing("/test/file", "nova linha")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !added {
		t.Error("deveria retornar true para arquivo novo")
	}

	content := string(mock.Files["/test/file"])
	if content != "nova linha\n" {
		t.Errorf("conteudo inesperado: %q", content)
	}
}

func TestMock_AppendToFileIfMissing_AlreadyExists(t *testing.T) {
	mock := NewMock()
	mock.Files["/test/file"] = []byte("linha existente\n")

	added, err := mock.AppendToFileIfMissing("/test/file", "linha existente")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if added {
		t.Error("nao deveria adicionar linha que ja existe")
	}
}

func TestMock_AppendToFileIfMissing_AppendWithNewline(t *testing.T) {
	mock := NewMock()
	mock.Files["/test/file"] = []byte("primeira") // Sem newline no final

	added, err := mock.AppendToFileIfMissing("/test/file", "segunda")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !added {
		t.Error("deveria adicionar linha nova")
	}

	content := string(mock.Files["/test/file"])
	expected := "primeira\nsegunda\n"
	if content != expected {
		t.Errorf("esperava %q, obteve %q", expected, content)
	}
}

func TestMock_ReadFile_NotFound(t *testing.T) {
	mock := NewMock()

	_, err := mock.ReadFile("/nao/existe")
	if err == nil {
		t.Error("deveria retornar erro para arquivo inexistente")
	}
}

func TestMock_Exec_Default(t *testing.T) {
	mock := NewMock()

	out, err := mock.Exec(nil, "cmd", "arg1")
	if err != nil {
		t.Errorf("exec padrao nao deveria retornar erro: %v", err)
	}
	if out != "" {
		t.Errorf("exec padrao deveria retornar string vazia: %q", out)
	}
	if len(mock.ExecLog) != 1 || mock.ExecLog[0] != "cmd arg1" {
		t.Errorf("exec log inesperado: %v", mock.ExecLog)
	}
}

func TestMock_ExecStream_Callback(t *testing.T) {
	mock := NewMock()
	mock.ExecResults["cmd run"] = ExecResult{Output: "line1\nline2\nline3"}

	var lines []string
	err := mock.ExecStream(nil, func(line string) {
		lines = append(lines, line)
	}, "cmd", "run")

	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("esperava 3 linhas, obteve %d: %v", len(lines), lines)
	}
}
