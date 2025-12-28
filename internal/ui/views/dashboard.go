package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/brew"
	"github.com/lazar/brewst/internal/state"
	"github.com/lazar/brewst/internal/ui/components"
	"github.com/lazar/brewst/internal/ui/styles"
)

// PanelType represents which panel is focused
type PanelType int

const (
	PanelInstalled PanelType = iota
	PanelSearch
	PanelDependencies
)

// DashboardView shows everything at once
type DashboardView struct {
	client brew.Client
	state  *state.State

	// Panels
	installedList list.Model
	searchInput   textinput.Model
	searchResults []brew.Package

	// State
	focusedPanel    PanelType
	selectedPkg     *brew.Package
	packageInfo     *brew.PackageInfo
	loadingInfo     bool
	searching       bool
	installedIndex  int // Manual selection tracking
	searchIndex     int
	depIndex        int // Dependency selection
	installedScroll int // Scroll positions
	searchScroll    int
	depScroll       int // Dependency scroll

	// Debouncing for package info loading
	pendingPackage *brew.Package // Package waiting to be loaded
	debounceID     int            // ID to track if debounce is still valid

	// Progress indicator
	spinner        spinner.Model
	operationInProgress bool
	operationMessage    string

	// Dialog for confirmations
	dialog *components.Dialog
	pendingAction string // Track what action is pending confirmation

	// Logs
	logs       []string // Log messages
	logsScroll int      // Scroll position in logs

	width  int
	height int
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(client brew.Client, state *state.State) *DashboardView {
	// Installed list
	installedDelegate := list.NewDefaultDelegate()
	installedDelegate.ShowDescription = false
	installedDelegate.SetHeight(1)
	installedList := list.New([]list.Item{}, installedDelegate, 0, 0)
	installedList.SetShowStatusBar(false)
	installedList.SetFilteringEnabled(false)
	installedList.SetShowHelp(false)
	installedList.SetShowTitle(false)

	// Search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search packages..."
	searchInput.CharLimit = 100

	// Spinner for progress indication
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Dialog for confirmations
	dialog := components.NewConfirmDialog("Confirm", "")

	return &DashboardView{
		client:        client,
		state:         state,
		installedList: installedList,
		searchInput:   searchInput,
		focusedPanel:  PanelInstalled,
		spinner:       s,
		dialog:        dialog,
	}
}

// SetSize sets the view size
func (v *DashboardView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Init initializes the view
func (v *DashboardView) Init() tea.Cmd {
	// Show loading state
	v.operationInProgress = true
	v.operationMessage = "Loading packages..."
	v.addLog("→ Loading installed packages...")

	v.updateInstalledList()

	// Select first package by default
	v.installedIndex = 0
	packages := v.state.GetFilteredPackages()
	if len(packages) > 0 {
		v.selectedPkg = &packages[0]
		return tea.Batch(
			v.loadPackageInfo(&packages[0]),
			v.spinner.Tick,
		)
	}

	return v.spinner.Tick
}

// Update handles messages
func (v *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle dialog updates first
	if v.dialog.IsVisible() {
		var cmd tea.Cmd
		v.dialog, cmd = v.dialog.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Always return when dialog was visible - don't let keys pass through
		return v, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case components.DialogMsg:
		if msg.Confirmed {
			// Execute the pending action
			switch v.pendingAction {
			case "install":
				if v.selectedPkg != nil {
					return v, v.installPackage(v.selectedPkg)
				}
			case "uninstall":
				if v.selectedPkg != nil {
					return v, v.uninstallPackage(v.selectedPkg)
				}
			case "upgrade":
				if v.selectedPkg != nil {
					return v, v.upgradePackage(v.selectedPkg.Name)
				}
			case "upgradeAll":
				return v, v.upgradeAll()
			case "doctor":
				return v, v.runDoctor()
			case "cleanup":
				return v, v.runCleanup()
			case "autoremove":
				return v, v.runAutoremove()
			}
		}
		v.pendingAction = ""
		return v, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return v, cmd

	case tea.KeyMsg:
		if v.searchInput.Focused() {
			switch msg.String() {
			case "esc":
				v.searchInput.Blur()
				return v, nil
			case "enter":
				query := v.searchInput.Value()
				if query != "" {
					return v, v.performSearch(query)
				}
				return v, nil
			case "tab":
				v.searchInput.Blur()
				v.focusedPanel = PanelDependencies
				return v, nil
			default:
				var cmd tea.Cmd
				v.searchInput, cmd = v.searchInput.Update(msg)
				return v, cmd
			}
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			switch v.focusedPanel {
			case PanelInstalled:
				if v.installedIndex > 0 {
					v.installedIndex--
					if v.installedIndex < v.installedScroll {
						v.installedScroll = v.installedIndex
					}
					v.updateSelectedPackage()
					return v, v.loadSelectedPackageInfo()
				}
			case PanelSearch:
				if v.searchIndex > 0 {
					v.searchIndex--
					if v.searchIndex < v.searchScroll {
						v.searchScroll = v.searchIndex
					}
					v.updateSelectedPackage()
					return v, v.loadSelectedPackageInfo()
				}
			case PanelDependencies:
				if v.depScroll > 0 {
					v.depScroll--
				}
			}
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			switch v.focusedPanel {
			case PanelInstalled:
				packages := v.state.GetFilteredPackages()
				if v.installedIndex < len(packages)-1 {
					v.installedIndex++
					visibleLines := v.getInstalledVisibleLines()
					if v.installedIndex >= v.installedScroll+visibleLines {
						v.installedScroll = v.installedIndex - visibleLines + 1
					}
					v.updateSelectedPackage()
					return v, v.loadSelectedPackageInfo()
				}
			case PanelSearch:
				if v.searchIndex < len(v.searchResults)-1 {
					v.searchIndex++
					visibleLines := v.getSearchVisibleLines()
					if v.searchIndex >= v.searchScroll+visibleLines {
						v.searchScroll = v.searchIndex - visibleLines + 1
					}
					v.updateSelectedPackage()
					return v, v.loadSelectedPackageInfo()
				}
			case PanelDependencies:
				if v.packageInfo != nil {
					maxScroll := len(v.packageInfo.Dependencies) - v.getDependenciesVisibleLines()
					if maxScroll < 0 {
						maxScroll = 0
					}
					if v.depScroll < maxScroll {
						v.depScroll++
					}
				}
			}
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			v.searchInput.Blur()
			switch v.focusedPanel {
			case PanelInstalled:
				v.focusedPanel = PanelSearch
				v.searchInput.Focus()
			case PanelSearch:
				v.focusedPanel = PanelDependencies
			case PanelDependencies:
				v.focusedPanel = PanelInstalled
			}
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if v.focusedPanel == PanelSearch && len(v.searchResults) > 0 && !v.searchInput.Focused() {
				if v.selectedPkg != nil && !v.selectedPkg.Installed {
					v.pendingAction = "install"
					v.searchInput.Blur()
					v.dialog.SetMessage(fmt.Sprintf("Install %s?", v.selectedPkg.Name))
					v.dialog.Show()
					return v, nil
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("u"))):
			if v.focusedPanel == PanelInstalled && v.selectedPkg != nil && v.selectedPkg.Outdated {
				v.pendingAction = "upgrade"
				v.searchInput.Blur()
				v.dialog.SetMessage(fmt.Sprintf("Upgrade %s?", v.selectedPkg.Name))
				v.dialog.Show()
				return v, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("x"))):
			if v.focusedPanel == PanelInstalled && v.selectedPkg != nil {
				v.pendingAction = "uninstall"
				v.searchInput.Blur()
				v.dialog.SetMessage(fmt.Sprintf("Uninstall %s?", v.selectedPkg.Name))
				v.dialog.Show()
				return v, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("U"))):
			if v.focusedPanel == PanelInstalled {
				outdatedCount := v.state.GetOutdatedCount()
				if outdatedCount > 0 {
					v.pendingAction = "upgradeAll"
					v.searchInput.Blur()
					v.dialog.SetMessage(fmt.Sprintf("Upgrade all %d outdated packages?", outdatedCount))
					v.dialog.Show()
				}
				return v, nil
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			return v, v.refresh()

		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			v.pendingAction = "doctor"
			v.searchInput.Blur()
			v.dialog.SetMessage("Run brew doctor to check for problems?")
			v.dialog.Show()
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
			v.pendingAction = "cleanup"
			v.searchInput.Blur()
			v.dialog.SetMessage("Run brew cleanup to remove old versions?")
			v.dialog.Show()
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			v.pendingAction = "autoremove"
			v.searchInput.Blur()
			v.dialog.SetMessage("Run brew autoremove to uninstall unused dependencies?")
			v.dialog.Show()
			return v, nil
		}

	case DebouncedLoadMsg:
		if msg.id == v.debounceID && msg.pkg != nil {
			return v, v.loadPackageInfo(msg.pkg)
		}
		return v, nil

	case PackageInfoLoadedMsg:
		v.packageInfo = msg.Info
		v.loadingInfo = false
		v.depScroll = 0
		return v, nil

	case SearchResultsMsg:
		v.searchResults = msg.Results
		v.searching = false
		v.searchIndex = 0
		v.searchScroll = 0
		v.searchInput.Blur()
		if len(v.searchResults) > 0 {
			v.selectedPkg = &v.searchResults[0]
			return v, v.loadPackageInfo(&v.searchResults[0])
		}
		return v, nil

	case PackagesLoadedMsg:
		v.updateInstalledList()
		v.operationInProgress = false
		v.operationMessage = ""
		v.installedIndex = 0
		v.installedScroll = 0
		packages := v.state.GetFilteredPackages()
		v.addLog(fmt.Sprintf("✓ Loaded %d packages", len(packages)))
		if len(packages) > 0 {
			v.selectedPkg = &packages[0]
			return v, v.loadPackageInfo(&packages[0])
		}
		return v, nil

	case OutdatedLoadedMsg:
		v.updateInstalledList()
		v.operationInProgress = false
		v.operationMessage = ""
		outdatedCount := 0
		for _, pkg := range v.state.GetFilteredPackages() {
			if pkg.Outdated {
				outdatedCount++
			}
		}
		if outdatedCount > 0 {
			v.addLog(fmt.Sprintf("⚠ Found %d outdated packages", outdatedCount))
		} else {
			v.addLog("✓ All packages are up to date")
		}
		return v, nil

	case SuccessMsgView:
		v.operationInProgress = false
		v.operationMessage = ""
		v.addLog("✓ " + msg.Msg)
		v.state.SetSuccess(msg.Msg)
		return v, func() tea.Msg {
			return RefreshPackagesMsg{}
		}

	case ErrorMsgView:
		v.loadingInfo = false
		v.searching = false
		v.operationInProgress = false
		v.operationMessage = ""
		v.addLog("Error: " + msg.Err.Error())
		v.state.SetError(msg.Err)
		return v, nil

	case DoctorOutputMsg:
		v.operationInProgress = false
		v.operationMessage = ""
		for _, line := range msg.Lines {
			if strings.TrimSpace(line) != "" {
				v.addLog(line)
			}
		}
		v.addLog("✓ Doctor completed")
		return v, nil
	}

	// Update focused panel
	var cmd tea.Cmd
	switch v.focusedPanel {
	case PanelInstalled:
		oldIdx := v.installedList.Index()
		v.installedList, cmd = v.installedList.Update(msg)
		if v.installedList.Index() != oldIdx && len(v.installedList.Items()) > 0 {
			if item, ok := v.installedList.Items()[v.installedList.Index()].(PackageListItem); ok {
				v.selectedPkg = &item.pkg
				cmds = append(cmds, v.loadPackageInfo(&item.pkg))
			}
		}
	}

	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *DashboardView) View() string {
	if v.width == 0 || v.height == 0 {
		return "Loading..."
	}

	// Layout: 50% left (installed list), 50% right (search + dependency tree + logs)
	statusHeight := 1
	contentHeight := v.height - statusHeight

	// Account for panel borders and padding in width calculation
	leftWidth := (v.width / 2) - 2
	rightWidth := (v.width / 2) - 2

	// Right side split: 35% search, 35% dependency tree, 30% logs
	searchHeight := int(float64(contentHeight) * 0.35)
	depTreeHeight := int(float64(contentHeight) * 0.35)
	logsHeight := contentHeight - searchHeight - depTreeHeight

	// Render panels
	installedPanel := v.renderInstalledPanel(leftWidth, contentHeight)
	searchPanel := v.renderSearchPanel(rightWidth, searchHeight)
	depTreePanel := v.renderDependencyTreePanel(rightWidth, depTreeHeight)
	logsPanel := v.renderLogsPanel(rightWidth, logsHeight)

	// Combine right side panels vertically
	rightSide := lipgloss.JoinVertical(lipgloss.Left, searchPanel, depTreePanel, logsPanel)

	// Combine left and right horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, installedPanel, rightSide)

	// Status bar
	statusBar := v.renderStatusBar()

	content := lipgloss.JoinVertical(lipgloss.Left, panels, statusBar)

	// Overlay dialog if visible
	if v.dialog.IsVisible() {
		content = v.dialog.Overlay(content, v.width, v.height)
	}

	return content
}

func (v *DashboardView) renderInstalledPanel(width, height int) string {
	panelStyle := styles.PanelStyle
	if v.focusedPanel == PanelInstalled {
		panelStyle = styles.ActivePanelStyle
	}

	title := styles.PanelTitleStyle.Render(fmt.Sprintf("📦 Installed (%d)", v.state.GetInstalledCount()))

	// Render packages as table with Name, Version, Type
	packages := v.state.GetFilteredPackages()
	var lines []string

	maxLines := height - 5 // title + border + header
	if maxLines < 5 {
		maxLines = 5
	}

	// Calculate column widths
	// Account for: border (4), padding (2), prefix (2), status (2) = 10 total
	contentWidth := width - 10
	nameWidth := int(float64(contentWidth) * 0.5)  // 50% for name
	versionWidth := int(float64(contentWidth) * 0.3) // 30% for version
	typeWidth := int(float64(contentWidth) * 0.2) // 20% for type

	// Header row
	header := fmt.Sprintf("  %-*s %-*s %-*s",
		nameWidth, "NAME",
		versionWidth, "VERSION",
		typeWidth, "TYPE")
	lines = append(lines, styles.DimStyle.Render(header))
	lines = append(lines, styles.DimStyle.Render(strings.Repeat("─", width-6)))

	// Calculate visible range based on scroll position
	start := v.installedScroll
	end := start + maxLines
	if end > len(packages) {
		end = len(packages)
	}

	for i := start; i < end; i++ {
		pkg := packages[i]

		prefix := " "
		if i == v.installedIndex && v.focusedPanel == PanelInstalled {
			prefix = "▶"
		}

		// Type text without emoji
		typeDisplay := "Formula"
		typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")) // Green for Formula
		if pkg.Type == brew.TypeCask {
			typeDisplay = "Cask"
			typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")) // Blue for Cask
		}

		// Truncate name if too long
		name := pkg.Name
		if len(name) > nameWidth-2 {
			name = name[:nameWidth-5] + "..."
		}

		// Truncate version if too long
		version := pkg.Version
		if version == "" {
			version = "-"
		}
		if len(version) > versionWidth-2 {
			version = version[:versionWidth-5] + "..."
		}

		// Status indicator
		status := "✓"
		if pkg.Outdated {
			status = "⚠"
		}

		// Apply color to type
		styledType := typeStyle.Render(fmt.Sprintf("%-*s", typeWidth, typeDisplay))

		// Build final line with styled type
		finalLine := fmt.Sprintf("%s %-*s %-*s %s %s",
			prefix,
			nameWidth, name,
			versionWidth, version,
			styledType,
			status)

		lines = append(lines, finalLine)
	}

	listContent := strings.Join(lines, "\n")
	if len(packages) == 0 {
		listContent = styles.DimStyle.Render("No packages installed")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, listContent)

	return panelStyle.
		Width(width).
		Render(content)
}

func (v *DashboardView) renderDependencyTreePanel(width, height int) string {
	panelStyle := styles.PanelStyle
	if v.focusedPanel == PanelDependencies {
		panelStyle = styles.ActivePanelStyle
	}

	title := styles.PanelTitleStyle.Render("🌳 Dependencies")

	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n")

	if v.packageInfo == nil {
		content.WriteString(styles.DimStyle.Render("Select a package to view dependencies"))
	} else if len(v.packageInfo.Dependencies) == 0 {
		content.WriteString(styles.DimStyle.Render("No dependencies"))
	} else {
		content.WriteString(styles.KeyStyle.Render(v.packageInfo.Name))
		content.WriteString("\n")

		maxLines := height - 6
		if maxLines < 1 {
			maxLines = 1
		}

		deps := v.packageInfo.Dependencies

		// Apply scrolling
		start := v.depScroll
		end := start + maxLines
		if end > len(deps) {
			end = len(deps)
		}

		for i := start; i < end; i++ {
			dep := deps[i]
			isLast := i == len(deps)-1

			var prefix string
			if isLast {
				prefix = "└── "
			} else {
				prefix = "├── "
			}

			content.WriteString(styles.ValueStyle.Render(prefix + dep))
			content.WriteString("\n")
		}

		// Show scroll indicator if there are more dependencies
		if end < len(deps) {
			remaining := len(deps) - end
			content.WriteString(styles.DimStyle.Render(fmt.Sprintf("    ↓ %d more (scroll with j/k)", remaining)))
			content.WriteString("\n")
		}
		if start > 0 {
			content.WriteString(styles.DimStyle.Render(fmt.Sprintf("    ↑ %d above", start)))
		}
	}

	return panelStyle.Width(width).Render(content.String())
}

func (v *DashboardView) renderLogsPanel(width, height int) string {
	panelStyle := styles.PanelStyle

	title := styles.PanelTitleStyle.Render("📋 Logs")

	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n")

	if len(v.logs) == 0 {
		content.WriteString(styles.DimStyle.Render("No logs yet"))
	} else {
		maxLines := height - 4
		if maxLines < 1 {
			maxLines = 1
		}

		// Show most recent logs (auto-scroll to bottom)
		start := 0
		if len(v.logs) > maxLines {
			start = len(v.logs) - maxLines
		}

		for i := start; i < len(v.logs); i++ {
			logLine := v.logs[i]
			// Color code based on content
			if strings.Contains(logLine, "Error") || strings.Contains(logLine, "error") {
				content.WriteString(styles.ErrorStyle.Render(logLine))
			} else if strings.Contains(logLine, "Success") || strings.Contains(logLine, "✓") {
				content.WriteString(styles.SuccessMessageStyle.Render(logLine))
			} else if strings.Contains(logLine, "Warning") || strings.Contains(logLine, "⚠") {
				content.WriteString(styles.OutdatedStyle.Render(logLine))
			} else {
				content.WriteString(styles.DimStyle.Render(logLine))
			}
			content.WriteString("\n")
		}
	}

	return panelStyle.Width(width).Render(content.String())
}

func (v *DashboardView) renderSearchPanel(width, height int) string {
	panelStyle := styles.PanelStyle
	if v.focusedPanel == PanelSearch {
		panelStyle = styles.ActivePanelStyle
	}

	title := styles.PanelTitleStyle.Render("🔍 Search")

	v.searchInput.Width = width - 6

	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n")
	content.WriteString(v.searchInput.View())
	content.WriteString("\n")

	if v.searching {
		content.WriteString(styles.DimStyle.Render("Searching..."))
	} else if len(v.searchResults) > 0 {
		content.WriteString(styles.DimStyle.Render(fmt.Sprintf("(%d results)", len(v.searchResults))))
		content.WriteString("\n")

		maxLines := height - 7
		if maxLines < 2 {
			maxLines = 2
		}

		start := v.searchScroll
		end := start + maxLines
		if end > len(v.searchResults) {
			end = len(v.searchResults)
		}

		for i := start; i < end; i++ {
			pkg := v.searchResults[i]

			prefix := "  "
			if i == v.searchIndex && v.focusedPanel == PanelSearch {
				prefix = "▶ "
			}

			pkgLine := prefix + pkg.Name
			if pkg.Installed {
				pkgLine = styles.InstalledStyle.Render(pkgLine + " ✓")
			}
			content.WriteString(pkgLine)
			content.WriteString("\n")
		}

		if end < len(v.searchResults) {
			remaining := len(v.searchResults) - end
			content.WriteString(styles.DimStyle.Render(fmt.Sprintf("  ↓ %d more", remaining)))
		}
	}

	return panelStyle.Width(width).Render(content.String())
}


func (v *DashboardView) renderPackageInfo() string {
	if v.packageInfo == nil {
		return ""
	}

	info := v.packageInfo
	var sections []string

	// Name and version
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left,
		styles.KeyStyle.Render("Name: "),
		styles.ValueStyle.Render(info.Name),
	))

	if info.Version != "" {
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left,
			styles.KeyStyle.Render("Version: "),
			styles.ValueStyle.Render(info.Version),
		))
	}

	// Type
	pkgType := "Formula"
	if info.Type == brew.TypeCask {
		pkgType = "Cask"
	}
	sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left,
		styles.KeyStyle.Render("Type: "),
		styles.ValueStyle.Render(pkgType),
	))

	// Description
	if info.Description != "" {
		sections = append(sections, "")
		sections = append(sections, styles.KeyStyle.Render("Description:"))
		sections = append(sections, styles.ValueStyle.Render(info.Description))
	}

	// Homepage
	if info.Homepage != "" {
		sections = append(sections, "")
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Left,
			styles.KeyStyle.Render("Homepage: "),
			styles.DimStyle.Render(info.Homepage),
		))
	}

	// Dependencies
	if len(info.Dependencies) > 0 {
		sections = append(sections, "")
		sections = append(sections, styles.KeyStyle.Render("Dependencies:"))
		sections = append(sections, styles.ValueStyle.Render("  "+strings.Join(info.Dependencies, ", ")))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (v *DashboardView) renderStatusBar() string {
	// If operation is in progress, show spinner and message
	if v.operationInProgress {
		statusText := fmt.Sprintf("%s %s", v.spinner.View(), v.operationMessage)
		return styles.StatusBarStyle.Width(v.width).Render(statusText)
	}

	var parts []string

	switch v.focusedPanel {
	case PanelInstalled:
		parts = append(parts, "u: Upgrade")
		parts = append(parts, "x: Uninstall")
		parts = append(parts, "U: Upgrade all")
	case PanelSearch:
		parts = append(parts, "Enter: Search/Install")
	}

	parts = append(parts, "Tab: Switch")
	parts = append(parts, "d: Doctor")
	parts = append(parts, "c: Cleanup")
	parts = append(parts, "a: Autoremove")
	parts = append(parts, "r: Refresh")
	parts = append(parts, "q: Quit")

	return styles.StatusBarStyle.Width(v.width).Render(strings.Join(parts, " • "))
}

func (v *DashboardView) updateInstalledList() {
	packages := v.state.GetFilteredPackages()
	items := make([]list.Item, len(packages))
	for i, pkg := range packages {
		items[i] = PackageListItem{pkg: pkg}
	}
	v.installedList.SetItems(items)
}

func (v *DashboardView) loadPackageInfo(pkg *brew.Package) tea.Cmd {
	v.loadingInfo = true
	return func() tea.Msg {
		ctx := context.Background()
		info, err := v.client.Info(ctx, pkg.Name, pkg.Type == brew.TypeCask)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return PackageInfoLoadedMsg{Info: info}
	}
}

func (v *DashboardView) updateSelectedPackage() {
	switch v.focusedPanel {
	case PanelInstalled:
		packages := v.state.GetFilteredPackages()
		if v.installedIndex >= 0 && v.installedIndex < len(packages) {
			v.selectedPkg = &packages[v.installedIndex]
		}
	case PanelSearch:
		if v.searchIndex >= 0 && v.searchIndex < len(v.searchResults) {
			v.selectedPkg = &v.searchResults[v.searchIndex]
		}
	}
}

func (v *DashboardView) loadSelectedPackageInfo() tea.Cmd {
	if v.selectedPkg != nil {
		return v.debouncedLoadPackageInfo(v.selectedPkg)
	}
	return nil
}

func (v *DashboardView) debouncedLoadPackageInfo(pkg *brew.Package) tea.Cmd {
	// Increment debounce ID to invalidate previous debounces
	v.debounceID++
	v.pendingPackage = pkg
	currentID := v.debounceID

	// Return a command that waits 200ms then sends a message
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return DebouncedLoadMsg{pkg: pkg, id: currentID}
	})
}

