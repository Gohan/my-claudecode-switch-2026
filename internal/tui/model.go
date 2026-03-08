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
	viewSaveOverwrite    // 覆盖确认（显示diff）
	viewSaveNewName      // 另存为新名字
	viewSaveZAI
	viewSaveTencentCloud
	viewConfirmApply
	viewConfirmDelete
)

type Model struct {
	state            viewState
	profiles         []profile.Profile
	current          map[string]interface{}
	cursor           int
	input            textinput.Model
	apiKeyInput      textinput.Model
	zaiStep          int // 0: 输入 name, 1: 输入 api key
	tencentStep      int // 0: 输入 name, 1: 输入 api key
	pendingSaveName  string           // 待保存的名字（用于覆盖/另存为）
	existingProfile  *profile.Profile // 已存在的 profile（用于显示 diff）
	saveOriginalName string           // 保存界面预填的原始名字（用于 hint 样式）
	message          string
	width            int
	height           int
}

// safeProfileIndex 检查 cursor 是否在有效范围内
func (m Model) safeProfileIndex() bool {
	return m.cursor >= 0 && m.cursor < len(m.profiles)
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

	profiles, loadErrors := profile.List()
	m.profiles = profiles

	// 如果有加载错误，显示警告信息
	if len(loadErrors) > 0 {
		var errMsgs []string
		for _, e := range loadErrors {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", e.Name, e.Err))
		}
		m.message = fmt.Sprintf("Warning: failed to load %d profile(s): %s", len(loadErrors), strings.Join(errMsgs, "; "))
	} else {
		m.message = ""
	}
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
	case viewSaveOverwrite:
		return m.updateSaveOverwrite(msg)
	case viewSaveNewName:
		return m.updateSaveNewName(msg)
	case viewSaveZAI:
		return m.updateSaveZAI(msg)
	case viewSaveTencentCloud:
		return m.updateSaveTencentCloud(msg)
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
		if m.safeProfileIndex() {
			m.state = viewConfirmApply
		}
	case "p":
		if m.safeProfileIndex() {
			m.state = viewPreview
		}
	case "s":
		m.state = viewSave
		// 如果有选中的 profile，将其名字设为 placeholder
		if m.safeProfileIndex() {
			m.saveOriginalName = m.profiles[m.cursor].Name
			m.input.Placeholder = m.profiles[m.cursor].Name
		} else {
			m.saveOriginalName = ""
			m.input.Placeholder = "profile-name"
		}
		m.input.SetValue("")
		m.input.Focus()
		return m, textinput.Blink
	case "z":
		m.state = viewSaveZAI
		m.zaiStep = 0
		m.input.SetValue("")
		m.input.Placeholder = "z.ai Coding Plan"
		m.saveOriginalName = "z.ai Coding Plan"
		m.apiKeyInput.SetValue("")
		m.input.Focus()
		return m, textinput.Blink
	case "t":
		m.state = viewSaveTencentCloud
		m.tencentStep = 0
		m.input.SetValue("")
		m.input.Placeholder = "Tencent Coding Plan"
		m.saveOriginalName = "Tencent Coding Plan"
		m.apiKeyInput.SetValue("")
		m.input.Focus()
		return m, textinput.Blink
	case "d":
		if m.safeProfileIndex() {
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
			m.saveOriginalName = ""
			m.input.Placeholder = "profile-name"
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			// 如果输入为空但有原始名字（placeholder），使用它
			if name == "" && m.saveOriginalName != "" {
				name = m.saveOriginalName
			}
			if name == "" {
				m.message = "Profile name cannot be empty"
				return m, nil
			}
			// 检查是否存在同名 profile
			for i, p := range m.profiles {
				if p.Name == name {
					m.pendingSaveName = name
					m.existingProfile = &m.profiles[i]
					m.state = viewSaveOverwrite
					m.saveOriginalName = ""
					m.input.Placeholder = "profile-name"
					return m, nil
				}
			}
			// 不存在，直接保存
			if err := profile.Save(name, m.current); err != nil {
				m.message = fmt.Sprintf("Error: %v", err)
			} else {
				m.message = fmt.Sprintf("✓ Saved profile: %s", name)
				m.loadData()
			}
			m.state = viewList
			m.saveOriginalName = ""
			m.input.Placeholder = "profile-name"
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// updateSaveOverwrite 处理覆盖确认界面
func (m Model) updateSaveOverwrite(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "y", "Y":
		// 确认覆盖
		if err := profile.Save(m.pendingSaveName, m.current); err != nil {
			m.message = fmt.Sprintf("Error: %v", err)
		} else {
			m.message = fmt.Sprintf("✓ Overwritten profile: %s", m.pendingSaveName)
			m.loadData()
		}
		m.state = viewList
		m.pendingSaveName = ""
		m.existingProfile = nil
		return m, nil
	case "n", "N":
		// 另存为新名字
		m.state = viewSaveNewName
		m.input.SetValue("")
		m.input.Focus()
		return m, textinput.Blink
	case "esc":
		// 取消
		m.state = viewList
		m.pendingSaveName = ""
		m.existingProfile = nil
		return m, nil
	}
	return m, nil
}

// updateSaveNewName 处理另存为新名字界面
func (m Model) updateSaveNewName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.state = viewList
			m.pendingSaveName = ""
			m.existingProfile = nil
			m.message = ""
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				m.message = "Profile name cannot be empty"
				return m, nil
			}
			// 再次检查是否存在同名
			for _, p := range m.profiles {
				if p.Name == name {
					m.message = fmt.Sprintf("Profile '%s' already exists", name)
					return m, nil
				}
			}
			if err := profile.Save(name, m.current); err != nil {
				m.message = fmt.Sprintf("Error: %v", err)
			} else {
				m.message = fmt.Sprintf("✓ Saved profile: %s", name)
				m.loadData()
			}
			m.state = viewList
			m.pendingSaveName = ""
			m.existingProfile = nil
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// apiProfileConfig 定义 API profile 的配置
type apiProfileConfig struct {
	name             string
	step             *int
	defaultName      string
	profileGetter    func() map[string]interface{}
	successMessage   string
}

