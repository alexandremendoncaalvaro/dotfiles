// Package orchestrator executa modulos em sequencia: Guard -> Check -> Apply.
package orchestrator

import (
	"context"
	"fmt"

	"github.com/ale/blueprint/internal/module"
)

// Result armazena o resultado da execucao de um modulo.
type Result struct {
	Module  module.Module
	Status  module.Status
	Applied bool
	Skipped bool
	Reason  string
	Err     error
	Notes   []string // Instrucoes pos-apply exibidas no sumario final
}

// notingReporter captura mensagens Info como Notes alem de repassar ao reporter original.
type notingReporter struct {
	inner module.Reporter
	notes []string
}

func (r *notingReporter) Info(msg string) {
	r.notes = append(r.notes, msg)
	r.inner.Info(msg)
}

func (r *notingReporter) Success(msg string)                   { r.inner.Success(msg) }
func (r *notingReporter) Warn(msg string)                      { r.inner.Warn(msg) }
func (r *notingReporter) Error(msg string)                     { r.inner.Error(msg) }
func (r *notingReporter) Step(current, total int, msg string)  { r.inner.Step(current, total, msg) }

// Orchestrator coordena a execucao de modulos.
type Orchestrator struct {
	sys      module.System
	reporter module.Reporter
}

// New cria um Orchestrator.
func New(sys module.System, reporter module.Reporter) *Orchestrator {
	return &Orchestrator{sys: sys, reporter: reporter}
}

// Run executa uma lista de modulos em sequencia.
// Para cada modulo: Guard -> Check -> Apply (se necessario).
func (o *Orchestrator) Run(ctx context.Context, modules []module.Module) []Result {
	var results []Result
	total := len(modules)

	for i, m := range modules {
		o.reporter.Step(i+1, total, fmt.Sprintf("Processando %s...", m.Name()))
		result := o.runOne(ctx, m)
		results = append(results, result)
	}

	return results
}

// CheckAll verifica o status de todos os modulos sem aplicar.
func (o *Orchestrator) CheckAll(ctx context.Context, modules []module.Module) []Result {
	var results []Result

	for _, m := range modules {
		result := Result{Module: m}

		// Verifica guard
		if guard, ok := m.(module.Guard); ok {
			shouldRun, reason := guard.ShouldRun(ctx, o.sys)
			if !shouldRun {
				result.Skipped = true
				result.Reason = reason
				result.Status = module.Status{Kind: module.Skipped, Message: reason}
				results = append(results, result)
				continue
			}
		}

		// Verifica status
		if checker, ok := m.(module.Checker); ok {
			status, err := checker.Check(ctx, o.sys)
			if err != nil {
				result.Err = err
			}
			result.Status = status
		}

		results = append(results, result)
	}

	return results
}

func (o *Orchestrator) runOne(ctx context.Context, m module.Module) Result {
	result := Result{Module: m}

	// 1. Guard: verifica se deve executar
	if guard, ok := m.(module.Guard); ok {
		shouldRun, reason := guard.ShouldRun(ctx, o.sys)
		if !shouldRun {
			o.reporter.Warn(fmt.Sprintf("%s: pulado — %s", m.Name(), reason))
			result.Skipped = true
			result.Reason = reason
			result.Status = module.Status{Kind: module.Skipped, Message: reason}
			return result
		}
	}

	// 2. Check: verifica estado atual
	if checker, ok := m.(module.Checker); ok {
		status, err := checker.Check(ctx, o.sys)
		if err != nil {
			o.reporter.Error(fmt.Sprintf("%s: erro ao verificar — %v", m.Name(), err))
			result.Err = err
			return result
		}
		result.Status = status

		if status.Kind == module.Installed {
			o.reporter.Success(fmt.Sprintf("%s: ja instalado", m.Name()))
			return result
		}
	}

	// 3. Apply: aplica mudancas
	if applier, ok := m.(module.Applier); ok {
		o.reporter.Info(fmt.Sprintf("%s: aplicando...", m.Name()))
		nr := &notingReporter{inner: o.reporter}
		if err := applier.Apply(ctx, o.sys, nr); err != nil {
			o.reporter.Error(fmt.Sprintf("%s: erro ao aplicar — %v", m.Name(), err))
			result.Err = err
			return result
		}
		result.Applied = true
		result.Notes = nr.notes
		o.reporter.Success(fmt.Sprintf("%s: aplicado com sucesso", m.Name()))
	}

	return result
}
