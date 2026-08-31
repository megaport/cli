//go:build !js && !wasm

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "megaport-config-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

	return tempDir
}

func TestNewConfigManager(t *testing.T) {
	tempDir := setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)
	require.NotNil(t, manager)

	configPath := filepath.Join(tempDir, "config.json")
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "Config file should exist")

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var config ConfigFile
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	assert.Equal(t, ConfigVersion, config.Version)
	assert.Empty(t, config.ActiveProfile)
	assert.Empty(t, config.Profiles)
	assert.NotNil(t, config.Defaults)
}

func TestCreateProfile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "Test profile")
	require.NoError(t, err)
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)

	profile, exists := profiles["test-profile"]
	assert.True(t, exists, "Profile should exist")
	assert.Equal(t, "access123", profile.AccessKey)
	assert.Equal(t, "secret123", profile.SecretKey)
	assert.Equal(t, "production", profile.Environment)
	assert.Equal(t, "Test profile", profile.Description)
}

func TestUpdateProfile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "Original")
	require.NoError(t, err)

	err = manager.UpdateProfile("test-profile", "", "", "staging", false, "")
	require.NoError(t, err)
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)

	profile, exists := profiles["test-profile"]
	assert.True(t, exists, "Profile should exist")
	assert.Equal(t, "access123", profile.AccessKey, "AccessKey should remain unchanged")
	assert.Equal(t, "secret123", profile.SecretKey, "SecretKey should remain unchanged")
	assert.Equal(t, "staging", profile.Environment, "Environment should be updated")
	assert.Equal(t, "Original", profile.Description, "Description should remain unchanged")

	err = manager.UpdateProfile("test-profile", "newaccess", "", "", true, "Updated desc")
	require.NoError(t, err)

	profiles, err = manager.ListProfiles()
	require.NoError(t, err)

	profile, exists = profiles["test-profile"]
	assert.True(t, exists, "Profile should exist")
	assert.Equal(t, "newaccess", profile.AccessKey, "AccessKey should be updated")
	assert.Equal(t, "secret123", profile.SecretKey, "SecretKey should remain unchanged")
	assert.Equal(t, "staging", profile.Environment, "Environment should remain unchanged")
	assert.Equal(t, "Updated desc", profile.Description, "Description should be updated")

	err = manager.UpdateProfile("non-existent", "foo", "bar", "production", false, "")
	assert.Error(t, err)
	assert.Equal(t, ErrProfileNotFound, err)
}

func TestDeleteProfile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "")
	require.NoError(t, err)

	err = manager.UseProfile("test-profile")
	require.NoError(t, err)

	err = manager.DeleteProfile("test-profile")
	assert.Error(t, err, "Should not allow deleting active profile")

	err = manager.CreateProfile("other-profile", "access456", "secret456", "staging", "")
	require.NoError(t, err)

	err = manager.DeleteProfile("other-profile")
	require.NoError(t, err)

	profiles, err := manager.ListProfiles()
	require.NoError(t, err)
	_, exists := profiles["other-profile"]
	assert.False(t, exists, "Profile should be deleted")

	err = manager.DeleteProfile("non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrProfileNotFound, err)
}

func TestGetCurrentProfile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	_, _, err = manager.GetCurrentProfile()
	assert.Equal(t, ErrProfileNotFound, err)

	err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "")
	require.NoError(t, err)

	err = manager.UseProfile("test-profile")
	require.NoError(t, err)

	profile, name, err := manager.GetCurrentProfile()
	require.NoError(t, err)
	assert.Equal(t, "test-profile", name)
	assert.Equal(t, "access123", profile.AccessKey)
	assert.Equal(t, "secret123", profile.SecretKey)
	assert.Equal(t, "production", profile.Environment)
}