func (m *Model) updateSaveAPI(msg tea.Msg, cfg apiProfileConfig) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.state = viewList
			*cfg.step = 0
			m.input.SetValue("")
			m.apiKeyInput.SetValue("")
			m.message = ""
			m.saveOriginalName = ""
			m.input.Placeholder = "profile-name"
			return m, nil
		case "enter":
			if *cfg.step == 0 {
				// 第一步：检查 name
				name := strings.TrimSpace(m.input.Value())
				if name == "" && m.saveOriginalName != "" {
					name = m.saveOriginalName
					m.input.SetValue(name)
				}
				if name == "" {
					m.message = "Profile name cannot be empty"
					return m, nil
				}
				// 进入第二步：输入 API key
				*cfg.step = 1
				m.message = ""

				// 检查 profile 是否已存在，如果是则预填 API key
				if existing, err := profile.GetByName(name); err == nil {
					if env, ok := existing.Settings["env"].(map[string]interface{}); ok {
						if token, ok := env["ANTHROPIC_AUTH_TOKEN"].(string); ok && token != "" {
							m.apiKeyInput.SetValue(token)
						}
					}
				}

				m.apiKeyInput.Focus()
				return m, textinput.Blink
			} else {
				// 第二步：保存 profile
				name := strings.TrimSpace(m.input.Value())
				if name == "" && m.saveOriginalName != "" {
					name = m.saveOriginalName
				}
				apiKey := strings.TrimSpace(m.apiKeyInput.Value())
				if apiKey == "" {
					m.message = "API key cannot be empty"
					return m, nil
				}
				apiProfile := cfg.profileGetter()
				// 将用户输入的 API key 设置到 profile 中
				if env, ok := apiProfile["env"].(map[string]interface{}); ok {
					env["ANTHROPIC_AUTH_TOKEN"] = apiKey
				}
				if err := profile.Save(name, apiProfile); err != nil {
					m.message = fmt.Sprintf("Error: %v", err)
				} else {
					m.message = cfg.successMessage + ": " + name
					m.loadData()
				}
				m.state = viewList
				*cfg.step = 0
				m.input.SetValue("")
				m.apiKeyInput.SetValue("")
				m.saveOriginalName = ""
				m.input.Placeholder = "profile-name"
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if *cfg.step == 0 {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	}
	return m, cmd
}

