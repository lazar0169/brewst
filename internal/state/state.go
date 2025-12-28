package state

import (
	"sync"

	"github.com/lazar0169/brewst/internal/brew"
)

// State represents the global application state
type State struct {
	mu sync.RWMutex

	// Package data
	InstalledPackages []brew.Package
	SearchResults     []brew.Package
	OutdatedPackages  []brew.OutdatedPackage
	Taps              []brew.Tap

	// Selected package for details view
	SelectedPackage *brew.Package

	// UI state
	Loading      bool
	LoadingMsg   string
	ErrorMsg     string
	SuccessMsg   string
	LastError    error

	// Filters
	ShowFormulae bool
	ShowCasks    bool
	OnlyOutdated bool
	OnlyPinned   bool

	// User preferences
	Favorites []string

	// Statistics
	TotalInstalled int
	TotalOutdated  int
}

// NewState creates a new application state
func NewState() *State {
	return &State{
		ShowFormulae: true,
		ShowCasks:    true,
		Favorites:    []string{},
	}
}

// SetInstalled sets the installed packages
func (s *State) SetInstalled(packages []brew.Package) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InstalledPackages = packages
	s.TotalInstalled = len(packages)
}

// SetOutdated sets the outdated packages
func (s *State) SetOutdated(packages []brew.OutdatedPackage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.OutdatedPackages = packages
	s.TotalOutdated = len(packages)
}

// SetSearchResults sets the search results
func (s *State) SetSearchResults(packages []brew.Package) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SearchResults = packages
}

// SetSelectedPackage sets the currently selected package
func (s *State) SetSelectedPackage(pkg *brew.Package) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SelectedPackage = pkg
}

// SetLoading sets the loading state
func (s *State) SetLoading(loading bool, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Loading = loading
	s.LoadingMsg = msg
}

// SetError sets an error message
func (s *State) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.ErrorMsg = err.Error()
		s.LastError = err
	} else {
		s.ErrorMsg = ""
		s.LastError = nil
	}
}

// SetSuccess sets a success message
func (s *State) SetSuccess(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SuccessMsg = msg
}

// ClearMessages clears all messages
func (s *State) ClearMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorMsg = ""
	s.SuccessMsg = ""
}

// GetFilteredPackages returns packages based on current filters
func (s *State) GetFilteredPackages() []brew.Package {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []brew.Package
	for _, pkg := range s.InstalledPackages {
		// Filter by type
		if !s.ShowFormulae && pkg.Type == brew.TypeFormula {
			continue
		}
		if !s.ShowCasks && pkg.Type == brew.TypeCask {
			continue
		}

		// Filter by status
		if s.OnlyOutdated && !pkg.Outdated {
			continue
		}
		if s.OnlyPinned && !pkg.Pinned {
			continue
		}

		filtered = append(filtered, pkg)
	}

	return filtered
}

// IsFavorite checks if a package is in favorites
func (s *State) IsFavorite(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, fav := range s.Favorites {
		if fav == name {
			return true
		}
	}
	return false
}

// ToggleFavorite toggles a package in favorites
func (s *State) ToggleFavorite(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, fav := range s.Favorites {
		if fav == name {
			// Remove from favorites
			s.Favorites = append(s.Favorites[:i], s.Favorites[i+1:]...)
			return
		}
	}

	// Add to favorites
	s.Favorites = append(s.Favorites, name)
}

// GetInstalledCount returns the number of installed packages
func (s *State) GetInstalledCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalInstalled
}

// GetOutdatedCount returns the number of outdated packages
func (s *State) GetOutdatedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TotalOutdated
}