func (v *DashboardView) performSearch(query string) tea.Cmd {
	v.searching = true
	return func() tea.Msg {
		ctx := context.Background()
		results, err := v.client.Search(ctx, query)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SearchResultsMsg{Results: results}
	}
}

func (v *DashboardView) addLog(msg string) {
	v.logs = append(v.logs, msg)
	// Keep only last 1000 lines
	if len(v.logs) > 1000 {
		v.logs = v.logs[len(v.logs)-1000:]
	}
}

func (v *DashboardView) installPackage(pkg *brew.Package) tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = fmt.Sprintf("Installing %s...", pkg.Name)
	v.addLog(fmt.Sprintf("→ Installing %s...", pkg.Name))
	return func() tea.Msg {
		ctx := context.Background()
		opts := brew.InstallOptions{Cask: pkg.Type == brew.TypeCask}
		err := v.client.Install(ctx, pkg.Name, opts)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Installed " + pkg.Name}
	}
}

func (v *DashboardView) uninstallPackage(pkg *brew.Package) tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = fmt.Sprintf("Uninstalling %s...", pkg.Name)
	v.addLog(fmt.Sprintf("→ Uninstalling %s...", pkg.Name))
	return func() tea.Msg {
		ctx := context.Background()
		opts := brew.UninstallOptions{Cask: pkg.Type == brew.TypeCask}
		err := v.client.Uninstall(ctx, pkg.Name, opts)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Uninstalled " + pkg.Name}
	}
}