func TestGetProfile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Create test profiles
	err = manager.CreateProfile("staging", "staging-access", "staging-secret", "staging", "Staging profile")
	require.NoError(t, err)

	err = manager.CreateProfile("production", "prod-access", "prod-secret", "production", "Production profile")
	require.NoError(t, err)

	// Test: get existing profile
	profile, err := manager.GetProfile("staging")
	require.NoError(t, err)
	assert.Equal(t, "staging-access", profile.AccessKey)
	assert.Equal(t, "staging-secret", profile.SecretKey)
	assert.Equal(t, "staging", profile.Environment)

	// Test: get another existing profile
	profile, err = manager.GetProfile("production")
	require.NoError(t, err)
	assert.Equal(t, "prod-access", profile.AccessKey)

	// Test: get non-existent profile
	_, err = manager.GetProfile("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent")
	assert.Contains(t, err.Error(), "not found")
}

func TestDefaultSettings(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	val, exists := manager.GetDefault("output")
	assert.False(t, exists)
	assert.Nil(t, val)

	err = manager.SetDefault("output", "json")
	require.NoError(t, err)

	err = manager.SetDefault("no-color", true)
	require.NoError(t, err)

	val, exists = manager.GetDefault("output")
	assert.True(t, exists)
	assert.Equal(t, "json", val)

	val, exists = manager.GetDefault("no-color")
	assert.True(t, exists)
	assert.Equal(t, true, val)

	// Remove a default
	err = manager.RemoveDefault("output")
	require.NoError(t, err)

	val, exists = manager.GetDefault("output")
	assert.False(t, exists)
	assert.Nil(t, val)

	// Clear all defaults
	err = manager.ClearDefaults()
	require.NoError(t, err)

	val, exists = manager.GetDefault("no-color")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestExportConfig(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	err = manager.CreateProfile("test-profile", "access123", "secret456", "production", "Test desc")
	require.NoError(t, err)

	err = manager.UseProfile("test-profile")
	require.NoError(t, err)

	err = manager.SetDefault("output", "json")
	require.NoError(t, err)

	exported, err := manager.Export()
	require.NoError(t, err)

	assert.Equal(t, "test-profile", exported.ActiveProfile)
	assert.Equal(t, "json", exported.Defaults["output"])

	exportedProfile, exists := exported.Profiles["test-profile"]
	assert.True(t, exists)
	assert.Equal(t, "[REDACTED]", exportedProfile.AccessKey)
	assert.Equal(t, "[REDACTED]", exportedProfile.SecretKey)
	assert.Equal(t, "production", exportedProfile.Environment)
	assert.Equal(t, "Test desc", exportedProfile.Description)
}

func TestListProfiles(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	profiles, err := manager.ListProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles)

	err = manager.CreateProfile("profile1", "access1", "secret1", "production", "")
	require.NoError(t, err)

	err = manager.CreateProfile("profile2", "access2", "secret2", "staging", "")
	require.NoError(t, err)

	profiles, err = manager.ListProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 2)

	_, exists := profiles["profile1"]
	assert.True(t, exists)

	_, exists = profiles["profile2"]
	assert.True(t, exists)
}

func TestUseProfile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	err = manager.CreateProfile("profile1", "access1", "secret1", "production", "")
	require.NoError(t, err)

	err = manager.CreateProfile("profile2", "access2", "secret2", "staging", "")
	require.NoError(t, err)

	err = manager.UseProfile("profile1")
	require.NoError(t, err)
	assert.Equal(t, "profile1", manager.config.ActiveProfile)

	err = manager.UseProfile("profile2")
	require.NoError(t, err)
	assert.Equal(t, "profile2", manager.config.ActiveProfile)

	err = manager.UseProfile("non-existent")
	assert.Error(t, err)
	assert.Equal(t, ErrProfileNotFound, err)
	assert.Equal(t, "profile2", manager.config.ActiveProfile, "Active profile should remain unchanged")
}

func TestNilSafety(t *testing.T) {
	var m *ConfigManager = nil
	profiles, err := m.ListProfiles()
	assert.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestConfigPersistence(t *testing.T) {
	setupTestConfig(t)

	{
		manager, err := NewConfigManager()
		require.NoError(t, err)

		err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "")
		require.NoError(t, err)

		err = manager.UseProfile("test-profile")
		require.NoError(t, err)
	}

	{
		manager, err := NewConfigManager()
		require.NoError(t, err)

		profile, name, err := manager.GetCurrentProfile()
		require.NoError(t, err)
		assert.Equal(t, "test-profile", name)
		assert.Equal(t, "access123", profile.AccessKey)
	}
}

