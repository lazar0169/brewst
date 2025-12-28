package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/brew"
	"github.com/lazar/brewst/internal/state"
	"github.com/lazar/brewst/internal/ui/components"
	"github.com/lazar/brewst/internal/ui/styles"
)

// InstalledView shows installed packages with panel layout
type InstalledView struct {
	client brew.Client
	state  *state.State

	list          *components.PackageList
	selectedPkg   *brew.Package
	packageInfo   *brew.PackageInfo
	loadingInfo   bool
	width         int
	height        int
	focusOnDetail bool
}

// NewInstalledView creates a new installed packages view
func NewInstalledView(client brew.Client, state *state.State) *InstalledView {
	return &InstalledView{
		client:        client,
		state:         state,
		list:          components.NewPackageList(80, 20),
		focusOnDetail: false,
	}
}

// SetSize sets the view size
func (v *InstalledView) SetSize(width, height int) {
	v.width = width
	v.height = height
	// List takes left side (40% of width)
	listWidth := int(float64(width) * 0.4)
	v.list.SetSize(listWidth-4, height-4)
}

// Init initializes the view
func (v *InstalledView) Init() tea.Cmd {
	v.list.SetPackages(v.state.GetFilteredPackages())
	v.list.SetTitle("Packages")

	// Load info for first package
	pkg := v.list.GetCurrentPackage()
	if pkg != nil {
		v.selectedPkg = pkg
		return v.loadPackageInfo(pkg)
	}
	return nil
}