func (v *DashboardView) upgradePackage(name string) tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = fmt.Sprintf("Upgrading %s...", name)
	v.addLog(fmt.Sprintf("→ Upgrading %s...", name))
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.Upgrade(ctx, []string{name})
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Upgraded " + name}
	}
}

func (v *DashboardView) upgradeAll() tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = "Upgrading all packages..."
	v.addLog("→ Upgrading all packages...")
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.Upgrade(ctx, []string{})
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Upgraded all packages"}
	}
}

func (v *DashboardView) refresh() tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = "Refreshing packages..."
	return func() tea.Msg {
		// Trigger refresh of installed and outdated
		return RefreshPackagesMsg{}
	}
}

func (v *DashboardView) runDoctor() tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = "Running brew doctor..."
	v.addLog("→ Running brew doctor...")
	return func() tea.Msg {
		ctx := context.Background()
		output, err := v.client.Doctor(ctx)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		// Add output to logs (split by lines)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		return DoctorOutputMsg{Lines: lines}
	}
}

func (v *DashboardView) runCleanup() tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = "Running brew cleanup..."
	v.addLog("→ Running brew cleanup...")
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.Cleanup(ctx)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Cleanup completed"}
	}
}

func (v *DashboardView) runAutoremove() tea.Cmd {
	v.operationInProgress = true
	v.operationMessage = "Running brew autoremove..."
	v.addLog("→ Running brew autoremove...")
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.Autoremove(ctx)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Autoremove completed"}
	}
}

