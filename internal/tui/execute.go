package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/blueprint/internal/module"
	"github.com/ale/blueprint/internal/orchestrator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Eventos enviados pela goroutine de processamento.

type logEvent struct {
	style string // "info", "success", "warn", "error", "step"
	text  string
}

type resultEvent struct {
	index  int
	result orchestrator.Result
}

type allDoneEvent struct{}

// setRunningEvent marca um modulo como "running".
type setRunningEvent struct {
	index int
}

// channelReporter implementa module.Reporter enviando eventos pelo channel.
type channelReporter struct {
	ch chan<- any
}

func (r *channelReporter) Info(msg string) {
	r.ch <- logEvent{style: "info", text: msg}
}

func (r *channelReporter) Success(msg string) {
	r.ch <- logEvent{style: "success", text: msg}
}

func (r *channelReporter) Warn(msg string) {
	r.ch <- logEvent{style: "warn", text: msg}
}

func (r *channelReporter) Error(msg string) {
	r.ch <- logEvent{style: "error", text: msg}
}

func (r *channelReporter) Step(current, total int, msg string) {
	r.ch <- logEvent{style: "step", text: fmt.Sprintf("[%d/%d] %s", current, total, msg)}
}

// moduleStatus representa o estado de um modulo durante a execucao.
type moduleStatus int

const (
	statusPending moduleStatus = iota
	statusRunning
	statusDone
	statusSkipped
	statusError
)

// moduleState rastreia o estado de cada modulo durante a execucao.
type moduleState struct {
	mod     module.Module
	status  moduleStatus
	message string // resumo (ex: "ja instalado", "sem sessao grafica")
}

// executeModel mostra o progresso da execucao com atualizacao em tempo real.
type executeModel struct {
	states  []moduleState
	sys     module.System
	spinner spinner.Model
	ch      chan any
	logs    []string // ultimas linhas de log do modulo atual (max 5)
	results []orchestrator.Result
	done    bool
}

const maxLogLines = 5

func newExecuteModel(modules []module.Module, sys module.System) executeModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = highlightStyle

	states := make([]moduleState, len(modules))
	for i, mod := range modules {
		states[i] = moduleState{mod: mod, status: statusPending}
	}

	return executeModel{
		states:  states,
		sys:     sys,
		spinner: s,
		ch:      make(chan any, 16),
	}
}

func (m executeModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.processModules(),
		m.waitForEvent(),
	)
}

func (m executeModel) Update(msg tea.Msg) (executeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case setRunningEvent:
		m.states[msg.index].status = statusRunning
		m.logs = nil
		return m, m.waitForEvent()

	case logEvent:
		m.logs = append(m.logs, m.formatLog(msg))
		if len(m.logs) > maxLogLines {
			m.logs = m.logs[len(m.logs)-maxLogLines:]
		}
		return m, m.waitForEvent()

	case resultEvent:
		r := msg.result
		st := &m.states[msg.index]

		switch {
		case r.Skipped:
			st.status = statusSkipped
			st.message = r.Reason
		case r.Err != nil:
			st.status = statusError
			st.message = r.Err.Error()
		case r.Applied:
			st.status = statusDone
			st.message = "aplicado"
		default:
			st.status = statusDone
			st.message = "ja instalado"
		}

		m.results = append(m.results, r)
		m.logs = nil
		return m, m.waitForEvent()

	case allDoneEvent:
		m.done = true
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m executeModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Executando..."))
	b.WriteString("\n\n")

	if !m.done {
		b.WriteString(m.spinner.View())
		b.WriteString(" Aplicando configuracoes...\n\n")
	}

	for _, st := range m.states {
		var icon, line string

		switch st.status {
		case statusDone:
			icon = successStyle.Render("✓")
			line = fmt.Sprintf("%s %s", st.mod.Name(), mutedStyle.Render("— "+st.message))
		case statusSkipped:
			icon = warningStyle.Render("⊘")
			line = fmt.Sprintf("%s %s", st.mod.Name(), mutedStyle.Render("— "+st.message))
		case statusError:
			icon = errorStyle.Render("✗")
			line = fmt.Sprintf("%s %s", st.mod.Name(), errorStyle.Render("— "+st.message))
		case statusRunning:
			icon = m.spinner.View()
			line = st.mod.Name()
		default: // statusPending
			icon = mutedStyle.Render("·")
			line = mutedStyle.Render(st.mod.Name())
		}

		b.WriteString(fmt.Sprintf("    %s %s\n", icon, line))
	}

	if len(m.logs) > 0 {
		b.WriteString("\n")
		for _, l := range m.logs {
			b.WriteString(fmt.Sprintf("    %s\n", l))
		}
	}

	return boxStyle.Render(b.String())
}

