package styles

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	Primary   = lipgloss.Color("170") // Purple
	Secondary = lipgloss.Color("62")  // Blue
	Success   = lipgloss.Color("42")  // Green
	Warning   = lipgloss.Color("214") // Yellow/Orange
	Danger    = lipgloss.Color("196") // Red
	Muted     = lipgloss.Color("240") // Gray
	Text      = lipgloss.Color("255") // White
	Pinned    = lipgloss.Color("39")  // Light Blue
)

// Component styles
var (
	AppStyle = lipgloss.NewStyle() // No padding to use full screen

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Muted)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottom(true).
			BorderForeground(Secondary).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(Secondary).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 1)

	// Package list styles
	InstalledStyle = lipgloss.NewStyle().
			Foreground(Success)

	OutdatedStyle = lipgloss.NewStyle().
			Foreground(Warning)

	PinnedStyle = lipgloss.NewStyle().
			Foreground(Pinned)

	FormulaStyle = lipgloss.NewStyle().
			Foreground(Text)

	CaskStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")) // Light cyan

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	SuccessMessageStyle = lipgloss.NewStyle().
				Foreground(Success).
				Bold(true)

	// Interactive elements
	SelectedStyle = lipgloss.NewStyle().
			Background(Secondary).
			Foreground(Text).
			Bold(true)

	UnselectedStyle = lipgloss.NewStyle().
			Foreground(Text)

	// Dialog styles
	DialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2).
			Width(60)

	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				MarginBottom(1)

	DialogButtonStyle = lipgloss.NewStyle().
				Foreground(Text).
				Background(Secondary).
				Padding(0, 3).
				MarginRight(2)

	DialogButtonActiveStyle = lipgloss.NewStyle().
				Foreground(Text).
				Background(Primary).
				Padding(0, 3).
				MarginRight(2).
				Bold(true)

	// Info styles
	KeyStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(Text)

	DimStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// Panel styles (lazygit-like)
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Secondary).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1)
)

// Helper functions
func MaxWidth(width int) lipgloss.Style {
	return lipgloss.NewStyle().MaxWidth(width)
}

func Width(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}

func Height(height int) lipgloss.Style {
	return lipgloss.NewStyle().Height(height)
}
