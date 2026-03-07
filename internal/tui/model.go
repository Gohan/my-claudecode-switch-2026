package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"claude-switch/internal/profile"
)

type viewState int

const (
	viewList viewState = iota
	viewPreview
	viewSave
	viewConfirmApply
	viewConfirmDelete
)

type Model struct {
	state    viewState
	profiles []profile.Profile
	current  map[string]interface{}
	cursor   int
	input    textinput.Model
	message  string
	width    int
	height   int
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "profile-name"
	ti.CharLimit = 64

	m := Model{
		state: viewList,
		input: ti,
	}
	m.loadData()
	return m
}

func (m *Model) loadData() {
	current, err := profile.LoadCurrent()
	if err != nil {
		m.current = make(map[string]interface{})
	} else {
		m.current = current
	}

	profiles, _ := profile.List()
	m.profiles = profiles
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.state {
	case viewList:
		return m.updateList(msg)
	case viewPreview:
		return m.updatePreview(msg)
	case viewSave:
		return m.updateSave(msg)
	case viewConfirmApply:
		return m.updateConfirmApply(msg)
	case viewConfirmDelete:
		return m.updateConfirmDelete(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.profiles)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.profiles) > 0 {
			m.state = viewConfirmApply
		}
	case "p":
		if len(m.profiles) > 0 {
			m.state = viewPreview
		}
	case "s":
		m.state = viewSave
		m.input.SetValue("")
		m.input.Focus()
		return m, textinput.Blink
	case "d":
		if len(m.profiles) > 0 {
			m.state = viewConfirmDelete
		}
	}
	return m, nil
}

func (m Model) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc", "q":
		m.state = viewList
	case "enter":
		m.state = viewConfirmApply
	}
	return m, nil
}

func (m Model) updateSave(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.state = viewList
			m.message = ""
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				m.message = "Profile name cannot be empty"
				return m, nil
			}
			if err := profile.Save(name, m.current); err != nil {
				m.message = fmt.Sprintf("Error: %v", err)
			} else {
				m.message = fmt.Sprintf("✓ Saved profile: %s", name)
				m.loadData()
			}
			m.state = viewList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirmApply(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "y", "Y":
		p := m.profiles[m.cursor]
		if err := profile.ApplyProfile(p); err != nil {
			m.message = fmt.Sprintf("Error: %v", err)
		} else {
			m.message = fmt.Sprintf("✓ Applied profile: %s", p.Name)
			m.loadData()
		}
		m.state = viewList
	case "n", "N", "esc":
		m.state = viewList
	}
	return m, nil
}

func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "y", "Y":
		p := m.profiles[m.cursor]
		if err := profile.Delete(p.Name); err != nil {
			m.message = fmt.Sprintf("Error: %v", err)
		} else {
			m.message = fmt.Sprintf("✓ Deleted profile: %s", p.Name)
			m.loadData()
			if m.cursor >= len(m.profiles) && m.cursor > 0 {
				m.cursor--
			}
		}
		m.state = viewList
	case "n", "N", "esc":
		m.state = viewList
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case viewList:
		return m.viewList()
	case viewPreview:
		return m.viewPreview()
	case viewSave:
		return m.viewSave()
	case viewConfirmApply:
		return m.viewConfirmApply()
	case viewConfirmDelete:
		return m.viewConfirmDelete()
	}
	return ""
}

