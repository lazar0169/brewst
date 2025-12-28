package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents user configuration
type Config struct {
	// Display preferences
	ShowCasksByDefault     bool `json:"show_casks_by_default"`
	ShowFormulaByDefault   bool `json:"show_formula_by_default"`
	ConfirmBeforeInstall   bool `json:"confirm_before_install"`
	ConfirmBeforeUninstall bool `json:"confirm_before_uninstall"`

	// Behavior
	AutoUpdateOnStartup bool `json:"auto_update_on_startup"`
	CacheTTL            int  `json:"cache_ttl"` // seconds

	// UI
	DefaultView string `json:"default_view"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ShowCasksByDefault:     true,
		ShowFormulaByDefault:   true,
		ConfirmBeforeInstall:   true,
		ConfirmBeforeUninstall: true,
		AutoUpdateOnStartup:    false,
		CacheTTL:               300,
		DefaultView:            "home",
	}
}

// LoadConfig loads configuration from disk
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		config := DefaultConfig()
		_ = config.Save() // Ignore save errors
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), err
	}

	// Parse config
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), err
	}

	return &config, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Marshal config
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Write config file
	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "brewst", "config.json"), nil
}

// LoadFavorites loads favorites from disk
func LoadFavorites() ([]string, error) {
	favPath, err := getFavoritesPath()
	if err != nil {
		return []string{}, err
	}

	// Check if favorites file exists
	if _, err := os.Stat(favPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read favorites file
	data, err := os.ReadFile(favPath)
	if err != nil {
		return []string{}, err
	}

	// Parse favorites
	var favorites []string
	if err := json.Unmarshal(data, &favorites); err != nil {
		return []string{}, err
	}

	return favorites, nil
}

// SaveFavorites saves favorites to disk
func SaveFavorites(favorites []string) error {
	favPath, err := getFavoritesPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	favDir := filepath.Dir(favPath)
	if err := os.MkdirAll(favDir, 0755); err != nil {
		return err
	}

	// Marshal favorites
	data, err := json.MarshalIndent(favorites, "", "  ")
	if err != nil {
		return err
	}

	// Write favorites file
	return os.WriteFile(favPath, data, 0644)
}

// getFavoritesPath returns the path to the favorites file
func getFavoritesPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "brewst", "favorites.json"), nil
}
