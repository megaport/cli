//go:build !js && !wasm

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrProfileNotFound = errors.New("profile not found")

	// renameFile and chmodFile are variables so tests can inject errors into the backup path.
	renameFile = os.Rename
	chmodFile  = os.Chmod
)

// ConfigManager handles configuration operations
type ConfigManager struct {
	config     *ConfigFile
	configPath string
}

func NewConfigManager() (*ConfigManager, error) {
	configPath, err := GetConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config file path: %w", err)
	}

	manager := &ConfigManager{
		configPath: configPath,
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := NewConfigFile()
		configData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		configDir, err := GetConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		if err := os.WriteFile(configPath, configData, 0600); err != nil {
			return nil, fmt.Errorf("failed to create default config: failed to write config file: %w", err)
		}

		manager.config = config
		return manager, nil
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ConfigFile
	err = json.Unmarshal(configData, &config)
	if err != nil {
		backupPath := configPath + ".corrupt-" + time.Now().Format("20060102-150405.000000000")
		if renameErr := renameFile(configPath, backupPath); renameErr != nil {
			if !os.IsNotExist(renameErr) {
				return nil, fmt.Errorf("config file is corrupted and could not be preserved: %w", renameErr)
			}
			// Another process already recovered the file; create a fresh default below.
			fmt.Fprintf(os.Stderr, "Warning: Config file is corrupted and was concurrently recovered.\n")
		} else {
			if chmodErr := chmodFile(backupPath, 0600); chmodErr != nil {
				// chmod failure is non-fatal: the backup is preserved and the fresh
				// default will still be created. Warn so the user can restrict manually.
				fmt.Fprintf(os.Stderr, "Warning: Config file is corrupted. Original preserved at %s (run: chmod 0600 -- %q)\n", backupPath, backupPath)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Config file is corrupted. Original preserved at %s\n", backupPath)
			}
		}
		config = *NewConfigFile()

		configData, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		err = os.WriteFile(configPath, configData, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}
		if err := os.Chmod(configPath, 0600); err != nil {
			return nil, fmt.Errorf("failed to set permissions on config file: %w", err)
		}
	}

	if config.Version < ConfigVersion {
		config.Version = ConfigVersion
		manager.config = &config
		err = manager.Save()
		if err != nil {
			return nil, fmt.Errorf("failed to save upgraded config: %w", err)
		}
	}

	needsSave := false
	if config.Profiles == nil {
		config.Profiles = make(map[string]*Profile)
		needsSave = true
	}
	if config.Defaults == nil {
		config.Defaults = make(map[string]interface{})
		needsSave = true
	}
	manager.config = &config
	if needsSave {
		if err := manager.Save(); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return manager, nil
}

func (m *ConfigManager) GetCurrentProfile() (*Profile, string, error) {
	profileName := m.config.ActiveProfile
	profile, exists := m.config.Profiles[profileName]
	if !exists {
		return nil, "", ErrProfileNotFound
	}
	return profile, profileName, nil
}

func (m *ConfigManager) GetProfile(name string) (*Profile, error) {
	profile, exists := m.config.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return profile, nil
}

func (m *ConfigManager) CreateProfile(name, accessKey, secretKey, environment, description string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if len(strings.TrimSpace(name)) == 0 {
		return fmt.Errorf("profile name cannot be just whitespace")
	}
	if m.config == nil {
		m.config = NewConfigFile()
	}
	if m.config.Profiles == nil {
		m.config.Profiles = make(map[string]*Profile)
	}
	m.config.Profiles[name] = &Profile{
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		Environment: environment,
		Description: description,
	}
	return m.Save()
}

func (m *ConfigManager) UpdateProfile(name, accessKey, secretKey, environment string, updateDescription bool, description string) error {
	profile, exists := m.config.Profiles[name]
	if !exists {
		return ErrProfileNotFound
	}
	if accessKey != "" {
		profile.AccessKey = accessKey
	}
	if secretKey != "" {
		profile.SecretKey = secretKey
	}
	if environment != "" {
		profile.Environment = environment
	}
	if updateDescription {
		profile.Description = description
	}
	return m.Save()
}

func (m *ConfigManager) DeleteProfile(name string) error {
	if _, exists := m.config.Profiles[name]; !exists {
		return ErrProfileNotFound
	}
	if m.config.ActiveProfile == name {
		return fmt.Errorf("cannot delete active profile; use 'config use-profile' to switch profiles first")
	}
	delete(m.config.Profiles, name)
	return m.Save()
}

func (m *ConfigManager) ListProfiles() (map[string]*Profile, error) {
	if m == nil || m.config == nil || m.config.Profiles == nil {
		return make(map[string]*Profile), nil
	}
	return m.config.Profiles, nil
}

func (m *ConfigManager) UseProfile(name string) error {
	if _, exists := m.config.Profiles[name]; !exists {
		return ErrProfileNotFound
	}
	m.config.ActiveProfile = name
	return m.Save()
}

func (m *ConfigManager) GetDefault(key string) (interface{}, bool) {
	value, exists := m.config.Defaults[key]
	return value, exists
}

func (m *ConfigManager) SetDefault(key string, value interface{}) error {
	m.config.Defaults[key] = value
	return m.Save()
}

func (m *ConfigManager) Save() error {
	configPath := m.configPath
	configData, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(configPath), ".megaport-config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	n, err := tmpFile.Write(configData)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp config file: %w", err)
	}
	if n != len(configData) {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp config file: short write (%d of %d bytes)", n, len(configData))
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp config file: %w", err)
	}
	if err := chmodFile(tmpPath, 0600); err != nil {
		return fmt.Errorf("failed to set permissions on temp config file: %w", err)
	}
	if err := renameFile(tmpPath, configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	// Rename already carries over the temp file's 0600 mode, but re-tighten
	// explicitly in case configPath pre-existed with looser permissions on a
	// platform where rename doesn't replace the mode bits.
	if err := chmodFile(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set permissions on config file: %w", err)
	}
	return nil
}

func (m *ConfigManager) Export() (*ConfigFile, error) {
	export := &ConfigFile{
		Version:       m.config.Version,
		ActiveProfile: m.config.ActiveProfile,
		Profiles:      make(map[string]*Profile),
		Defaults:      m.config.Defaults,
	}
	for name, profile := range m.config.Profiles {
		export.Profiles[name] = &Profile{
			AccessKey:   "[REDACTED]",
			SecretKey:   "[REDACTED]",
			Environment: profile.Environment,
			Description: profile.Description,
		}
	}
	return export, nil
}

func (m *ConfigManager) RemoveDefault(key string) error {
	if _, exists := m.config.Defaults[key]; exists {
		delete(m.config.Defaults, key)
		return m.Save()
	}
	return nil
}

func (m *ConfigManager) ClearDefaults() error {
	m.config.Defaults = make(map[string]interface{})
	return m.Save()
}