func (m Model) updateSaveZAI(msg tea.Msg) (tea.Model, tea.Cmd) {
	cfg := apiProfileConfig{
		name:           "z.ai",
		step:           &m.zaiStep,
		defaultName:    "z.ai Coding Plan",
		profileGetter:  profile.DefaultZAIProfile,
		successMessage: "✓ Saved z.ai profile",
	}
	return m.updateSaveAPI(msg, cfg)
}

func (m Model) updateSaveTencentCloud(msg tea.Msg) (tea.Model, tea.Cmd) {
	cfg := apiProfileConfig{
		name:           "TencentCloud",
		step:           &m.tencentStep,
		defaultName:    "Tencent Coding Plan",
		profileGetter:  profile.DefaultTencentCloudProfile,
		successMessage: "✓ Saved TencentCloud profile",
	}
	return m.updateSaveAPI(msg, cfg)
}

func (m Model) updateConfirmApply(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "y", "Y":
		if !m.safeProfileIndex() {
			m.state = viewList
			return m, nil
		}
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
		if !m.safeProfileIndex() {
			m.state = viewList
			return m, nil
		}
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
	case viewSaveOverwrite:
		return m.viewSaveOverwrite()
	case viewSaveNewName:
		return m.viewSaveNewName()
	case viewSaveZAI:
		return m.viewSaveZAI()
	case viewSaveTencentCloud:
		return m.viewSaveTencentCloud()
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
	help := "[enter] apply  [p] preview  [s] save  [z] z.ai  [t] tencent  [d] delete  [q] quit"
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

	// 如果输入为空且有原始名字（作为 placeholder），显示提示
	inputVal := strings.TrimSpace(m.input.Value())
	if inputVal == "" && m.saveOriginalName != "" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  (press [enter] to overwrite '" + m.saveOriginalName + "' or type new name)"))
	}

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

func (m Model) viewSaveTencentCloud() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Create TencentCloud CodingPlan Profile"))
	b.WriteString("\n\n")

	if m.tencentStep == 0 {
		b.WriteString(dimStyle.Render("Step 1/2: Enter profile name"))
		b.WriteString("\n\n")
		b.WriteString("Profile name: ")
		b.WriteString(m.input.View())
	} else {
		b.WriteString(dimStyle.Render("Step 2/2: Enter your TencentCloud API key"))
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
	b.WriteString(dimStyle.Render("  • ANTHROPIC_BASE_URL: https://api.lkeap.cloud.tencent.com/coding/anthropic"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: 1"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • Model mappings: haiku→tc-code-latest, sonnet→kimi-k2.5, opus→minimax-m2.5"))

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

// viewSaveOverwrite 显示覆盖确认界面（带 diff 预览）
func (m Model) viewSaveOverwrite() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Overwrite Profile"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Profile '%s' already exists.\n\n", selectedStyle.Render(m.pendingSaveName)))
	b.WriteString(dimStyle.Render("Changes (saved profile → current settings):"))
	b.WriteString("\n\n")

	// 显示 diff：从已有 profile 到当前 settings
	if m.existingProfile != nil {
		diff := profile.Diff(m.existingProfile.Settings, m.current)

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
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("[y] overwrite  [n] save as new  [esc] cancel"))

	return b.String()
}

// viewSaveNewName 显示另存为新名字界面
func (m Model) viewSaveNewName() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Save as New Profile"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Original name: %s", m.pendingSaveName)))
	b.WriteString("\n\n")
	b.WriteString("New profile name: ")
	b.WriteString(m.input.View())

	if m.message != "" {
		b.WriteString("\n\n")
		b.WriteString(messageStyle.Render(m.message))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("[enter] save  [esc] cancel"))

	return b.String()
}
