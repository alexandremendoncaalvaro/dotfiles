package tui

import (
	"fmt"

	"github.com/ale/blueprint/internal/module"
	"github.com/ale/blueprint/internal/profile"
	tea "github.com/charmbracelet/bubbletea"
)

// screen define as telas possiveis do TUI.
type screen int

const (
	screenWelcome screen = iota
	screenProfileSelect
	screenModuleConfirm
	screenExecute
	screenSummary
)

// model e o Model raiz do Bubble Tea (state machine de telas).
type model struct {
	screen       screen
	registry     *module.Registry
	modules      []module.Module
	sys          module.System
	profile      profile.Profile
	autoDetected bool

	// Estado das telas
	welcome       welcomeModel
	profileSelect profileSelectModel
	moduleConfirm moduleConfirmModel
	execute       executeModel
	summary       summaryModel
}

// Run inicia o TUI interativo.
// Recebe o registry completo para que a troca de perfil no TUI funcione corretamente.
// Se autoDetected=true, pula a selecao de perfil e vai direto para confirmacao de modulos.
func Run(registry *module.Registry, sys module.System, prof profile.Profile, autoDetected bool) error {
	modules := profile.Resolve(prof, registry)

	// Se o perfil foi auto-detectado, pula direto para confirmacao de modulos
	initialScreen := screenWelcome
	if autoDetected {
		initialScreen = screenModuleConfirm
	}

	m := model{
		screen:       initialScreen,
		registry:     registry,
		modules:      modules,
		sys:          sys,
		profile:      prof,
		autoDetected: autoDetected,
		welcome:      newWelcomeModel(),
	}

	if autoDetected {
		m.moduleConfirm = newModuleConfirmModel(modules)
		m.moduleConfirm.profile = prof
		m.moduleConfirm.autoDetected = true
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("erro no TUI: %w", err)
	}

	// Verifica se houve erros na execucao
	if fm, ok := finalModel.(model); ok && fm.summary.hasErrors {
		return fmt.Errorf("execucao concluida com erros")
	}

	return nil
}

func (m model) Init() tea.Cmd {
	return m.welcome.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Ctrl+C sai de qualquer tela
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenWelcome:
		return m.updateWelcome(msg)
	case screenProfileSelect:
		return m.updateProfileSelect(msg)
	case screenModuleConfirm:
		return m.updateModuleConfirm(msg)
	case screenExecute:
		return m.updateExecute(msg)
	case screenSummary:
		return m.updateSummary(msg)
	}

	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenWelcome:
		return m.welcome.View()
	case screenProfileSelect:
		return m.profileSelect.View()
	case screenModuleConfirm:
		return m.moduleConfirm.View()
	case screenExecute:
		return m.execute.View()
	case screenSummary:
		return m.summary.View()
	}
	return ""
}

// Transicoes entre telas

func (m model) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.welcome, cmd = m.welcome.Update(msg)

	if m.welcome.done {
		m.screen = screenProfileSelect
		m.profileSelect = newProfileSelectModel()
		return m, m.profileSelect.Init()
	}

	return m, cmd
}

func (m model) updateProfileSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.profileSelect, cmd = m.profileSelect.Update(msg)

	if m.profileSelect.done {
		m.profile = m.profileSelect.selected
		// Resolve modulos a partir do registry completo
		m.modules = profile.Resolve(m.profile, m.registry)

		m.screen = screenModuleConfirm
		m.moduleConfirm = newModuleConfirmModel(m.modules)
		m.moduleConfirm.profile = m.profile
		return m, nil
	}

	return m, cmd
}

func (m model) updateModuleConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.moduleConfirm, cmd = m.moduleConfirm.Update(msg)

	if m.moduleConfirm.done {
		selectedModules := m.moduleConfirm.selectedModules()
		m.screen = screenExecute
		m.execute = newExecuteModel(selectedModules, m.sys)
		return m, m.execute.Init()
	}

	return m, cmd
}

func (m model) updateExecute(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.execute, cmd = m.execute.Update(msg)

	if m.execute.done {
		m.screen = screenSummary
		m.summary = newSummaryModel(m.execute.results)
		return m, nil
	}

	return m, cmd
}

func (m model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.summary, cmd = m.summary.Update(msg)

	if m.summary.done {
		return m, tea.Quit
	}

	return m, cmd
}

