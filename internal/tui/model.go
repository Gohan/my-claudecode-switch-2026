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
	viewSaveZAI
	viewConfirmApply
	viewConfirmDelete
)

type Model struct {
	state     viewState
	profiles  []profile.Profile
	current   map[string]interface{}
	cursor    int
	input     textinput.Model
	apiKeyInput textinput.Model
	zaiStep   int // 0: 输入 name, 1: 输入 api key
	message   string
	width     int
	height    int
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "profile-name"
	ti.CharLimit = 64

	aki := textinput.New()
	aki.Placeholder = "your_zai_api_key"
	aki.CharLimit = 128
	aki.EchoMode = textinput.EchoPassword
	aki.EchoCharacter = '*'

	m := Model{
		state:       viewList,
		input:       ti,
		apiKeyInput: aki,
		zaiStep:     0,
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
	case viewSaveZAI:
		return m.updateSaveZAI(msg)
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
	case "z":
		m.state = viewSaveZAI
		m.zaiStep = 0
		m.input.SetValue("")
		m.apiKeyInput.SetValue("")
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

func (m Model) updateSaveZAI(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.state = viewList
			m.zaiStep = 0
			m.input.SetValue("")
			m.apiKeyInput.SetValue("")
			m.message = ""
			return m, nil
		case "enter":
			if m.zaiStep == 0 {
				// 第一步：检查 name
				name := strings.TrimSpace(m.input.Value())
				if name == "" {
					m.message = "Profile name cannot be empty"
					return m, nil
				}
				// 进入第二步：输入 API key
				m.zaiStep = 1
				m.message = ""
				m.apiKeyInput.Focus()
				return m, textinput.Blink
			} else {
				// 第二步：保存 profile
				name := strings.TrimSpace(m.input.Value())
				apiKey := strings.TrimSpace(m.apiKeyInput.Value())
				if apiKey == "" {
					m.message = "API key cannot be empty"
					return m, nil
				}
				zaiProfile := profile.DefaultZAIProfile()
				// 将用户输入的 API key 设置到 profile 中
				if env, ok := zaiProfile["env"].(map[string]interface{}); ok {
					env["ANTHROPIC_AUTH_TOKEN"] = apiKey
				}
				if err := profile.Save(name, zaiProfile); err != nil {
					m.message = fmt.Sprintf("Error: %v", err)
				} else {
					m.message = fmt.Sprintf("✓ Saved z.ai profile: %s", name)
					m.loadData()
				}
				m.state = viewList
				m.zaiStep = 0
				m.input.SetValue("")
				m.apiKeyInput.SetValue("")
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if m.zaiStep == 0 {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	}
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
	case viewSaveZAI:
		return m.viewSaveZAI()
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
			// 显示模型映射
			modelMap := profile.GetModelMapping(p.Settings)
			if len(modelMap) > 0 {
				var mappings []string
				if haiku, ok := modelMap["haiku"]; ok {
					mappings = append(mappings, fmt.Sprintf("haiku→%s", haiku))
				}
				if sonnet, ok := modelMap["sonnet"]; ok {
					mappings = append(mappings, fmt.Sprintf("sonnet→%s", sonnet))
				}
				if opus, ok := modelMap["opus"]; ok {
					mappings = append(mappings, fmt.Sprintf("opus→%s", opus))
				}
				if len(mappings) > 0 {
					parts = append(parts, fmt.Sprintf("models: %s", strings.Join(mappings, ", ")))
				}
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
	help := "[enter] apply  [p] preview  [s] save current  [z] new z.ai  [d] delete  [q] quit"
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

func (m Model) viewSaveZAI() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Create z.ai API Profile"))
	b.WriteString("\n\n")

	if m.zaiStep == 0 {
		b.WriteString(dimStyle.Render("Step 1/2: Enter profile name"))
		b.WriteString("\n\n")
		b.WriteString("Profile name: ")
		b.WriteString(m.input.View())
	} else {
		b.WriteString(dimStyle.Render("Step 2/2: Enter your z.ai API key"))
		b.WriteString("\n\n")
		b.WriteString("Profile name: ")
		b.WriteString(activeStyle.Render(m.input.Value()))
		b.WriteString("\n\n")
		b.WriteString("API Key: ")
		b.WriteString(m.apiKeyInput.View())
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  (The key will be masked for security)"))
	}

	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("This profile will include:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • ANTHROPIC_BASE_URL: https://api.z.ai/api/anthropic"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • API_TIMEOUT_MS: 3000000"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: 1"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • Model mappings: haiku→glm-4.5-air, sonnet→glm-4.7, opus→glm-5"))

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(messageStyle.Render(m.message))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[enter] next  [esc] cancel"))

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