func TestCorruptedConfigFile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)
	err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "")
	require.NoError(t, err)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)
	corruptContent := []byte("{this is not valid json")
	err = os.WriteFile(configPath, corruptContent, 0644)
	require.NoError(t, err)

	manager, err = NewConfigManager()
	require.NoError(t, err)

	profiles, err := manager.ListProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles, "New config after recovery should have empty profiles")

	// The corrupt file must be preserved, not destroyed in place
	matches, err := filepath.Glob(configPath + ".corrupt-*")
	require.NoError(t, err)
	require.Len(t, matches, 1, "Corrupt config should be preserved with a .corrupt- suffix")

	preserved, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, corruptContent, preserved, "Preserved file should contain the original corrupt bytes")

	err = manager.CreateProfile("new-profile", "access123", "secret123", "production", "")
	require.NoError(t, err)
}

func TestCorruptedConfigFilePreservesOriginal(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)
	err = manager.CreateProfile("my-profile", "AKIATEST123", "secretXYZ456", "production", "")
	require.NoError(t, err)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	// Append garbage to the valid config so unmarshal fails but the profile data is still present
	original, err := os.ReadFile(configPath)
	require.NoError(t, err)
	corrupt := make([]byte, len(original), len(original)+16)
	copy(corrupt, original)
	corrupt = append(corrupt, []byte(" *** CORRUPT ***")...)
	err = os.WriteFile(configPath, corrupt, 0600)
	require.NoError(t, err)

	_, err = NewConfigManager()
	require.NoError(t, err)

	// Backup must exist and still contain the original profile data
	matches, err := filepath.Glob(configPath + ".corrupt-*")
	require.NoError(t, err)
	require.Len(t, matches, 1, "Expected exactly one .corrupt- backup file")

	preserved, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(preserved), "AKIATEST123", "Backup should contain original access key")
	assert.Contains(t, string(preserved), "secretXYZ456", "Backup should contain original secret key")
	assert.Contains(t, string(preserved), "my-profile", "Backup should contain original profile name")
}

func TestCorruptedConfigFile_Permissions(t *testing.T) {
	setupTestConfig(t)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	// Write corrupted config with overly permissive 0644 to simulate drift
	err = os.WriteFile(configPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	// Verify the file is indeed 0644 before recovery
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), info.Mode().Perm(), "precondition: file should be 0644 before recovery")

	_, err = NewConfigManager()
	require.NoError(t, err)

	// New default config must be 0600
	info, err = os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "config file should have 0600 permissions after corruption recovery")

	// Backup must also be 0600 regardless of the original file's permissions
	matches, err := filepath.Glob(configPath + ".corrupt-*")
	require.NoError(t, err)
	require.Len(t, matches, 1, "Expected exactly one .corrupt- backup file")
	info, err = os.Stat(matches[0])
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "backup file should have 0600 permissions")
}

func TestCorruptedConfigFile_RenameFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tempDir, err := os.MkdirTemp("", "megaport-config-test")
	require.NoError(t, err)
	defer func() {
		_ = os.Chmod(tempDir, 0700)
		os.RemoveAll(tempDir)
	}()
	t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	err = os.WriteFile(configPath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	// Make the config directory read-only so rename fails
	err = os.Chmod(tempDir, 0500)
	require.NoError(t, err)

	_, err = NewConfigManager()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be preserved")
}

