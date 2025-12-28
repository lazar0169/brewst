package brew

import (
	"context"
)

// Client defines the interface for interacting with Homebrew
type Client interface {
	// ListInstalled returns all installed packages
	ListInstalled(ctx context.Context, formulae bool, casks bool) ([]Package, error)

	// Search searches for packages in the Homebrew repository
	Search(ctx context.Context, query string) ([]Package, error)

	// Info returns detailed information about a package
	Info(ctx context.Context, name string, cask bool) (*PackageInfo, error)

	// Install installs a package
	Install(ctx context.Context, name string, opts InstallOptions) error

	// Uninstall uninstalls a package
	Uninstall(ctx context.Context, name string, opts UninstallOptions) error

	// Update updates Homebrew
	Update(ctx context.Context) error

	// Upgrade upgrades packages
	Upgrade(ctx context.Context, packages []string) error

	// Outdated returns packages that have updates available
	Outdated(ctx context.Context) ([]OutdatedPackage, error)

	// Pin pins a package to prevent updates
	Pin(ctx context.Context, name string) error

	// Unpin unpins a package
	Unpin(ctx context.Context, name string) error

	// Doctor runs brew doctor diagnostics
	Doctor(ctx context.Context) (string, error)

	// ListTaps returns all taps
	ListTaps(ctx context.Context) ([]Tap, error)

	// TapAdd adds a new tap
	TapAdd(ctx context.Context, name string) error

	// TapRemove removes a tap
	TapRemove(ctx context.Context, name string) error

	// Cleanup removes old versions and cache
	Cleanup(ctx context.Context) error

	// Autoremove uninstalls formulae that were only installed as dependencies
	Autoremove(ctx context.Context) error
}

// NewClient creates a new Homebrew client
func NewClient() Client {
	return &client{}
}

type client struct{}

func (c *client) ListInstalled(ctx context.Context, formulae bool, casks bool) ([]Package, error) {
	var packages []Package

	if formulae {
		output, err := execute(ctx, "list", "--formula", "--versions")
		if err != nil {
			return nil, err
		}
		formulas := parsePackageNamesWithVersions(output, TypeFormula)
		packages = append(packages, formulas...)
	}

	if casks {
		output, err := execute(ctx, "list", "--cask", "--versions")
		if err != nil {
			return nil, err
		}
		caskList := parsePackageNamesWithVersions(output, TypeCask)
		packages = append(packages, caskList...)
	}

	return packages, nil
}

func (c *client) Search(ctx context.Context, query string) ([]Package, error) {
	output, err := execute(ctx, "search", query)
	if err != nil {
		return nil, err
	}
	return parseSearchResults(output)
}

func (c *client) Info(ctx context.Context, name string, cask bool) (*PackageInfo, error) {
	args := []string{"info", name}
	if cask {
		args = append(args, "--cask")
	}

	output, err := execute(ctx, args...)
	if err != nil {
		return nil, err
	}

	pkgType := TypeFormula
	if cask {
		pkgType = TypeCask
	}

	return parsePackageInfoText(output, name, pkgType), nil
}

func (c *client) Install(ctx context.Context, name string, opts InstallOptions) error {
	args := []string{"install", name}
	if opts.Cask {
		args = append(args, "--cask")
	}
	if opts.Force {
		args = append(args, "--force")
	}

	_, err := execute(ctx, args...)
	return err
}

func (c *client) Uninstall(ctx context.Context, name string, opts UninstallOptions) error {
	args := []string{"uninstall", name}
	if opts.Cask {
		args = append(args, "--cask")
	}
	if opts.Force {
		args = append(args, "--force")
	}

	_, err := execute(ctx, args...)
	return err
}

func (c *client) Update(ctx context.Context) error {
	_, err := execute(ctx, "update")
	return err
}

func (c *client) Upgrade(ctx context.Context, packages []string) error {
	args := []string{"upgrade"}
	args = append(args, packages...)

	_, err := execute(ctx, args...)
	return err
}

func (c *client) Outdated(ctx context.Context) ([]OutdatedPackage, error) {
	output, err := execute(ctx, "outdated")
	if err != nil {
		return []OutdatedPackage{}, nil
	}
	return parseOutdatedText(output), nil
}

func (c *client) Pin(ctx context.Context, name string) error {
	_, err := execute(ctx, "pin", name)
	return err
}

func (c *client) Unpin(ctx context.Context, name string) error {
	_, err := execute(ctx, "unpin", name)
	return err
}

func (c *client) Doctor(ctx context.Context) (string, error) {
	return execute(ctx, "doctor")
}

func (c *client) ListTaps(ctx context.Context) ([]Tap, error) {
	output, err := execute(ctx, "tap")
	if err != nil {
		return nil, err
	}
	return parseTaps(output)
}

func (c *client) TapAdd(ctx context.Context, name string) error {
	_, err := execute(ctx, "tap", name)
	return err
}

func (c *client) TapRemove(ctx context.Context, name string) error {
	_, err := execute(ctx, "untap", name)
	return err
}

func (c *client) Cleanup(ctx context.Context) error {
	_, err := execute(ctx, "cleanup")
	return err
}

func (c *client) Autoremove(ctx context.Context) error {
	_, err := execute(ctx, "autoremove")
	return err
}
