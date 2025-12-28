package app

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar0169/brewst/internal/brew"
	"github.com/lazar0169/brewst/internal/state"
	"github.com/lazar0169/brewst/internal/ui/components"
	"github.com/lazar0169/brewst/internal/ui/styles"
	"github.com/lazar0169/brewst/internal/ui/views"
)

// ViewType represents the different views in the application
type ViewType int

const (
	ViewHome ViewType = iota
	ViewInstalled
	ViewSearch
	ViewDetails
	ViewOutdated
	ViewTaps
	ViewDiagnostics
)

// Model is the main application model
type Model struct {
	// Dependencies
	brewClient brew.Client
	state      *state.State
	config     *state.Config

	// UI components
	header    *components.Header
	statusBar *components.StatusBar
	dialog    *components.Dialog
	spinner   spinner.Model

	// Views
	currentView ViewType
	viewStack   []ViewType
	views       map[ViewType]tea.Model

	// Window size
	width  int
	height int

	// Application state
	ready bool
	err   error
}

// Msg types for navigation
type (
	NavigateMsg     ViewType
	BackMsg         struct{}
	ErrorMsg        struct{ Err error }
	SuccessMsg      struct{ Msg string }
	PackagesLoadedMsg struct{ Packages []brew.Package }
	OutdatedLoadedMsg struct{ Packages []brew.OutdatedPackage }
	TapsLoadedMsg struct{ Taps []brew.Tap }
)

