package views

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lazar/brewst/internal/brew"
	"github.com/lazar/brewst/internal/state"
	"github.com/lazar/brewst/internal/ui/components"
	"github.com/lazar/brewst/internal/ui/styles"
	"github.com/sahilm/fuzzy"
)

// SearchView provides search functionality
type SearchView struct {
	client brew.Client
	state  *state.State

	textInput textinput.Model
	list      *components.PackageList
	results   []brew.Package
	searching bool
	width     int
	height    int
}

// NewSearchView creates a new search view
func NewSearchView(client brew.Client, state *state.State) *SearchView {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return &SearchView{
		client:    client,
		state:     state,
		textInput: ti,
		list:      components.NewPackageList(80, 20),
		results:   []brew.Package{},
		searching: false,
	}
}

// SetSize sets the view size
func (v *SearchView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.list.SetSize(width-4, height-8)
}

// Init initializes the view
func (v *SearchView) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (v *SearchView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// If in search input, perform search
			if v.textInput.Focused() {
				query := v.textInput.Value()
				if query != "" {
					return v, v.performSearch(query)
				}
			} else {
				// If in results list, show details or install
				pkg := v.list.GetCurrentPackage()
				if pkg != nil {
					v.state.SetSelectedPackage(pkg)
					if pkg.Installed {
						return v, func() tea.Msg {
							return NavigateToDetailsMsg{}
						}
					} else {
						return v, func() tea.Msg {
							return RequestInstallMsg{Package: *pkg}
						}
					}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			// Toggle focus between input and list
			if v.textInput.Focused() && len(v.results) > 0 {
				v.textInput.Blur()
			} else {
				v.textInput.Focus()
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Focus back to input if in list
			if !v.textInput.Focused() {
				v.textInput.Focus()
				return v, nil
			}
		}

	case SearchResultsMsg:
		v.results = msg.Results
		v.searching = false
		// Apply fuzzy search if there's a query
		query := v.textInput.Value()
		if query != "" {
			v.results = v.fuzzyFilter(query, v.results)
		}
		v.list.SetPackages(v.results)
		v.list.SetTitle("Search Results")
		return v, nil

	case ErrorMsgView:
		v.searching = false
		return v, nil
	}

	// Update text input
	if v.textInput.Focused() {
		var cmd tea.Cmd
		v.textInput, cmd = v.textInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else {
		// Update list
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

// View renders the view
func (v *SearchView) View() string {
	// Search input
	searchBox := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.TitleStyle.Render("Search Packages"),
		"",
		v.textInput.View(),
	)

	// Status
	status := ""
	if v.searching {
		status = styles.DimStyle.Render("Searching...")
	} else if len(v.results) > 0 {
		status = styles.DimStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, "Found ", fmt.Sprint(len(v.results)), " packages"))
	}

	// Results list
	listView := ""
	if len(v.results) > 0 {
		listView = v.list.View()
	} else if v.textInput.Value() != "" && !v.searching {
		listView = styles.DimStyle.Render("No results found")
	}

	// Help
	helpText := "Enter: Search/Install | Tab: Toggle focus | Esc: Back"
	help := styles.HelpStyle.Render(helpText)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		searchBox,
		"",
		status,
		"",
		listView,
		"",
		help,
	)

	return styles.AppStyle.Render(content)
}

func (v *SearchView) performSearch(query string) tea.Cmd {
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

func (v *SearchView) fuzzyFilter(query string, packages []brew.Package) []brew.Package {
	// Create list of package names
	names := make([]string, len(packages))
	for i, pkg := range packages {
		names[i] = pkg.Name
	}

	// Perform fuzzy search
	matches := fuzzy.Find(query, names)

	// Build filtered results
	filtered := make([]brew.Package, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, packages[match.Index])
	}

	return filtered
}

// Message types
type SearchResultsMsg struct{ Results []brew.Package }
