package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/ui/styles"
)

// DialogType represents the type of dialog
type DialogType int

const (
	DialogConfirm DialogType = iota
	DialogInfo
	DialogError
)

// Dialog represents a dialog box
type Dialog struct {
	title       string
	message     string
	dialogType  DialogType
	selectedBtn int
	buttons     []string
	visible     bool
	onConfirm   func()
	onCancel    func()
}

// DialogMsg is sent when a dialog action is confirmed
type DialogMsg struct {
	Confirmed bool
	Action    string
}

// NewDialog creates a new dialog
func NewDialog(title, message string, dialogType DialogType) *Dialog {
	buttons := []string{"Confirm", "Cancel"}
	selectedBtn := 0 // Default to Confirm
	if dialogType == DialogInfo || dialogType == DialogError {
		buttons = []string{"OK"}
	}

	return &Dialog{
		title:       title,
		message:     message,
		dialogType:  dialogType,
		selectedBtn: selectedBtn,
		buttons:     buttons,
		visible:     false,
	}
}

// NewConfirmDialog creates a confirmation dialog
func NewConfirmDialog(title, message string) *Dialog {
	return NewDialog(title, message, DialogConfirm)
}

// Show shows the dialog
func (d *Dialog) Show() {
	d.visible = true
	d.selectedBtn = 0
}

// Hide hides the dialog
func (d *Dialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible
func (d *Dialog) IsVisible() bool {
	return d.visible
}

// SetMessage sets the dialog message
func (d *Dialog) SetMessage(message string) {
	d.message = message
}

// Update handles dialog input
func (d *Dialog) Update(msg tea.Msg) (*Dialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			if d.selectedBtn > 0 {
				d.selectedBtn--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			if d.selectedBtn < len(d.buttons)-1 {
				d.selectedBtn++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			d.selectedBtn = (d.selectedBtn + 1) % len(d.buttons)
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			d.visible = false
			if d.dialogType == DialogConfirm {
				confirmed := d.selectedBtn == 0
				return d, func() tea.Msg {
					return DialogMsg{Confirmed: confirmed}
				}
			}
			return d, func() tea.Msg {
				return DialogMsg{Confirmed: true}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			d.visible = false
			return d, func() tea.Msg {
				return DialogMsg{Confirmed: false}
			}
		}
	}

	return d, nil
}

// View renders the dialog
func (d *Dialog) View() string {
	if !d.visible {
		return ""
	}

	// Title
	title := styles.DialogTitleStyle.Render(d.title)

	// Message
	message := styles.ValueStyle.Render(d.message)

	// Buttons
	var buttons []string
	for i, btnText := range d.buttons {
		if i == d.selectedBtn {
			buttons = append(buttons, styles.DialogButtonActiveStyle.Render(btnText))
		} else {
			buttons = append(buttons, styles.DialogButtonStyle.Render(btnText))
		}
	}
	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Left, buttons...)

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		message,
		"",
		buttonsRow,
	)

	// Apply dialog box style
	dialog := styles.DialogBoxStyle.Render(content)

	// Center the dialog
	return lipgloss.Place(
		80, 10,
		lipgloss.Center, lipgloss.Center,
		dialog,
	)
}

// Overlay renders the dialog as an overlay on top of content
func (d *Dialog) Overlay(content string, width, height int) string {
	if !d.visible {
		return content
	}

	dialog := d.View()

	// Place dialog in center of screen
	overlay := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Muted),
	)

	return overlay
}