// Update handles messages
func (v *InstalledView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			// Toggle focus between list and detail
			v.focusOnDetail = !v.focusOnDetail
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// View full details
			pkg := v.list.GetCurrentPackage()
			if pkg != nil {
				v.state.SetSelectedPackage(pkg)
				return v, func() tea.Msg {
					return NavigateToDetailsMsg{}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("u"))):
			// Uninstall package
			pkg := v.list.GetCurrentPackage()
			if pkg != nil {
				return v, func() tea.Msg {
					return RequestUninstallMsg{Package: *pkg}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
			// Pin/Unpin package
			pkg := v.list.GetCurrentPackage()
			if pkg != nil {
				return v, func() tea.Msg {
					return TogglePinMsg{PackageName: pkg.Name, Pinned: pkg.Pinned}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			// Refresh list
			return v, func() tea.Msg {
				return RefreshPackagesMsg{}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "space"))):
			// Toggle multi-select mode
			if !v.list.IsMultiMode() {
				v.list.ToggleMultiMode()
			} else {
				v.list.ToggleSelection()
			}
			return v, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Exit multi-select mode or go back
			if v.list.IsMultiMode() {
				v.list.ToggleMultiMode()
				v.list.ClearSelection()
				return v, nil
			}
		}

	case PackageInfoLoadedMsg:
		v.packageInfo = msg.Info
		v.loadingInfo = false
		return v, nil

	case ErrorMsgView:
		v.loadingInfo = false
		return v, nil
	}

	// Update list
	var cmd tea.Cmd
	oldPkg := v.list.GetCurrentPackage()
	v.list, cmd = v.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Check if selection changed
	newPkg := v.list.GetCurrentPackage()
	if newPkg != nil && (oldPkg == nil || oldPkg.Name != newPkg.Name) {
		v.selectedPkg = newPkg
		cmds = append(cmds, v.loadPackageInfo(newPkg))
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *InstalledView) View() string {
	if v.width == 0 || v.height == 0 {
		return "Loading..."
	}

	// Calculate panel dimensions
	leftWidth := int(float64(v.width) * 0.4)
	rightWidth := v.width - leftWidth - 1
	panelHeight := v.height - 3 // Leave room for status bar

	// Left panel: Package list
	leftPanel := v.renderLeftPanel(leftWidth, panelHeight)

	// Right panel: Package details
	rightPanel := v.renderRightPanel(rightWidth, panelHeight)

	// Combine panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Status bar
	statusBar := v.renderStatusBar()

	// Combine vertically
	return lipgloss.JoinVertical(lipgloss.Left, panels, statusBar)
}

func (v *InstalledView) renderLeftPanel(width, height int) string {
	listStyle := styles.PanelStyle
	if !v.focusOnDetail {
		listStyle = styles.ActivePanelStyle
	}

	title := styles.PanelTitleStyle.Render(fmt.Sprintf("📦 Packages (%d)", len(v.state.GetFilteredPackages())))

	v.list.SetSize(width-4, height-3)
	listContent := v.list.View()

	content := lipgloss.JoinVertical(lipgloss.Left, title, listContent)

	return listStyle.
		Width(width - 2).
		Height(height).
		Render(content)
}

func (v *InstalledView) renderRightPanel(width, height int) string {
	panelStyle := styles.PanelStyle
	if v.focusOnDetail {
		panelStyle = styles.ActivePanelStyle
	}

	var content string

	if v.loadingInfo {
		content = styles.DimStyle.Render("Loading package info...")
	} else if v.packageInfo == nil {
		content = styles.DimStyle.Render("Select a package to view details")
	} else {
		content = v.renderPackageInfo()
	}

	title := styles.PanelTitleStyle.Render("ℹ️  Details")

	fullContent := lipgloss.JoinVertical(lipgloss.Left, title, "", content)

	return panelStyle.
		Width(width - 2).
		Height(height).
		Render(fullContent)
}

func (v *InstalledView) renderPackageInfo() string {
	if v.packageInfo == nil {
		return ""
	}

	info := v.packageInfo
	var sections []string

	// Name and version
	nameSection := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.KeyStyle.Render("Name: "),
		styles.ValueStyle.Render(info.Name),
	)
	sections = append(sections, nameSection)

	if info.Version != "" {
		versionSection := lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.KeyStyle.Render("Version: "),
			styles.ValueStyle.Render(info.Version),
		)
		sections = append(sections, versionSection)
	}

	// Type
	pkgType := "Formula"
	if info.Type == brew.TypeCask {
		pkgType = "Cask"
	}
	typeSection := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.KeyStyle.Render("Type: "),
		styles.ValueStyle.Render(pkgType),
	)
	sections = append(sections, typeSection)

	// Description
	if info.Description != "" {
		sections = append(sections, "")
		sections = append(sections, styles.KeyStyle.Render("Description:"))
		sections = append(sections, styles.ValueStyle.Render(info.Description))
	}

	// Homepage
	if info.Homepage != "" {
		sections = append(sections, "")
		homepageSection := lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.KeyStyle.Render("Homepage: "),
			styles.DimStyle.Render(info.Homepage),
		)
		sections = append(sections, homepageSection)
	}

	// Dependencies
	if len(info.Dependencies) > 0 {
		sections = append(sections, "")
		sections = append(sections, styles.KeyStyle.Render("Dependencies:"))
		depList := "  " + strings.Join(info.Dependencies, ", ")
		sections = append(sections, styles.ValueStyle.Render(depList))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (v *InstalledView) renderStatusBar() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Installed: %d", v.state.GetInstalledCount()))

	if v.state.GetOutdatedCount() > 0 {
		parts = append(parts, fmt.Sprintf("Outdated: %d", v.state.GetOutdatedCount()))
	}

	parts = append(parts, "Tab: Switch panel")
	parts = append(parts, "u: Uninstall")
	parts = append(parts, "p: Pin")
	parts = append(parts, "r: Refresh")
	parts = append(parts, "Esc: Back")

	statusText := strings.Join(parts, " • ")

	return styles.StatusBarStyle.
		Width(v.width).
		Render(statusText)
}

func (v *InstalledView) loadPackageInfo(pkg *brew.Package) tea.Cmd {
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

// Message types
type (
	NavigateToDetailsMsg struct{}
	RequestUninstallMsg  struct{ Package brew.Package }
	TogglePinMsg         struct {
		PackageName string
		Pinned      bool
	}
	RefreshPackagesMsg struct{}
)

// Bubble Tea commands

func uninstallPackage(client brew.Client, pkg brew.Package) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		opts := brew.UninstallOptions{
			Cask: pkg.Type == brew.TypeCask,
		}
		err := client.Uninstall(ctx, pkg.Name, opts)
		if err != nil {
			return ErrorMsgView{Err: err}
		}
		return SuccessMsgView{Msg: "Successfully uninstalled " + pkg.Name}
	}
}

func togglePin(client brew.Client, packageName string, currentlyPinned bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		if currentlyPinned {
			err = client.Unpin(ctx, packageName)
		} else {
			err = client.Pin(ctx, packageName)
		}
		if err != nil {
			return ErrorMsgView{Err: err}
		}

		action := "pinned"
		if currentlyPinned {
			action = "unpinned"
		}
		return SuccessMsgView{Msg: "Successfully " + action + " " + packageName}
	}
}

type ErrorMsgView struct{ Err error }
type SuccessMsgView struct{ Msg string }