// New creates a new application model
func New() *Model {
	config, _ := state.LoadConfig()
	favorites, _ := state.LoadFavorites()

	appState := state.NewState()
	appState.Favorites = favorites
	appState.ShowFormulae = config.ShowFormulaByDefault
	appState.ShowCasks = config.ShowCasksByDefault

	brewClient := brew.NewClient()

	// Initialize views
	viewsMap := make(map[ViewType]tea.Model)
	viewsMap[ViewHome] = views.NewDashboardView(brewClient, appState) // Use dashboard as home
	viewsMap[ViewInstalled] = views.NewInstalledView(brewClient, appState)
	viewsMap[ViewSearch] = views.NewSearchView(brewClient, appState)
	viewsMap[ViewDetails] = views.NewDetailsView(brewClient, appState)
	viewsMap[ViewOutdated] = views.NewOutdatedView(brewClient, appState)
	viewsMap[ViewTaps] = views.NewTapsView(brewClient, appState)
	viewsMap[ViewDiagnostics] = views.NewDiagnosticsView(brewClient, appState)

	// Initialize spinner for loading screen
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &Model{
		brewClient:  brewClient,
		state:       appState,
		config:      config,
		header:      components.NewHeader(),
		statusBar:   components.NewStatusBar(),
		dialog:      components.NewConfirmDialog("", ""),
		spinner:     s,
		currentView: ViewHome,
		viewStack:   []ViewType{},
		views:       viewsMap,
		ready:       false,
	}
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Load packages
	cmds = append(cmds,
		loadInstalledPackages(m.brewClient),
		loadOutdatedPackages(m.brewClient),
		m.spinner.Tick,
	)

	// Initialize the home view (dashboard)
	if view, ok := m.views[ViewHome]; ok {
		if v, ok := view.(interface{ Init() tea.Cmd }); ok {
			if cmd := v.Init(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return tea.Batch(cmds...)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle spinner ticks while loading
	if !m.ready {
		if _, ok := msg.(spinner.TickMsg); ok {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Handle dialog updates first
	if m.dialog.IsVisible() {
		var cmd tea.Cmd
		m.dialog, cmd = m.dialog.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// If dialog is still visible, don't process other updates
		if m.dialog.IsVisible() {
			return m, tea.Batch(cmds...)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update all views with full terminal size
		for _, view := range m.views {
			if v, ok := view.(interface{ SetSize(int, int) }); ok {
				v.SetSize(m.width, m.height)
			}
		}

		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Global key bindings
		switch msg.String() {
		case "ctrl+c", "q":
			// Save favorites before quitting
			_ = state.SaveFavorites(m.state.Favorites)
			return m, tea.Quit

		case "esc":
			if len(m.viewStack) > 0 {
				return m, func() tea.Msg { return BackMsg{} }
			}

		case "?":
			return m, nil

		case "1":
			return m, func() tea.Msg { return NavigateMsg(ViewHome) }
		case "2":
			return m, func() tea.Msg { return NavigateMsg(ViewInstalled) }
		case "3":
			return m, func() tea.Msg { return NavigateMsg(ViewSearch) }
		case "4":
			return m, func() tea.Msg { return NavigateMsg(ViewOutdated) }
		case "5":
			return m, func() tea.Msg { return NavigateMsg(ViewTaps) }
		case "6":
			return m, func() tea.Msg { return NavigateMsg(ViewDiagnostics) }
		}

	case NavigateMsg:
		m.viewStack = append(m.viewStack, m.currentView)
		m.currentView = ViewType(msg)
		m.state.ClearMessages()
		if view, ok := m.views[m.currentView]; ok {
			if v, ok := view.(interface{ Init() tea.Cmd }); ok {
				return m, v.Init()
			}
		}
		return m, nil

	case BackMsg:
		if len(m.viewStack) > 0 {
			m.currentView = m.viewStack[len(m.viewStack)-1]
			m.viewStack = m.viewStack[:len(m.viewStack)-1]
			m.state.ClearMessages()
		}
		return m, nil

	case ErrorMsg:
		m.state.SetError(msg.Err)
		return m, nil

	case SuccessMsg:
		m.state.SetSuccess(msg.Msg)
		return m, nil

	case PackagesLoadedMsg:
		m.state.SetInstalled(msg.Packages)
		if view, ok := m.views[m.currentView]; ok {
			viewMsg := views.PackagesLoadedMsg{Packages: msg.Packages}
			updatedView, cmd := view.Update(viewMsg)
			m.views[m.currentView] = updatedView
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case OutdatedLoadedMsg:
		m.state.SetOutdated(msg.Packages)
		if view, ok := m.views[m.currentView]; ok {
			viewMsg := views.OutdatedLoadedMsg{Packages: msg.Packages}
			updatedView, cmd := view.Update(viewMsg)
			m.views[m.currentView] = updatedView
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case TapsLoadedMsg:
		m.state.Taps = msg.Taps
		return m, nil
	}

	switch msg.(type) {
	case views.RefreshPackagesMsg:
		return m, tea.Batch(
			loadInstalledPackages(m.brewClient),
			loadOutdatedPackages(m.brewClient),
		)
	}

	// Update current view
	if view, ok := m.views[m.currentView]; ok {
		updatedView, cmd := view.Update(msg)
		m.views[m.currentView] = updatedView
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		switch updatedView.(type) {
		case *views.HomeView:
			if navMsg, ok := msg.(views.NavigateMsg); ok {
				viewMap := map[int]ViewType{
					1: ViewInstalled,
					2: ViewSearch,
					3: ViewOutdated,
					4: ViewTaps,
					5: ViewDiagnostics,
				}
				if viewType, ok := viewMap[int(navMsg)]; ok {
					return m, func() tea.Msg { return NavigateMsg(viewType) }
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the application
func (m Model) View() string {
	if !m.ready {
		loadingStyle := lipgloss.NewStyle().
			Padding(2, 4).
			Foreground(styles.Primary).
			Bold(true)

		loadingText := lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			fmt.Sprintf("%s Loading Brewst...", m.spinner.View()),
			"",
			styles.DimStyle.Render("Loading installed packages..."),
			styles.DimStyle.Render("This may take a few seconds..."),
		)

		if m.height > 0 {
			verticalPadding := (m.height - 6) / 2
			if verticalPadding > 0 {
				loadingText = lipgloss.NewStyle().PaddingTop(verticalPadding).Render(loadingText)
			}
		}

		return loadingStyle.Render(loadingText)
	}

	var content string
	if view, ok := m.views[m.currentView]; ok {
		content = view.View()
	} else {
		content = "View not implemented yet"
	}

	if m.dialog.IsVisible() {
		content = m.dialog.Overlay(content, m.width, m.height)
	}

	return content
}

// Helper functions

func (m *Model) getViewName(view ViewType) string {
	switch view {
	case ViewHome:
		return "Home"
	case ViewInstalled:
		return "Installed Packages"
	case ViewSearch:
		return "Search"
	case ViewDetails:
		return "Package Details"
	case ViewOutdated:
		return "Outdated Packages"
	case ViewTaps:
		return "Taps"
	case ViewDiagnostics:
		return "Diagnostics"
	default:
		return "Unknown"
	}
}

func (m *Model) getStatusText() string {
	if m.state.ErrorMsg != "" {
		return "Error: " + m.state.ErrorMsg
	}
	if m.state.SuccessMsg != "" {
		return m.state.SuccessMsg
	}
	if m.state.Loading {
		return m.state.LoadingMsg
	}

	installed := m.state.GetInstalledCount()
	outdated := m.state.GetOutdatedCount()

	if outdated > 0 {
		return fmt.Sprintf("Installed: %d | Outdated: %d | Press ? for help", installed, outdated)
	}
	return fmt.Sprintf("Installed: %d | Press ? for help", installed)
}

func loadInstalledPackages(client brew.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		packages, err := client.ListInstalled(ctx, true, true)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return PackagesLoadedMsg{Packages: packages}
	}
}

func loadOutdatedPackages(client brew.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		outdated, err := client.Outdated(ctx)
		if err != nil {
			// Don't return error, just empty list
			return OutdatedLoadedMsg{Packages: []brew.OutdatedPackage{}}
		}
		return OutdatedLoadedMsg{Packages: outdated}
	}
}

func loadTaps(client brew.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		taps, err := client.ListTaps(ctx)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TapsLoadedMsg{Taps: taps}
	}
}
