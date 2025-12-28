package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar0169/brewst/internal/brew"
	"github.com/lazar0169/brewst/internal/state"
	"github.com/lazar0169/brewst/internal/ui/styles"
)

// DetailsView shows package details
type DetailsView struct {
	client brew.Client
	state  *state.State

	packageInfo *brew.PackageInfo
	loading     bool
	width       int
	height      int
}

// NewDetailsView creates a new package details view
func NewDetailsView(client brew.Client, state *state.State) *DetailsView {
	return &DetailsView{
		client:  client,
		state:   state,
		loading: false,
	}
}

// SetSize sets the view size
func (v *DetailsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Init initializes the view
func (v *DetailsView) Init() tea.Cmd {
	if v.state.SelectedPackage != nil {
		return v.loadPackageInfo(v.state.SelectedPackage)
	}
	return nil
}

// Update handles messages
func (v *DetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("i"))):
			// Install/Uninstall
			if v.state.SelectedPackage != nil {
				if v.state.SelectedPackage.Installed {
					return v, func() tea.Msg {
						return RequestUninstallMsg{Package: *v.state.SelectedPackage}
					}
				} else {
					return v, func() tea.Msg {
						return RequestInstallMsg{Package: *v.state.SelectedPackage}
					}
				}
			}
		}

	case PackageInfoLoadedMsg:
		v.packageInfo = msg.Info
		v.loading = false
		return v, nil

	case ErrorMsgView:
		v.loading = false
		return v, nil
	}

	return v, nil
}

// View renders the view
func (v *DetailsView) View() string {
	if v.loading {
		return styles.AppStyle.Render("Loading package details...")
	}

	if v.packageInfo == nil {
		return styles.AppStyle.Render("No package selected")
	}

	info := v.packageInfo

	// Package header
	header := styles.TitleStyle.Render(info.Name)
	if info.Version != "" {
		version := styles.DimStyle.Render(fmt.Sprintf("Version: %s", info.Version))
		header = lipgloss.JoinVertical(lipgloss.Left, header, version)
	}

	// Type
	pkgType := "Formula"
	if info.Type == brew.TypeCask {
		pkgType = "Cask"
	}
	typeStr := fmt.Sprintf("%s %s", styles.KeyStyle.Render("Type:"), pkgType)

	// Description
	desc := ""
	if info.Description != "" {
		desc = fmt.Sprintf("%s %s", styles.KeyStyle.Render("Description:"), info.Description)
	}

	// Homepage
	homepage := ""
	if info.Homepage != "" {
		homepage = fmt.Sprintf("%s %s", styles.KeyStyle.Render("Homepage:"), info.Homepage)
	}

	// Dependencies
	deps := ""
	if len(info.Dependencies) > 0 {
		depsTitle := styles.KeyStyle.Render("Dependencies:")
		depsList := strings.Join(info.Dependencies, ", ")
		deps = fmt.Sprintf("%s %s", depsTitle, depsList)
	}

	// Build dependencies
	buildDeps := ""
	if len(info.BuildDeps) > 0 {
		buildDepsTitle := styles.KeyStyle.Render("Build Dependencies:")
		buildDepsList := strings.Join(info.BuildDeps, ", ")
		buildDeps = fmt.Sprintf("%s %s", buildDepsTitle, buildDepsList)
	}

	// Caveats
	caveats := ""
	if info.Caveats != "" {
		caveatsTitle := styles.KeyStyle.Render("Caveats:")
		caveats = lipgloss.JoinVertical(lipgloss.Left, caveatsTitle, info.Caveats)
	}

	// Assemble content
	var sections []string
	sections = append(sections, header, "", typeStr)
	if desc != "" {
		sections = append(sections, desc)
	}
	if homepage != "" {
		sections = append(sections, homepage)
	}
	if deps != "" {
		sections = append(sections, "", deps)
	}
	if buildDeps != "" {
		sections = append(sections, buildDeps)
	}
	if caveats != "" {
		sections = append(sections, "", caveats)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Help text
	helpText := "i: Install/Uninstall | Esc: Back"
	help := styles.HelpStyle.Render(helpText)

	return styles.AppStyle.Render(content) + "\n" + help
}

func (v *DetailsView) loadPackageInfo(pkg *brew.Package) tea.Cmd {
	v.loading = true
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
	PackageInfoLoadedMsg struct{ Info *brew.PackageInfo }
	RequestInstallMsg    struct{ Package brew.Package }
)