func TestCorruptedConfigFile_ChmodFailure(t *testing.T) {
	setupTestConfig(t)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	err = os.WriteFile(configPath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	old := chmodFile
	defer func() { chmodFile = old }()
	chmodFile = func(_ string, _ os.FileMode) error {
		return fmt.Errorf("chmod: operation not permitted")
	}

	// chmod failure is non-fatal: the CLI should still succeed and create a fresh default.
	manager, err := NewConfigManager()
	require.NoError(t, err)
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestConfigManagerSave_ChmodTmpFileError(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	old := chmodFile
	defer func() { chmodFile = old }()
	chmodFile = func(_ string, _ os.FileMode) error {
		return fmt.Errorf("chmod: operation not permitted")
	}

	err = manager.Save()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set permissions on temp config file")
}

func TestConfigManagerSave_RenameError(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	old := renameFile
	defer func() { renameFile = old }()
	renameFile = func(_, _ string) error {
		return fmt.Errorf("rename: permission denied")
	}

	err = manager.Save()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write config file")
}

func TestConfigManagerSave_ChmodFinalError(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	old := chmodFile
	defer func() { chmodFile = old }()
	callCount := 0
	chmodFile = func(path string, mode os.FileMode) error {
		callCount++
		if callCount == 1 {
			// Let the temp file's own chmod succeed; only fail the final one.
			return os.Chmod(path, mode)
		}
		return fmt.Errorf("chmod: operation not permitted")
	}

	err = manager.Save()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set permissions on config file")
}

func TestCorruptedConfigFile_ConcurrentRecovery(t *testing.T) {
	setupTestConfig(t)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	err = os.WriteFile(configPath, []byte("not valid json"), 0600)
	require.NoError(t, err)

	// Simulate another process having renamed the corrupt file between our ReadFile
	// and our Rename: the stub moves the source file (as the other process would)
	// then returns ENOENT so our code takes the concurrent-recovery branch.
	simulatedBackup := configPath + ".corrupt-concurrent"
	old := renameFile
	defer func() { renameFile = old }()
	renameFile = func(src, _ string) error {
		_ = os.Rename(src, simulatedBackup)
		return os.ErrNotExist
	}

	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Fresh default should be created and usable
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)
	assert.Empty(t, profiles)

	// The backup created by the "other process" should still be present
	_, statErr := os.Stat(simulatedBackup)
	assert.NoError(t, statErr, "Backup from concurrent process should still exist")
}

func TestSpecialProfileNames(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	err = manager.CreateProfile("", "access123", "secret123", "production", "")
	assert.Error(t, err, "Empty profile name should be rejected")

	err = manager.CreateProfile("   ", "access123", "secret123", "production", "")
	assert.Error(t, err, "Whitespace-only profile name should be rejected")

	specialName := "test!@#$%^&*()_+-=[]{}|;':,./<>?"
	err = manager.CreateProfile(specialName, "access123", "secret123", "production", "")
	require.NoError(t, err)

	unicodeName := "उपयोगकर्ता-परीक्षण"
	err = manager.CreateProfile(unicodeName, "access123", "secret123", "production", "")
	require.NoError(t, err)

	// Verify profiles were created
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)

	_, exists := profiles[specialName]
	assert.True(t, exists, "Profile with special characters should exist")

	_, exists = profiles[unicodeName]
	assert.True(t, exists, "Profile with Unicode characters should exist")

	// Test we can use these profiles
	err = manager.UseProfile(specialName)
	require.NoError(t, err)

	err = manager.UseProfile(unicodeName)
	require.NoError(t, err)
}

func TestDuplicateProfiles(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Create initial profile
	err = manager.CreateProfile("test-profile", "access123", "secret123", "production", "Original")
	require.NoError(t, err)

	// Create profile with same name - should overwrite
	err = manager.CreateProfile("test-profile", "newaccess", "newsecret", "staging", "Updated")
	require.NoError(t, err)

	// Verify profile was overwritten
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)

	profile, exists := profiles["test-profile"]
	assert.True(t, exists, "Profile should exist")
	assert.Equal(t, "newaccess", profile.AccessKey, "Access key should be updated")
	assert.Equal(t, "newsecret", profile.SecretKey, "Secret key should be updated")
	assert.Equal(t, "staging", profile.Environment, "Environment should be updated")
	assert.Equal(t, "Updated", profile.Description, "Description should be updated")
}

