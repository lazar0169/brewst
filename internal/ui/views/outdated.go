package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar0169/brewst/internal/brew"
	"github.com/lazar0169/brewst/internal/state"
	"github.com/lazar0169/brewst/internal/ui/styles"
)

// OutdatedItem represents an outdated package in the list
type OutdatedItem struct {
	pkg brew.OutdatedPackage
}

func (i OutdatedItem) FilterValue() string { return i.pkg.Name }
func (i OutdatedItem) Title() string {
	name := i.pkg.Name
	if i.pkg.Pinned {
		name = styles.PinnedStyle.Render(name + " 📌")
	} else {
		name = styles.OutdatedStyle.Render(name + " ⚠")
	}
	return name
}
func (i OutdatedItem) Description() string {
	return fmt.Sprintf("%s → %s", i.pkg.CurrentVersion, i.pkg.LatestVersion)
}

// OutdatedView shows outdated packages
type OutdatedView struct {
	client brew.Client
	state  *state.State

	list   list.Model
	width  int
	height int
}

// NewOutdatedView creates a new outdated packages view
func NewOutdatedView(client brew.Client, state *state.State) *OutdatedView {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "Outdated Packages"
	l.Styles.Title = styles.TitleStyle

	return &OutdatedView{
		client: client,
		state:  state,
		list:   l,
	}
}

// SetSize sets the view size
func (v *OutdatedView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width-4, height-4)
}

// Init initializes the view
func (v *OutdatedView) Init() tea.Cmd {
	items := make([]list.Item, len(v.state.OutdatedPackages))
	for i, pkg := range v.state.OutdatedPackages {
		items[i] = OutdatedItem{pkg: pkg}
	}
	v.list.SetItems(items)
	return nil
}

// Update handles messages
func (v *OutdatedView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("u"))):
			// Upgrade selected package
			if item, ok := v.list.SelectedItem().(OutdatedItem); ok {
				return v, v.upgradePackage(item.pkg.Name)
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("U"))):
			// Upgrade all packages
			return v, v.upgradeAll()

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			// Refresh outdated list
			return v, func() tea.Msg {
				return RefreshOutdatedMsg{}
			}
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
func (v *OutdatedView) View() string {
	if len(v.state.OutdatedPackages) == 0 {
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TitleStyle.Render("Outdated Packages"),
			"",
			styles.SuccessMessageStyle.Render("All packages are up to date!"),
		)
		return styles.AppStyle.Render(content)
	}

	helpText := "u: Upgrade selected | U: Upgrade all | r: Refresh | Esc: Back"
	help := styles.HelpStyle.Render(helpText)

	return v.list.View() + "\n" + help
}

func (v *OutdatedView) upgradePackage(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.Upgrade(ctx, []string{name})
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Successfully upgraded " + name}
	}
}

func (v *OutdatedView) upgradeAll() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.Upgrade(ctx, []string{})
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Successfully upgraded all packages"}
	}
}

// Message types
type RefreshOutdatedMsg struct{}