func (v *DashboardView) getInstalledVisibleLines() int {
	contentHeight := v.height - 1
	installedHeight := contentHeight
	return installedHeight - 5
}

func (v *DashboardView) getSearchVisibleLines() int {
	contentHeight := v.height - 1
	searchHeight := int(float64(contentHeight) * 0.35)
	maxLines := searchHeight - 7
	if maxLines < 2 {
		maxLines = 2
	}
	return maxLines
}

func (v *DashboardView) getDependenciesVisibleLines() int {
	contentHeight := v.height - 1
	depTreeHeight := int(float64(contentHeight) * 0.35)
	maxLines := depTreeHeight - 6
	if maxLines < 1 {
		maxLines = 1
	}
	return maxLines
}

// PackageListItem for installed list
type PackageListItem struct {
	pkg brew.Package
}

func (i PackageListItem) FilterValue() string { return i.pkg.Name }
func (i PackageListItem) Title() string {
	name := i.pkg.Name
	if i.pkg.Outdated {
		return styles.OutdatedStyle.Render(name + " ⚠")
	}
	return styles.InstalledStyle.Render(name + " ✓")
}
func (i PackageListItem) Description() string { return "" }

// Message types
type PackagesLoadedMsg struct{ Packages []brew.Package }
type OutdatedLoadedMsg struct{ Packages []brew.OutdatedPackage }
type DebouncedLoadMsg struct {
	pkg *brew.Package
	id  int
}
type DoctorOutputMsg struct{ Lines []string }
