package tui

import (
	"fmt"
	"strings"

	"github.com/ale/blueprint/internal/orchestrator"
	tea "github.com/charmbracelet/bubbletea"
)

// summaryModel mostra o resumo final da execucao.
type summaryModel struct {
	results   []orchestrator.Result
	done      bool
	hasErrors bool
}

func newSummaryModel(results []orchestrator.Result) summaryModel {
	hasErrors := false
	for _, r := range results {
		if r.Err != nil {
			hasErrors = true
			break
		}
	}
	return summaryModel{results: results, hasErrors: hasErrors}
}

func (m summaryModel) Init() tea.Cmd {
	return nil
}

func (m summaryModel) Update(msg tea.Msg) (summaryModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter", "q", "esc":
			m.done = true
		}
	}
	return m, nil
}

func (m summaryModel) View() string {
	var b strings.Builder

	if m.hasErrors {
		b.WriteString(titleStyle.Render("Concluido com erros"))
	} else {
		b.WriteString(titleStyle.Render("Concluido!"))
	}
	b.WriteString("\n\n")

	for _, r := range m.results {
		var icon, status string

		switch {
		case r.Skipped:
			icon = warningStyle.Render("[SKIP]")
			status = mutedStyle.Render(r.Reason)
		case r.Err != nil:
			icon = errorStyle.Render("[ERRO]")
			status = errorStyle.Render(r.Err.Error())
		case r.Applied:
			icon = successStyle.Render("[OK]  ")
			status = successStyle.Render("aplicado")
		default:
			icon = successStyle.Render("[OK]  ")
			status = mutedStyle.Render("ja instalado")
		}

		b.WriteString(fmt.Sprintf("  %s %s — %s\n", icon, r.Module.Name(), status))
	}

	// Notas pos-apply (instrucoes importantes para o usuario)
	var allNotes []string
	for _, r := range m.results {
		allNotes = append(allNotes, r.Notes...)
	}
	if len(allNotes) > 0 {
		b.WriteString("\n")
		b.WriteString(highlightStyle.Render("Proximos passos:"))
		b.WriteString("\n")
		for _, note := range allNotes {
			b.WriteString(fmt.Sprintf("  %s\n", note))
		}
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Pressione ENTER ou q para sair"))

	return boxStyle.Render(b.String())
}
