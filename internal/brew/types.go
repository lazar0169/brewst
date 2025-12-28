package brew

import "time"

// PackageType represents the type of package
type PackageType string

const (
	TypeFormula PackageType = "formula"
	TypeCask    PackageType = "cask"
)

// Package represents a Homebrew package (formula or cask)
type Package struct {
	Name        string      `json:"name"`
	FullName    string      `json:"full_name"`
	Version     string      `json:"version"`
	Description string      `json:"desc"`
	Homepage    string      `json:"homepage"`
	Type        PackageType `json:"-"`
	Installed   bool        `json:"-"`
	Outdated    bool        `json:"-"`
	Pinned      bool        `json:"-"`
}

// PackageInfo represents detailed information about a package
type PackageInfo struct {
	Package
	Dependencies []string  `json:"dependencies"`
	BuildDeps    []string  `json:"build_dependencies"`
	Caveats      string    `json:"caveats"`
	InstallDate  time.Time `json:"-"`
}

// OutdatedPackage represents a package that has an available update
type OutdatedPackage struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"installed_versions"`
	LatestVersion  string `json:"current_version"`
	Pinned         bool   `json:"pinned"`
}

// Tap represents a Homebrew tap (third-party repository)
type Tap struct {
	Name     string
	Official bool
	Remote   string
}

// InstallOptions represents options for installing packages
type InstallOptions struct {
	Cask  bool
	Force bool
}

// UninstallOptions represents options for uninstalling packages
type UninstallOptions struct {
	Cask  bool
	Force bool
}
