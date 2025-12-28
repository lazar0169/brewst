package brew

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parsePackageNames parses plain text output from brew list (just package names)
func parsePackageNames(output string, pkgType PackageType) []Package {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	packages := make([]Package, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by whitespace in case multiple packages on one line
		names := strings.Fields(line)
		for _, name := range names {
			pkg := Package{
				Name:      name,
				FullName:  name,
				Type:      pkgType,
				Installed: true,
			}
			packages = append(packages, pkg)
		}
	}

	return packages
}

// parsePackageNamesWithVersions parses output from brew list --versions
// Format: "package-name 1.2.3" or "package-name 1.2.3 1.2.4" (multiple versions)
func parsePackageNamesWithVersions(output string, pkgType PackageType) []Package {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	packages := make([]Package, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by whitespace: first field is name, rest are versions
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		name := fields[0]
		version := ""
		if len(fields) > 1 {
			version = fields[1]
		}

		pkg := Package{
			Name:      name,
			FullName:  name,
			Version:   version,
			Type:      pkgType,
			Installed: true,
		}
		packages = append(packages, pkg)
	}

	return packages
}

// parsePackages parses JSON output from brew list command
func parsePackages(output string, pkgType PackageType) ([]Package, error) {
	if strings.TrimSpace(output) == "" {
		return []Package{}, nil
	}

	var rawPackages []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &rawPackages); err != nil {
		return nil, fmt.Errorf("failed to parse packages: %w", err)
	}

	packages := make([]Package, 0, len(rawPackages))
	for _, raw := range rawPackages {
		pkg := Package{
			Type:      pkgType,
			Installed: true,
		}

		if name, ok := raw["name"].(string); ok {
			pkg.Name = name
		}
		if fullName, ok := raw["full_name"].(string); ok {
			pkg.FullName = fullName
		} else {
			pkg.FullName = pkg.Name
		}

		// Handle version - might be in different places
		if versions, ok := raw["installed"].([]interface{}); ok && len(versions) > 0 {
			if vmap, ok := versions[0].(map[string]interface{}); ok {
				if version, ok := vmap["version"].(string); ok {
					pkg.Version = version
				}
			}
		}
		if pkg.Version == "" {
			if version, ok := raw["version"].(string); ok {
				pkg.Version = version
			}
		}

		if desc, ok := raw["desc"].(string); ok {
			pkg.Description = desc
		}
		if homepage, ok := raw["homepage"].(string); ok {
			pkg.Homepage = homepage
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// parsePackageInfo parses JSON output from brew info command
func parsePackageInfo(output string, pkgType PackageType) (*PackageInfo, error) {
	var rawPackages []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &rawPackages); err != nil {
		return nil, fmt.Errorf("failed to parse package info: %w", err)
	}

	if len(rawPackages) == 0 {
		return nil, fmt.Errorf("no package information found")
	}

	raw := rawPackages[0]
	info := &PackageInfo{
		Package: Package{
			Type: pkgType,
		},
	}

	if name, ok := raw["name"].(string); ok {
		info.Name = name
	}
	if fullName, ok := raw["full_name"].(string); ok {
		info.FullName = fullName
	} else {
		info.FullName = info.Name
	}
	if version, ok := raw["version"].(string); ok {
		info.Version = version
	}
	if desc, ok := raw["desc"].(string); ok {
		info.Description = desc
	}
	if homepage, ok := raw["homepage"].(string); ok {
		info.Homepage = homepage
	}
	if caveats, ok := raw["caveats"].(string); ok {
		info.Caveats = caveats
	}

	// Parse dependencies
	if deps, ok := raw["dependencies"].([]interface{}); ok {
		info.Dependencies = make([]string, 0, len(deps))
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				info.Dependencies = append(info.Dependencies, depStr)
			}
		}
	}

	// Parse build dependencies
	if buildDeps, ok := raw["build_dependencies"].([]interface{}); ok {
		info.BuildDeps = make([]string, 0, len(buildDeps))
		for _, dep := range buildDeps {
			if depStr, ok := dep.(string); ok {
				info.BuildDeps = append(info.BuildDeps, depStr)
			}
		}
	}

	return info, nil
}

