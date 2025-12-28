package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/ui/styles"
)

// Header represents the application header
type Header struct {
	title    string
	subtitle string
	width    int
}

// NewHeader creates a new header component
func NewHeader() *Header {
	return &Header{
		title:    "Brewst",
		subtitle: "Homebrew TUI",
	}
}

// SetWidth sets the header width
func (h *Header) SetWidth(width int) {
	h.width = width
}

// SetTitle sets the header title
func (h *Header) SetTitle(title string) {
	h.title = title
}

// SetSubtitle sets the header subtitle
func (h *Header) SetSubtitle(subtitle string) {
	h.subtitle = subtitle
}

// View renders the header
func (h *Header) View() string {
	title := styles.TitleStyle.Render(h.title)
	subtitle := styles.SubtitleStyle.Render(h.subtitle)

	header := lipgloss.JoinVertical(lipgloss.Left, title, subtitle)

	if h.width > 0 {
		header = styles.HeaderStyle.Width(h.width - 2).Render(header)
	} else {
		header = styles.HeaderStyle.Render(header)
	}

	return header
}

// Height returns the header height
func (h *Header) Height() int {
	return lipgloss.Height(h.View())
}

// ViewWithBreadcrumb renders the header with a breadcrumb
func (h *Header) ViewWithBreadcrumb(breadcrumb string) string {
	title := fmt.Sprintf("%s > %s", h.title, breadcrumb)
	titleStyled := styles.TitleStyle.Render(title)
	subtitle := styles.SubtitleStyle.Render(h.subtitle)

	header := lipgloss.JoinVertical(lipgloss.Left, titleStyled, subtitle)

	if h.width > 0 {
		header = styles.HeaderStyle.Width(h.width - 2).Render(header)
	} else {
		header = styles.HeaderStyle.Render(header)
	}

	return header
}
