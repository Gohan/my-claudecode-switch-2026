package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	addedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	removedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444"))

	unchangedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginTop(1)

	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

	// Modal styles for popup overlay
	modalBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(1, 2).
			Width(50)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	modalOverlayStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0F0F1A"))
)