func TestConfigVersionHandling(t *testing.T) {
	setupTestConfig(t)

	// Create a config file with older version
	oldConfig := ConfigFile{
		Version:       1, // Older version than current
		ActiveProfile: "",
		Profiles:      make(map[string]*Profile),
		Defaults:      make(map[string]interface{}),
	}

	oldConfig.Profiles["old-profile"] = &Profile{
		AccessKey:   "old-access",
		SecretKey:   "old-secret",
		Environment: "production",
	}

	// Write this to the config file
	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	data, err := json.MarshalIndent(oldConfig, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Now try to load the config - should handle version migration
	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Verify old profile data was retained
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)

	profile, exists := profiles["old-profile"]
	assert.True(t, exists, "Old profile should be migrated")
	assert.Equal(t, "old-access", profile.AccessKey)

	// Verify config was migrated to current version
	assert.Equal(t, ConfigVersion, manager.config.Version, "Config should be upgraded to current version")
}
func TestReadOnlyConfigFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Chmod doesn't reliably block directory writes on Windows, and error text differs")
	}
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tempDir, err := os.MkdirTemp("", "megaport-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Setenv("MEGAPORT_CONFIG_DIR", tempDir)

	// Create the config directory first
	configDir, err := GetConfigDir()
	require.NoError(t, err)

	// Create a version 1 config file (to trigger migration)
	configPath := filepath.Join(configDir, "config.json")
	oldConfig := ConfigFile{
		Version:  1, // Old version to force migration
		Profiles: make(map[string]*Profile),
		Defaults: make(map[string]interface{}),
	}
	data, err := json.MarshalIndent(oldConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Make the config directory read-only. Save() now writes atomically
	// (temp file + rename), so a read-only config file no longer blocks a
	// write; only a read-only directory does, since it prevents creating
	// the temp file.
	err = os.Chmod(configDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(configDir, 0700) //nolint:errcheck // best-effort restore so cleanup can remove tempDir

	// Try to load config - should fail due to version migration
	// attempting to save while the config directory is read-only
	_, err = NewConfigManager()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestSaveRetightensOverPermissiveFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	// Simulate a pre-existing config file left world-readable.
	require.NoError(t, os.Chmod(configPath, 0644))

	require.NoError(t, manager.CreateProfile("test-profile", "access123", "secret123", "production", ""))

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "Save should re-tighten an over-permissive config file to 0600")
}

func TestSaveLeavesNoTempFile(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)
	require.NoError(t, manager.CreateProfile("test-profile", "access123", "secret123", "production", ""))

	configPath, err := GetConfigFilePath()
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(filepath.Dir(configPath), ".megaport-config-*.tmp"))
	require.NoError(t, err)
	assert.Empty(t, matches, "Save should not leave temp files behind after a successful write")
}

func TestProfileNameCaseSensitivity(t *testing.T) {
	setupTestConfig(t)

	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Create profiles with names differing only by case
	err = manager.CreateProfile("test-profile", "access1", "secret1", "production", "")
	require.NoError(t, err)

	err = manager.CreateProfile("Test-Profile", "access2", "secret2", "staging", "")
	require.NoError(t, err)

	// Verify both profiles exist
	profiles, err := manager.ListProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 2, "Should have two distinct profiles with case-sensitive names")

	// Verify we can access each profile correctly
	profile1, exists := profiles["test-profile"]
	assert.True(t, exists)
	assert.Equal(t, "access1", profile1.AccessKey)

	profile2, exists := profiles["Test-Profile"]
	assert.True(t, exists)
	assert.Equal(t, "access2", profile2.AccessKey)

	// Verify case-sensitivity in profile selection
	err = manager.UseProfile("test-profile")
	require.NoError(t, err)
	assert.Equal(t, "test-profile", manager.config.ActiveProfile)

	err = manager.UseProfile("Test-Profile")
	require.NoError(t, err)
	assert.Equal(t, "Test-Profile", manager.config.ActiveProfile)
}

