package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar0169/brewst/internal/brew"
	"github.com/lazar0169/brewst/internal/state"
	"github.com/lazar0169/brewst/internal/ui/styles"
)

// HomeView is the main dashboard view
type HomeView struct {
	client brew.Client
	state  *state.State

	selectedIndex int
	menuItems     []menuItem
	width         int
	height        int
}

type menuItem struct {
	title       string
	description string
	key         string
	action      int
}

// NewHomeView creates a new home view
func NewHomeView(client brew.Client, state *state.State) *HomeView {
	items := []menuItem{
		{title: "Installed Packages", description: "View and manage installed formulae and casks", key: "1", action: 1},
		{title: "Search", description: "Search for packages to install", key: "2", action: 2},
		{title: "Outdated Packages", description: "View and upgrade outdated packages", key: "3", action: 3},
		{title: "Taps", description: "Manage Homebrew taps", key: "4", action: 4},
		{title: "Diagnostics", description: "Run brew doctor", key: "5", action: 5},
	}

	return &HomeView{
		client:        client,
		state:         state,
		selectedIndex: 0,
		menuItems:     items,
	}
}

// SetSize sets the view size
func (v *HomeView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Init initializes the view
func (v *HomeView) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (v *HomeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if v.selectedIndex > 0 {
				v.selectedIndex--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if v.selectedIndex < len(v.menuItems)-1 {
				v.selectedIndex++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			action := v.menuItems[v.selectedIndex].action
			return v, func() tea.Msg {
				return NavigateMsg(action)
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("1", "2", "3", "4", "5"))):
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(v.menuItems) {
				action := v.menuItems[idx].action
				return v, func() tea.Msg {
					return NavigateMsg(action)
				}
			}
		}
	}

	return v, nil
}

// View renders the view
func (v *HomeView) View() string {
	// Stats section
	statsSection := v.renderStats()

	// Menu section
	menuSection := v.renderMenu()

	// Help section
	helpSection := styles.HelpStyle.Render("Use ↑/↓ or j/k to navigate, Enter to select, q to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		statsSection,
		"",
		menuSection,
		"",
		helpSection,
	)

	return styles.AppStyle.Render(content)
}

func (v *HomeView) renderStats() string {
	installedCount := v.state.GetInstalledCount()
	outdatedCount := v.state.GetOutdatedCount()
	favCount := len(v.state.Favorites)

	statsTitle := styles.TitleStyle.Render("Statistics")

	installedStat := fmt.Sprintf("%s %d", styles.KeyStyle.Render("Installed:"), installedCount)
	outdatedStat := fmt.Sprintf("%s %d", styles.KeyStyle.Render("Outdated:"), outdatedCount)
	favStat := fmt.Sprintf("%s %d", styles.KeyStyle.Render("Favorites:"), favCount)

	stats := lipgloss.JoinVertical(
		lipgloss.Left,
		installedStat,
		outdatedStat,
		favStat,
	)

	return lipgloss.JoinVertical(lipgloss.Left, statsTitle, "", stats)
}

func (v *HomeView) renderMenu() string {
	menuTitle := styles.TitleStyle.Render("Navigation")

	var menuItems []string
	for i, item := range v.menuItems {
		var itemStr string
		if i == v.selectedIndex {
			// Selected item
			title := fmt.Sprintf("[%s] %s", item.key, item.title)
			itemStr = styles.SelectedStyle.Render("▶ " + title)
			desc := styles.DimStyle.Render("  " + item.description)
			itemStr = lipgloss.JoinVertical(lipgloss.Left, itemStr, desc)
		} else {
			// Unselected item
			title := fmt.Sprintf("[%s] %s", item.key, item.title)
			itemStr = styles.UnselectedStyle.Render("  " + title)
		}
		menuItems = append(menuItems, itemStr)
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, menuItems...)

	return lipgloss.JoinVertical(lipgloss.Left, menuTitle, "", menu)
}

// NavigateMsg is sent when navigation is requested
type NavigateMsg int
