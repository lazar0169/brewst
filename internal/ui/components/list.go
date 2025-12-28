package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar0169/brewst/internal/brew"
	"github.com/lazar0169/brewst/internal/ui/styles"
)

// PackageItem represents a package in the list
type PackageItem struct {
	pkg      brew.Package
	selected bool
}

// FilterValue implements list.Item
func (i PackageItem) FilterValue() string {
	return i.pkg.Name
}

// Title returns the item title
func (i PackageItem) Title() string {
	prefix := "  "
	if i.selected {
		prefix = "✓ "
	}

	name := i.pkg.Name
	if i.pkg.Type == brew.TypeCask {
		name = styles.CaskStyle.Render(name)
	}

	if i.pkg.Pinned {
		name = styles.PinnedStyle.Render(name + " 📌")
	} else if i.pkg.Outdated {
		name = styles.OutdatedStyle.Render(name + " ⚠")
	} else if i.pkg.Installed {
		name = styles.InstalledStyle.Render(name + " ✓")
	}

	return prefix + name
}

// Description returns the item description
func (i PackageItem) Description() string {
	version := ""
	if i.pkg.Version != "" {
		version = fmt.Sprintf("v%s", i.pkg.Version)
	}

	desc := i.pkg.Description
	if len(desc) > 60 {
		desc = desc[:57] + "..."
	}

	if version != "" && desc != "" {
		return fmt.Sprintf("%s • %s", version, desc)
	} else if version != "" {
		return version
	}
	return desc
}

// PackageList wraps bubbles list for package display
type PackageList struct {
	list      list.Model
	items     []PackageItem
	multiMode bool
}

// NewPackageList creates a new package list
func NewPackageList(width, height int) *PackageList {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetSpacing(0)

	l := list.New([]list.Item{}, delegate, width, height)
	l.Title = "Packages"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styles.TitleStyle
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.Primary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.Primary)

	return &PackageList{
		list:      l,
		items:     []PackageItem{},
		multiMode: false,
	}
}

// SetPackages sets the list packages
func (l *PackageList) SetPackages(packages []brew.Package) {
	items := make([]list.Item, len(packages))
	l.items = make([]PackageItem, len(packages))

	for i, pkg := range packages {
		l.items[i] = PackageItem{pkg: pkg, selected: false}
		items[i] = l.items[i]
	}

	l.list.SetItems(items)
}

// SetTitle sets the list title
func (l *PackageList) SetTitle(title string) {
	l.list.Title = title
}

// SetSize sets the list size
func (l *PackageList) SetSize(width, height int) {
	l.list.SetSize(width, height)
}

// ToggleMultiMode toggles multi-selection mode
func (l *PackageList) ToggleMultiMode() {
	l.multiMode = !l.multiMode
}

// IsMultiMode returns whether multi-select mode is enabled
func (l *PackageList) IsMultiMode() bool {
	return l.multiMode
}

// ToggleSelection toggles selection for the current item
func (l *PackageList) ToggleSelection() {
	idx := l.list.Index()
	if idx >= 0 && idx < len(l.items) {
		l.items[idx].selected = !l.items[idx].selected
		// Update the list items
		items := make([]list.Item, len(l.items))
		for i, item := range l.items {
			items[i] = item
		}
		l.list.SetItems(items)
	}
}

// GetSelected returns all selected packages
func (l *PackageList) GetSelected() []brew.Package {
	var selected []brew.Package
	for _, item := range l.items {
		if item.selected {
			selected = append(selected, item.pkg)
		}
	}
	return selected
}

// ClearSelection clears all selections
func (l *PackageList) ClearSelection() {
	for i := range l.items {
		l.items[i].selected = false
	}
	items := make([]list.Item, len(l.items))
	for i, item := range l.items {
		items[i] = item
	}
	l.list.SetItems(items)
}

// GetCurrentPackage returns the currently highlighted package
func (l *PackageList) GetCurrentPackage() *brew.Package {
	idx := l.list.Index()
	if idx >= 0 && idx < len(l.items) {
		return &l.items[idx].pkg
	}
	return nil
}

// IsEmpty returns whether the list is empty
func (l *PackageList) IsEmpty() bool {
	return len(l.items) == 0
}

// Update handles list updates
func (l *PackageList) Update(msg tea.Msg) (*PackageList, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle space for multi-select
		if l.multiMode && key.Matches(msg, key.NewBinding(key.WithKeys(" ", "space"))) {
			l.ToggleSelection()
			return l, nil
		}
	}

	l.list, cmd = l.list.Update(msg)
	return l, cmd
}

// View renders the list
func (l *PackageList) View() string {
	if l.multiMode {
		help := styles.HelpStyle.Render("Multi-select mode: Space to toggle, Enter to confirm")
		return lipgloss.JoinVertical(lipgloss.Left, l.list.View(), help)
	}
	return l.list.View()
}

// FilterValue returns the current filter value
func (l *PackageList) FilterValue() string {
	return l.list.FilterValue()
}

// ResetFilter resets the list filter
func (l *PackageList) ResetFilter() {
	l.list.ResetFilter()
}

// FormatPackageList formats a list of packages as a string
func FormatPackageList(packages []brew.Package) string {
	var builder strings.Builder

	for _, pkg := range packages {
		name := pkg.Name
		if pkg.Type == brew.TypeCask {
			name = fmt.Sprintf("%s (cask)", name)
		}

		version := ""
		if pkg.Version != "" {
			version = fmt.Sprintf(" v%s", pkg.Version)
		}

		status := ""
		if pkg.Pinned {
			status = " [pinned]"
		} else if pkg.Outdated {
			status = " [outdated]"
		}

		builder.WriteString(fmt.Sprintf("- %s%s%s\n", name, version, status))
	}

	return builder.String()
}
