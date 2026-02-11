package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/blueprint/internal/module"
	"github.com/ale/blueprint/internal/orchestrator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Eventos enviados pela goroutine de processamento.

type logStyle int

const (
	logInfo logStyle = iota
	logSuccess
	logWarn
	logError
	logStep
)

type logEvent struct {
	style logStyle
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
	r.ch <- logEvent{style: logInfo, text: msg}
}

func (r *channelReporter) Success(msg string) {
	r.ch <- logEvent{style: logSuccess, text: msg}
}

func (r *channelReporter) Warn(msg string) {
	r.ch <- logEvent{style: logWarn, text: msg}
}

func (r *channelReporter) Error(msg string) {
	r.ch <- logEvent{style: logError, text: msg}
}

func (r *channelReporter) Step(current, total int, msg string) {
	r.ch <- logEvent{style: logStep, text: fmt.Sprintf("[%d/%d] %s", current, total, msg)}
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
	allLogs []logEvent // buffer acumulativo de TODOS os logs (nunca limpa)
	results []orchestrator.Result
	done    bool
	width   int
	height  int
}

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
		return m, m.waitForEvent()

	case logEvent:
		m.allLogs = append(m.allLogs, msg)
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
	// Dimensoes do terminal (fallback so quando ainda nao recebemos WindowSizeMsg)
	w := m.width
	if w == 0 {
		w = 80
	}
	h := m.height
	if h == 0 {
		h = 24
	}

	// Larguras dos paineis: esquerda ~40%, direita = resto.
	leftW := w * 40 / 100
	if leftW < 30 {
		leftW = 30
	}
	rightW := w - leftW

	// Paineis ocupam a altura inteira do terminal.
	panelH := h
	if panelH < 5 {
		panelH = 5
	}
	// Linhas de conteudo: panelH - 2 (bordas) - 1 (titulo)
	contentH := panelH - 3
	if contentH < 1 {
		contentH = 1
	}

	// --- Painel esquerdo: Progresso ---
	var left strings.Builder

	if !m.done {
		left.WriteString(m.spinner.View())
		left.WriteString(" Aplicando configuracoes...\n\n")
	} else {
		left.WriteString("Concluido.\n\n")
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

		left.WriteString(fmt.Sprintf("  %s %s\n", icon, line))
	}

	leftPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Width(leftW).
		Height(panelH).
		Render(lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("Progresso") + "\n" + left.String())

	// --- Painel direito: Log ---
	var right strings.Builder

	start := 0
	if len(m.allLogs) > contentH {
		start = len(m.allLogs) - contentH
	}

	for i := start; i < len(m.allLogs); i++ {
		right.WriteString("  " + m.formatLog(m.allLogs[i]) + "\n")
	}

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1).
		Width(rightW).
		Height(panelH).
		Render(lipgloss.NewStyle().Bold(true).Foreground(colorMuted).Render("Log") + "\n" + right.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
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
					m.ch <- logEvent{style: logWarn, text: fmt.Sprintf("%s: %s", mod.Name(), reason)}
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
			var checkStatus module.Status
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
				checkStatus = status

				if status.Kind == module.Installed {
					msg := status.Message
					if msg == "" {
						msg = "ja instalado"
					}
					m.ch <- logEvent{style: logSuccess, text: fmt.Sprintf("%s: %s", mod.Name(), msg)}
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
							Status: checkStatus,
							Err:    err,
						},
					}
					continue
				}
				m.ch <- resultEvent{
					index: i,
					result: orchestrator.Result{
						Module:  mod,
						Status:  checkStatus,
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
	case logSuccess:
		return successStyle.Render("[OK] " + ev.text)
	case logWarn:
		return warningStyle.Render("[WARN] " + ev.text)
	case logError:
		return errorStyle.Render("[ERRO] " + ev.text)
	case logStep:
		return highlightStyle.Render(ev.text)
	default:
		return mutedStyle.Render(ev.text)
	}
}
