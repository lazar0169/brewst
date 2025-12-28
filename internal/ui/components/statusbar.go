package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/ui/styles"
)

// StatusBar represents the application status bar
type StatusBar struct {
	leftText  string
	rightText string
	width     int
}

// NewStatusBar creates a new status bar component
func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

// SetWidth sets the status bar width
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// SetLeftText sets the left side text
func (s *StatusBar) SetLeftText(text string) {
	s.leftText = text
}

// SetRightText sets the right side text
func (s *StatusBar) SetRightText(text string) {
	s.rightText = text
}

// View renders the status bar
func (s *StatusBar) View() string {
	if s.width == 0 {
		return styles.StatusBarStyle.Render(fmt.Sprintf("%s | %s", s.leftText, s.rightText))
	}

	leftPart := styles.StatusBarStyle.Render(s.leftText)
	rightPart := styles.StatusBarStyle.Render(s.rightText)

	// Calculate spacing
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)
	totalPadding := 4 // Account for padding in StatusBarStyle
	spacingWidth := s.width - leftWidth - rightWidth - totalPadding

	if spacingWidth < 0 {
		spacingWidth = 0
	}

	spacing := styles.StatusBarStyle.Render(lipgloss.NewStyle().Width(spacingWidth).Render(""))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPart, spacing, rightPart)
}

// Height returns the status bar height
func (s *StatusBar) Height() int {
	return 1
}

// ViewWithHelp renders the status bar with help text
func (s *StatusBar) ViewWithHelp(helpText string) string {
	if s.width == 0 {
		help := styles.HelpStyle.Render(helpText)
		status := styles.StatusBarStyle.Render(fmt.Sprintf("%s | %s", s.leftText, s.rightText))
		return lipgloss.JoinVertical(lipgloss.Left, help, status)
	}

	help := styles.HelpStyle.Width(s.width - 2).Render(helpText)

	leftPart := styles.StatusBarStyle.Render(s.leftText)
	rightPart := styles.StatusBarStyle.Render(s.rightText)

	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightPart)
	totalPadding := 4
	spacingWidth := s.width - leftWidth - rightWidth - totalPadding

	if spacingWidth < 0 {
		spacingWidth = 0
	}

	spacing := styles.StatusBarStyle.Render(lipgloss.NewStyle().Width(spacingWidth).Render(""))
	statusLine := lipgloss.JoinHorizontal(lipgloss.Top, leftPart, spacing, rightPart)

	return lipgloss.JoinVertical(lipgloss.Left, help, statusLine)
}
