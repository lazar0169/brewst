package views

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/brew"
	"github.com/lazar/brewst/internal/state"
	"github.com/lazar/brewst/internal/ui/styles"
)

// DiagnosticsView shows brew doctor output
type DiagnosticsView struct {
	client brew.Client
	state  *state.State

	viewport viewport.Model
	output   string
	loading  bool
	width    int
	height   int
}

// NewDiagnosticsView creates a new diagnostics view
func NewDiagnosticsView(client brew.Client, state *state.State) *DiagnosticsView {
	vp := viewport.New(80, 20)
	return &DiagnosticsView{
		client:   client,
		state:    state,
		viewport: vp,
		loading:  false,
	}
}

// SetSize sets the view size
func (v *DiagnosticsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width - 4
	v.viewport.Height = height - 8
}

// Init initializes the view
func (v *DiagnosticsView) Init() tea.Cmd {
	return v.runDiagnostics()
}

// Update handles messages
func (v *DiagnosticsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			// Refresh diagnostics
			return v, v.runDiagnostics()
		}

	case DiagnosticsLoadedMsg:
		v.output = msg.Output
		v.loading = false
		v.viewport.SetContent(v.output)
		return v, nil

	case ErrorMsgView:
		v.loading = false
		v.output = "Error running diagnostics: " + msg.Err.Error()
		v.viewport.SetContent(v.output)
		return v, nil
	}

	// Update viewport
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *DiagnosticsView) View() string {
	title := styles.TitleStyle.Render("Homebrew Diagnostics (brew doctor)")

	if v.loading {
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			styles.DimStyle.Render("Running diagnostics..."),
		)
		return styles.AppStyle.Render(content)
	}

	helpText := "r: Refresh | ↑/↓: Scroll | Esc: Back"
	help := styles.HelpStyle.Render(helpText)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		v.viewport.View(),
		"",
		help,
	)

	return styles.AppStyle.Render(content)
}

func (v *DiagnosticsView) runDiagnostics() tea.Cmd {
	v.loading = true
	return func() tea.Msg {
		ctx := context.Background()
		output, err := v.client.Doctor(ctx)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return DiagnosticsLoadedMsg{Output: output}
	}
}

// Message types
type DiagnosticsLoadedMsg struct{ Output string }
