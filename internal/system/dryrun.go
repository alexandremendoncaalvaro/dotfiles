package system

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ale/blueprint/internal/module"
)

// DryRun envolve um System real mas intercepta operacoes de escrita,
// logando o que faria sem executar.
type DryRun struct {
	inner module.System
	log   func(msg string)
}

// NewDryRun cria um System que loga ao inves de executar.
func NewDryRun(inner module.System, log func(msg string)) *DryRun {
	return &DryRun{inner: inner, log: log}
}

// Exec loga o comando mas nao o executa. Retorna ("", nil).
// NOTA: em dry-run, qualquer logica que dependa da saida de Exec
// (ex: parsear output) vai receber string vazia e seguir como se
// tivesse sucesso. Isso e intencional — dry-run pula toda escrita.
func (d *DryRun) Exec(_ context.Context, name string, args ...string) (string, error) {
	d.log(fmt.Sprintf("[dry-run] executaria: %s %s", name, strings.Join(args, " ")))
	return "", nil
}

func (d *DryRun) ExecStream(_ context.Context, _ func(line string), name string, args ...string) error {
	d.log(fmt.Sprintf("[dry-run] executaria (stream): %s %s", name, strings.Join(args, " ")))
	return nil
}

// Operacoes de leitura delegam para o sistema real
func (d *DryRun) FileExists(path string) bool        { return d.inner.FileExists(path) }
func (d *DryRun) ReadFile(path string) ([]byte, error) { return d.inner.ReadFile(path) }
func (d *DryRun) HomeDir() string                     { return d.inner.HomeDir() }
func (d *DryRun) IsContainer() bool                   { return d.inner.IsContainer() }
func (d *DryRun) IsWSL() bool                         { return d.inner.IsWSL() }
func (d *DryRun) Env(key string) string               { return d.inner.Env(key) }
func (d *DryRun) CommandExists(name string) bool       { return d.inner.CommandExists(name) }

// Operacoes de escrita sao logadas mas nao executadas
func (d *DryRun) WriteFile(path string, _ []byte, _ os.FileMode) error {
	d.log(fmt.Sprintf("[dry-run] escreveria arquivo: %s", path))
	return nil
}

func (d *DryRun) MkdirAll(path string, _ os.FileMode) error {
	d.log(fmt.Sprintf("[dry-run] criaria diretorio: %s", path))
	return nil
}

func (d *DryRun) Symlink(oldname, newname string) error {
	d.log(fmt.Sprintf("[dry-run] criaria symlink: %s -> %s", newname, oldname))
	return nil
}

func (d *DryRun) AppendToFileIfMissing(path, line string) (bool, error) {
	// Verifica se ja existe (leitura real)
	data, err := d.inner.ReadFile(path)
	if err == nil && strings.Contains(string(data), line) {
		return false, nil
	}
	d.log(fmt.Sprintf("[dry-run] adicionaria ao %s: %s", path, line))
	return true, nil
}
