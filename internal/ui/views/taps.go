package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar0169/brewst/internal/brew"
	"github.com/lazar0169/brewst/internal/state"
	"github.com/lazar0169/brewst/internal/ui/styles"
)

// TapItem represents a tap in the list
type TapItem struct {
	tap brew.Tap
}

func (i TapItem) FilterValue() string { return i.tap.Name }
func (i TapItem) Title() string {
	name := i.tap.Name
	if i.tap.Official {
		return styles.InstalledStyle.Render(name + " ✓")
	}
	return name
}
func (i TapItem) Description() string {
	if i.tap.Official {
		return "Official Homebrew tap"
	}
	return "Third-party tap"
}

// TapsView shows and manages Homebrew taps
type TapsView struct {
	client brew.Client
	state  *state.State

	list   list.Model
	width  int
	height int
}

// NewTapsView creates a new taps view
func NewTapsView(client brew.Client, state *state.State) *TapsView {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "Homebrew Taps"
	l.Styles.Title = styles.TitleStyle

	return &TapsView{
		client: client,
		state:  state,
		list:   l,
	}
}

// SetSize sets the view size
func (v *TapsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width-4, height-4)
}

// Init initializes the view
func (v *TapsView) Init() tea.Cmd {
	items := make([]list.Item, len(v.state.Taps))
	for i, tap := range v.state.Taps {
		items[i] = TapItem{tap: tap}
	}
	v.list.SetItems(items)
	return nil
}

// Update handles messages
func (v *TapsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Refresh taps list
			return v, loadTaps(v.client)
		}
	}

	// Update list
	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *TapsView) View() string {
	if len(v.state.Taps) == 0 {
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Homebrew Taps"),
			"",
			styles.DimStyle.Render("No taps found"),
		)
		return styles.AppStyle.Render(content)
	}

	helpText := fmt.Sprintf("Total taps: %d | r: Refresh | Esc: Back", len(v.state.Taps))
	help := styles.HelpStyle.Render(helpText)

	return v.list.View() + "\n" + help
}

func loadTaps(client brew.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		taps, err := client.ListTaps(ctx)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return TapsLoadedMsg{Taps: taps}
	}
}

type TapsLoadedMsg struct{ Taps []brew.Tap }