// parseOutdated parses JSON output from brew outdated command
func parseOutdated(output string) ([]OutdatedPackage, error) {
	if strings.TrimSpace(output) == "" {
		return []OutdatedPackage{}, nil
	}

	var rawPackages []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &rawPackages); err != nil {
		return nil, fmt.Errorf("failed to parse outdated packages: %w", err)
	}

	packages := make([]OutdatedPackage, 0, len(rawPackages))
	for _, raw := range rawPackages {
		pkg := OutdatedPackage{}

		if name, ok := raw["name"].(string); ok {
			pkg.Name = name
		}

		// Handle installed versions - might be array or string
		if versions, ok := raw["installed_versions"].([]interface{}); ok && len(versions) > 0 {
			if version, ok := versions[0].(string); ok {
				pkg.CurrentVersion = version
			}
		} else if version, ok := raw["installed_versions"].(string); ok {
			pkg.CurrentVersion = version
		}

		if version, ok := raw["current_version"].(string); ok {
			pkg.LatestVersion = version
		}

		if pinned, ok := raw["pinned"].(bool); ok {
			pkg.Pinned = pinned
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// parseSearchResults parses plain text output from brew search
func parseSearchResults(output string) ([]Package, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	packages := make([]Package, 0)

	currentType := TypeFormula
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect section headers
		if strings.Contains(line, "Formulae") {
			currentType = TypeFormula
			continue
		}
		if strings.Contains(line, "Casks") {
			currentType = TypeCask
			continue
		}

		// Skip separator lines
		if strings.HasPrefix(line, "=") {
			continue
		}

		// Parse package name
		pkg := Package{
			Name:      line,
			FullName:  line,
			Type:      currentType,
			Installed: false,
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// parseTaps parses plain text output from brew tap
func parseTaps(output string) ([]Tap, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	taps := make([]Tap, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		tap := Tap{
			Name:     line,
			Official: strings.HasPrefix(line, "homebrew/"),
		}
		taps = append(taps, tap)
	}

	return taps, nil
}

// parsePackageInfoText parses plain text output from brew info
func parsePackageInfoText(output, name string, pkgType PackageType) *PackageInfo {
	lines := strings.Split(output, "\n")

	info := &PackageInfo{
		Package: Package{
			Name:      name,
			FullName:  name,
			Type:      pkgType,
			Installed: true,
		},
		Dependencies: []string{},
		BuildDeps:    []string{},
	}

	// Parse basic info from first line (usually contains name and version)
	if len(lines) > 0 {
		firstLine := lines[0]
		// Try to extract version (format: "name: stable version")
		if strings.Contains(firstLine, ":") {
			parts := strings.SplitN(firstLine, ":", 2)
			if len(parts) == 2 {
				versionPart := strings.TrimSpace(parts[1])
				// Extract version (first word after colon)
				versionFields := strings.Fields(versionPart)
				if len(versionFields) > 0 {
					if versionFields[0] == "stable" && len(versionFields) > 1 {
						info.Version = versionFields[1]
					} else {
						info.Version = versionFields[0]
					}
				}
			}
		}
	}

	// Parse description and other info
	inDependencies := false
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check if we're in the dependencies section
		if strings.Contains(line, "==> Dependencies") {
			inDependencies = true
			continue
		}

		// Exit dependencies section when we hit another section
		if inDependencies && strings.HasPrefix(line, "==>") {
			inDependencies = false
		}

		// Parse dependencies if we're in that section
		if inDependencies {
			// Look for "Required:" or "Build:" lines
			if strings.HasPrefix(line, "Required:") {
				depsStr := strings.TrimPrefix(line, "Required:")
				depsStr = strings.TrimSpace(depsStr)
				// Split by comma and clean up
				deps := strings.Split(depsStr, ",")
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					// Remove checkmarks and X marks
					dep = strings.TrimSuffix(dep, "✔")
					dep = strings.TrimSuffix(dep, "✘")
					dep = strings.TrimSpace(dep)
					if dep != "" {
						info.Dependencies = append(info.Dependencies, dep)
					}
				}
			} else if strings.HasPrefix(line, "Build:") {
				depsStr := strings.TrimPrefix(line, "Build:")
				depsStr = strings.TrimSpace(depsStr)
				deps := strings.Split(depsStr, ",")
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					dep = strings.TrimSuffix(dep, "✔")
					dep = strings.TrimSuffix(dep, "✘")
					dep = strings.TrimSpace(dep)
					if dep != "" {
						info.BuildDeps = append(info.BuildDeps, dep)
					}
				}
			}
		}

		// Description is usually the second non-empty line
		if info.Description == "" && !strings.HasPrefix(line, name+":") &&
		   !strings.HasPrefix(line, "http") && !strings.HasPrefix(line, "From:") &&
		   !strings.HasPrefix(line, "/") && // Skip install path
		   !strings.Contains(line, "==") && line != "" && i > 0 {
			info.Description = line
		}

		// Look for homepage
		if strings.HasPrefix(line, "http") && !strings.Contains(line, "github.com/Homebrew") {
			info.Homepage = line
		}
	}

	return info
}

// parseOutdatedText parses plain text output from brew outdated
func parseOutdatedText(output string) []OutdatedPackage {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	packages := make([]OutdatedPackage, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "package (current) < latest" or just "package"
		// Example: "git (2.39.0) < 2.39.1"
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		pkg := OutdatedPackage{
			Name: parts[0],
		}

		// Try to parse versions if present
		if len(parts) >= 4 && parts[2] == "<" {
			// Remove parentheses from current version
			pkg.CurrentVersion = strings.Trim(parts[1], "()")
			pkg.LatestVersion = parts[3]
		}

		packages = append(packages, pkg)
	}

	return packages
}