func TestVeryLongPaths(t *testing.T) {
	// Create a deeply nested temporary directory
	tempBase, err := os.MkdirTemp("", "megaport-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempBase)

	// Create a very long path that stays within OS limits but is unusually deep
	longPath := tempBase
	for i := 0; i < 15; i++ {
		longPath = filepath.Join(longPath, fmt.Sprintf("nested_dir_%d", i))
	}

	// Set as config dir
	t.Setenv("MEGAPORT_CONFIG_DIR", longPath)

	// Should still work
	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Verify we can create and use profiles
	err = manager.CreateProfile("test", "access", "secret", "production", "")
	require.NoError(t, err)

	// Verify proper path creation
	_, err = os.Stat(filepath.Join(longPath, "config.json"))
	require.NoError(t, err)
}

func TestSymlinkConfigDir(t *testing.T) {
	// Create real dir and symlink dir
	realDir, err := os.MkdirTemp("", "megaport-real")
	require.NoError(t, err)
	defer os.RemoveAll(realDir)

	symlinkDir, err := os.MkdirTemp("", "megaport-symlink")
	require.NoError(t, err)
	defer os.RemoveAll(symlinkDir)

	// Remove the symlink target
	os.RemoveAll(symlinkDir)

	// Create symlink
	err = os.Symlink(realDir, symlinkDir)
	require.NoError(t, err)

	// Use symlink as config dir
	t.Setenv("MEGAPORT_CONFIG_DIR", symlinkDir)

	// Should work with symlink
	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Verify we can create profiles
	err = manager.CreateProfile("test", "access", "secret", "production", "")
	require.NoError(t, err)

	// Config file should exist in the real directory
	_, err = os.Stat(filepath.Join(realDir, "config.json"))
	require.NoError(t, err)
}

func TestImportWithMissingFields(t *testing.T) {
	setupTestConfigEnv(t)

	// Create a minimal import file with missing fields
	minimalConfig := `{
        "version": 1,
        "activeProfile": "minimal-profile",
        "profiles": {
            "minimal-profile": {
                "accessKey": "access",
                "secretKey": "secret"
            }
        },
        "defaults": {}
    }`

	importPath := filepath.Join(os.TempDir(), "minimal-config.json")
	err := os.WriteFile(importPath, []byte(minimalConfig), 0644)
	require.NoError(t, err)
	defer os.Remove(importPath)

	// Import this minimal config
	cmd, _ := setupTestCmd()
	cmd.Flags().String("file", importPath, "")
	err = cmd.ParseFlags([]string{"--file=" + importPath})
	require.NoError(t, err)

	// Mock confirmation
	oldConfirmPrompt := utils.GetConfirmPrompt()
	utils.SetConfirmPrompt(func(message string, noColor bool) bool {
		return true
	})
	defer func() { utils.SetConfirmPrompt(oldConfirmPrompt) }()

	_, err = captureOutputFromAction(func() error {
		return ImportConfig(cmd, nil, false)
	})
	require.NoError(t, err)

	// Verify the import worked and defaulted missing fields
	manager, err := NewConfigManager()
	require.NoError(t, err)

	profile, _, err := manager.GetCurrentProfile()
	require.NoError(t, err)
	assert.Equal(t, "minimal-profile", manager.config.ActiveProfile)
	assert.Equal(t, "access", profile.AccessKey)
	assert.Equal(t, "secret", profile.SecretKey)
	assert.NotEmpty(t, profile.Environment, "Environment should have a default value")
}

func TestExportWithMaxProfiles(t *testing.T) {
	setupTestConfigEnv(t)

	// Create many profiles (testing export with large dataset)
	manager, err := NewConfigManager()
	require.NoError(t, err)

	// Create 100 profiles
	for i := 0; i < 100; i++ {
		profileName := fmt.Sprintf("profile-%d", i)
		err = manager.CreateProfile(profileName, "access", "secret", "production", "")
		require.NoError(t, err)
	}

	// Export all these profiles
	exportPath := filepath.Join(os.TempDir(), "many-profiles-export.json")
	cmd, _ := setupTestCmd()
	cmd.Flags().String("file", exportPath, "")
	err = cmd.ParseFlags([]string{"--file=" + exportPath})
	require.NoError(t, err)

	_, err = captureOutputFromAction(func() error {
		return ExportConfig(cmd, nil, false)
	})
	require.NoError(t, err)

	// Verify export worked
	_, err = os.Stat(exportPath)
	require.NoError(t, err)
	defer os.Remove(exportPath)

	// Read the export file and verify it contains all profiles
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)

	var exportedConfig ConfigFile
	err = json.Unmarshal(data, &exportedConfig)
	require.NoError(t, err)

	assert.Equal(t, 100, len(exportedConfig.Profiles), "Export should contain all 100 profiles")
}
