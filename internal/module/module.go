// Package module define as interfaces centrais do dominio.
// Cada modulo de dotfiles implementa essas interfaces para
// declarar o que faz, verificar estado e aplicar mudancas.
package module

import (
	"context"
	"os"
)

// System define as operacoes de sistema que os modulos podem usar.
// A interface pertence ao dominio; implementacoes concretas (real, mock, dry-run)
// ficam no pacote system.
type System interface {
	// Exec executa um comando e retorna a saida combinada.
	Exec(ctx context.Context, name string, args ...string) (string, error)

	// ExecStream executa um comando e chama callback para cada linha de saida.
	ExecStream(ctx context.Context, callback func(line string), name string, args ...string) error

	// FileExists verifica se um arquivo ou diretorio existe.
	FileExists(path string) bool

	// ReadFile le o conteudo de um arquivo.
	ReadFile(path string) ([]byte, error)

	// WriteFile escreve conteudo em um arquivo.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// MkdirAll cria diretorios recursivamente.
	MkdirAll(path string, perm os.FileMode) error

	// Symlink cria um link simbolico de oldname para newname.
	Symlink(oldname, newname string) error

	// HomeDir retorna o diretorio home do usuario.
	HomeDir() string

	// IsContainer retorna true se estiver rodando dentro de um container.
	IsContainer() bool

	// Env retorna o valor de uma variavel de ambiente.
	Env(key string) string

	// CommandExists verifica se um comando esta disponivel no PATH.
	CommandExists(name string) bool

	// AppendToFileIfMissing adiciona uma linha ao arquivo se ela nao existir.
	// Retorna true se a linha foi adicionada.
	AppendToFileIfMissing(path, line string) (bool, error)
}

// Module representa um modulo de configuracao.
// Todo modulo deve ter nome, descricao e tags para filtragem por perfil.
type Module interface {
	Name() string
	Description() string
	Tags() []string
}

// Guard verifica se o modulo deve ser executado no ambiente atual.
// Retorna false + motivo quando o modulo deve ser pulado.
type Guard interface {
	ShouldRun(ctx context.Context, sys System) (bool, string)
}

// Checker verifica o estado atual do modulo no sistema.
type Checker interface {
	Check(ctx context.Context, sys System) (Status, error)
}

// Applier aplica as mudancas do modulo no sistema.
type Applier interface {
	Apply(ctx context.Context, sys System, reporter Reporter) error
}
