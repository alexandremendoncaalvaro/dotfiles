package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/ale/dotfiles/internal/module"
	"github.com/ale/dotfiles/internal/system"
)

// testReporter coleta mensagens para verificacao nos testes.
type testReporter struct {
	messages []string
}

func (r *testReporter) Info(msg string)    { r.messages = append(r.messages, "INFO: "+msg) }
func (r *testReporter) Success(msg string) { r.messages = append(r.messages, "OK: "+msg) }
func (r *testReporter) Warn(msg string)    { r.messages = append(r.messages, "WARN: "+msg) }
func (r *testReporter) Error(msg string)   { r.messages = append(r.messages, "ERR: "+msg) }
func (r *testReporter) Step(current, total int, msg string) {
	r.messages = append(r.messages, fmt.Sprintf("STEP %d/%d: %s", current, total, msg))
}

// fakeModule implementa Module + Checker + Applier para testes.
type fakeModule struct {
	name        string
	tags        []string
	checkStatus module.Status
	checkErr    error
	applyErr    error
	applied     bool
}

func (f *fakeModule) Name() string        { return f.name }
func (f *fakeModule) Description() string { return "modulo de teste" }
func (f *fakeModule) Tags() []string      { return f.tags }

func (f *fakeModule) Check(_ context.Context, _ module.System) (module.Status, error) {
	return f.checkStatus, f.checkErr
}

func (f *fakeModule) Apply(_ context.Context, _ module.System, _ module.Reporter) error {
	f.applied = true
	return f.applyErr
}

// fakeGuardedModule adiciona Guard ao fakeModule.
type fakeGuardedModule struct {
	fakeModule
	shouldRun bool
	reason    string
}

func (f *fakeGuardedModule) ShouldRun(_ context.Context, _ module.System) (bool, string) {
	return f.shouldRun, f.reason
}

func TestRun_ApplyMissing(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeModule{
		name:        "test-mod",
		tags:        []string{"shell"},
		checkStatus: module.Status{Kind: module.Missing},
	}

	results := orch.Run(context.Background(), []module.Module{mod})

	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, obteve %d", len(results))
	}

	if !results[0].Applied {
		t.Error("esperava que o modulo fosse aplicado")
	}

	if !mod.applied {
		t.Error("esperava que Apply fosse chamado")
	}
}

func TestRun_SkipInstalled(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeModule{
		name:        "test-mod",
		tags:        []string{"shell"},
		checkStatus: module.Status{Kind: module.Installed},
	}

	results := orch.Run(context.Background(), []module.Module{mod})

	if results[0].Applied {
		t.Error("nao deveria aplicar modulo ja instalado")
	}

	if mod.applied {
		t.Error("Apply nao deveria ser chamado")
	}
}

func TestRun_GuardSkips(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeGuardedModule{
		fakeModule: fakeModule{
			name: "guarded-mod",
			tags: []string{"desktop"},
		},
		shouldRun: false,
		reason:    "dentro de container",
	}

	results := orch.Run(context.Background(), []module.Module{mod})

	if !results[0].Skipped {
		t.Error("esperava que o modulo fosse pulado")
	}

	if results[0].Reason != "dentro de container" {
		t.Errorf("motivo errado: %s", results[0].Reason)
	}
}

func TestRun_ApplyError(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeModule{
		name:        "fail-mod",
		tags:        []string{"shell"},
		checkStatus: module.Status{Kind: module.Missing},
		applyErr:    fmt.Errorf("erro simulado"),
	}

	results := orch.Run(context.Background(), []module.Module{mod})

	if results[0].Err == nil {
		t.Error("esperava erro no resultado")
	}
}

func TestCheckAll(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod1 := &fakeModule{
		name:        "mod1",
		tags:        []string{"shell"},
		checkStatus: module.Status{Kind: module.Installed},
	}
	mod2 := &fakeGuardedModule{
		fakeModule: fakeModule{
			name: "mod2",
			tags: []string{"desktop"},
		},
		shouldRun: false,
		reason:    "container",
	}

	results := orch.CheckAll(context.Background(), []module.Module{mod1, mod2})

	if len(results) != 2 {
		t.Fatalf("esperava 2 resultados, obteve %d", len(results))
	}

	if results[0].Status.Kind != module.Installed {
		t.Error("mod1 deveria estar instalado")
	}

	if !results[1].Skipped {
		t.Error("mod2 deveria estar pulado")
	}
}

func TestRun_CheckError(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeModule{
		name:     "fail-check",
		tags:     []string{"shell"},
		checkErr: fmt.Errorf("check falhou"),
	}

	results := orch.Run(context.Background(), []module.Module{mod})

	if results[0].Err == nil {
		t.Error("esperava erro no resultado quando Check falha")
	}
	if mod.applied {
		t.Error("Apply nao deveria ser chamado quando Check retorna erro")
	}
}

func TestRun_PartialStatus(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeModule{
		name:        "partial-mod",
		tags:        []string{"shell"},
		checkStatus: module.Status{Kind: module.Partial, Message: "parcialmente configurado"},
	}

	results := orch.Run(context.Background(), []module.Module{mod})

	if !results[0].Applied {
		t.Error("modulo Partial deveria ser aplicado")
	}
	if !mod.applied {
		t.Error("Apply deveria ser chamado para status Partial")
	}
}

func TestCheckAll_CheckerError(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &fakeModule{
		name:     "err-check",
		tags:     []string{"shell"},
		checkErr: fmt.Errorf("erro ao verificar"),
	}

	results := orch.CheckAll(context.Background(), []module.Module{mod})

	if results[0].Err == nil {
		t.Error("esperava erro no resultado de CheckAll quando checker falha")
	}
}

// bareModule implementa apenas Module (sem Checker, Guard ou Applier).
type bareModule struct {
	name string
}

func (b *bareModule) Name() string        { return b.name }
func (b *bareModule) Description() string { return "modulo sem interfaces extras" }
func (b *bareModule) Tags() []string      { return []string{"test"} }

func TestRun_BareModule(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &bareModule{name: "bare"}

	results := orch.Run(context.Background(), []module.Module{mod})

	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, obteve %d", len(results))
	}
	if results[0].Applied {
		t.Error("bare module sem Applier nao deveria ser marcado como aplicado")
	}
	if results[0].Skipped {
		t.Error("bare module sem Guard nao deveria ser pulado")
	}
	if results[0].Err != nil {
		t.Errorf("nao deveria ter erro: %v", results[0].Err)
	}
}

func TestCheckAll_BareModule(t *testing.T) {
	mock := system.NewMock()
	reporter := &testReporter{}
	orch := New(mock, reporter)

	mod := &bareModule{name: "bare"}

	results := orch.CheckAll(context.Background(), []module.Module{mod})

	if results[0].Skipped {
		t.Error("bare module nao deveria ser pulado")
	}
	// Sem Checker, status fica zero-value
	if results[0].Status.Kind != module.Installed {
		// zero-value de StatusKind e 0 = Installed
		t.Logf("status de bare module: %s (zero-value esperado)", results[0].Status.Kind)
	}
}