// processModules processa os modulos um a um, replicando a logica do
// orchestrator.runOne (Guard → Check → Apply). Manter em sincronia com
// orchestrator.go se a logica de execucao mudar.
//
// Nota: a goroutine nao e cancelavel. Em caso de Ctrl+C o processo encerra
// e a goroutine morre junto — aceitavel para uma ferramenta CLI.
func (m executeModel) processModules() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		reporter := &channelReporter{ch: m.ch}

		for i, st := range m.states {
			mod := st.mod

			// 1. Guard — decide antes de marcar como running
			if guard, ok := mod.(module.Guard); ok {
				shouldRun, reason := guard.ShouldRun(ctx, m.sys)
				if !shouldRun {
					m.ch <- resultEvent{
						index: i,
						result: orchestrator.Result{
							Module:  mod,
							Skipped: true,
							Reason:  reason,
							Status:  module.Status{Kind: module.Skipped, Message: reason},
						},
					}
					continue
				}
			}

			// Marca como running so depois do guard passar
			m.ch <- setRunningEvent{index: i}

			// 2. Check
			if checker, ok := mod.(module.Checker); ok {
				status, err := checker.Check(ctx, m.sys)
				if err != nil {
					reporter.Error(fmt.Sprintf("%s: erro ao verificar — %v", mod.Name(), err))
					m.ch <- resultEvent{
						index: i,
						result: orchestrator.Result{
							Module: mod,
							Err:    err,
						},
					}
					continue
				}

				if status.Kind == module.Installed {
					m.ch <- resultEvent{
						index: i,
						result: orchestrator.Result{
							Module: mod,
							Status: status,
						},
					}
					continue
				}
			}

			// 3. Apply
			if applier, ok := mod.(module.Applier); ok {
				if err := applier.Apply(ctx, m.sys, reporter); err != nil {
					reporter.Error(fmt.Sprintf("%s: erro ao aplicar — %v", mod.Name(), err))
					m.ch <- resultEvent{
						index: i,
						result: orchestrator.Result{
							Module: mod,
							Err:    err,
						},
					}
					continue
				}
				m.ch <- resultEvent{
					index: i,
					result: orchestrator.Result{
						Module:  mod,
						Applied: true,
					},
				}
			} else {
				// Modulo sem Apply — marca como done
				m.ch <- resultEvent{
					index: i,
					result: orchestrator.Result{Module: mod},
				}
			}
		}

		close(m.ch)
		return nil
	}
}

// waitForEvent le um evento do channel e retorna como tea.Msg.
func (m executeModel) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.ch
		if !ok {
			return allDoneEvent{}
		}
		return msg
	}
}

// formatLog formata uma logEvent para exibicao.
func (m executeModel) formatLog(ev logEvent) string {
	switch ev.style {
	case "success":
		return successStyle.Render("[OK] " + ev.text)
	case "warn":
		return warningStyle.Render("[WARN] " + ev.text)
	case "error":
		return errorStyle.Render("[ERRO] " + ev.text)
	case "step":
		return highlightStyle.Render(ev.text)
	default:
		return mutedStyle.Render(ev.text)
	}
}