func (m Model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔧 Claude Code Settings Switch"))
	b.WriteString("\n\n")

	model, baseURL := profile.GetSummary(m.current)
	b.WriteString(dimStyle.Render("Current: "))
	if model != "" {
		b.WriteString(fmt.Sprintf("model=%s", model))
	}
	if baseURL != "" {
		b.WriteString(fmt.Sprintf(" | url=%s", baseURL))
	}
	if model == "" && baseURL == "" {
		b.WriteString(dimStyle.Render("(no settings found)"))
	}
	b.WriteString("\n\n")

	if len(m.profiles) == 0 {
		b.WriteString(dimStyle.Render("  No profiles saved. Press [s] to save current settings."))
		b.WriteString("\n")
	} else {
		for i, p := range m.profiles {
			cursor := "  "
			if i == m.cursor {
				cursor = "> "
			}

			isActive := profile.IsActive(m.current, p)

			icon := "○"
			nameStr := p.Name
			if isActive {
				icon = activeStyle.Render("●")
				nameStr = activeStyle.Render(p.Name + " [active]")
			} else if i == m.cursor {
				icon = selectedStyle.Render("○")
				nameStr = selectedStyle.Render(p.Name)
			}

			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, icon, nameStr))

			pModel, pURL := profile.GetSummary(p.Settings)
			var parts []string
			if pModel != "" {
				parts = append(parts, fmt.Sprintf("model: %s", pModel))
			}
			if pURL != "" {
				parts = append(parts, fmt.Sprintf("url: %s", pURL))
			}
			if len(parts) > 0 {
				b.WriteString(dimStyle.Render("    "+strings.Join(parts, " | ")) + "\n")
			}
		}
	}

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(messageStyle.Render(m.message))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := "[enter] apply  [p] preview  [s] save current  [d] delete  [q] quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) viewPreview() string {
	var b strings.Builder

	p := m.profiles[m.cursor]
	b.WriteString(titleStyle.Render(fmt.Sprintf("Preview: %s", p.Name)))
	b.WriteString("\n\n")

	diff := profile.Diff(m.current, p.Settings)

	var lines []string
	for _, d := range diff {
		switch d.Status {
		case profile.DiffUnchanged:
			val := profile.MaskSensitive(d.Key, d.OldValue)
			lines = append(lines, unchangedStyle.Render(fmt.Sprintf("  %s: %s", d.Key, val)))
		case profile.DiffModified:
			oldVal := profile.MaskSensitive(d.Key, d.OldValue)
			newVal := profile.MaskSensitive(d.Key, d.NewValue)
			lines = append(lines, fmt.Sprintf("  %s:", d.Key))
			lines = append(lines, removedStyle.Render(fmt.Sprintf("    - %s", oldVal)))
			lines = append(lines, addedStyle.Render(fmt.Sprintf("    + %s", newVal)))
		case profile.DiffAdded:
			newVal := profile.MaskSensitive(d.Key, d.NewValue)
			lines = append(lines, addedStyle.Render(fmt.Sprintf("  + %s: %s", d.Key, newVal)))
		case profile.DiffRemoved:
			oldVal := profile.MaskSensitive(d.Key, d.OldValue)
			lines = append(lines, removedStyle.Render(fmt.Sprintf("  - %s: %s", d.Key, oldVal)))
		}
	}

	content := strings.Join(lines, "\n")
	b.WriteString(boxStyle.Render(content))

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("[enter] apply  [esc] back"))

	return b.String()
}

func (m Model) viewSave() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Save Current Settings as Profile"))
	b.WriteString("\n\n")
	b.WriteString("Profile name: ")
	b.WriteString(m.input.View())

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(messageStyle.Render(m.message))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("[enter] save  [esc] cancel"))

	return b.String()
}

func (m Model) viewConfirmApply() string {
	var b strings.Builder

	p := m.profiles[m.cursor]
	b.WriteString(titleStyle.Render("Confirm Apply"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Apply profile '%s' to settings.json?\n\n", selectedStyle.Render(p.Name)))
	b.WriteString(helpStyle.Render("[y] yes  [n/esc] no"))

	return b.String()
}

func (m Model) viewConfirmDelete() string {
	var b strings.Builder

	p := m.profiles[m.cursor]
	b.WriteString(titleStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Delete profile '%s'?\n\n", removedStyle.Render(p.Name)))
	b.WriteString(helpStyle.Render("[y] yes  [n/esc] no"))

	return b.String()
}
