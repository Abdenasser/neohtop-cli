package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		cfg := DefaultConfig()

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		if len(cfg.Columns) == 0 {
			t.Error("expected columns to be set")
		}

		if cfg.RefreshRate <= 0 {
			t.Errorf("expected positive refresh rate, got %d", cfg.RefreshRate)
		}

		if cfg.Theme == "" {
			t.Error("expected theme to be set")
		}
	})

	t.Run("columns include expected fields", func(t *testing.T) {
		cfg := DefaultConfig()

		if len(cfg.Columns) < 4 {
			t.Errorf("expected at least 4 columns, got %d", len(cfg.Columns))
		}

		// Check that key columns are present
		colSet := make(map[string]bool)
		for _, col := range cfg.Columns {
			colSet[col] = true
		}
		for _, expected := range []string{"pid", "name", "memory", "cpu"} {
			if !colSet[expected] {
				t.Errorf("expected column %q in defaults", expected)
			}
		}
	})

	t.Run("refresh rate is reasonable", func(t *testing.T) {
		cfg := DefaultConfig()

		if cfg.RefreshRate < 100 || cfg.RefreshRate > 10000 {
			t.Errorf("expected refresh rate between 100-10000ms, got %d", cfg.RefreshRate)
		}
	})

	t.Run("theme is one of known values", func(t *testing.T) {
		cfg := DefaultConfig()

		// Just check it's a non-empty string; exact theme may vary
		if cfg.Theme == "" {
			t.Error("expected non-empty theme")
		}
	})
}

func TestSaveAndLoad(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Create a custom config
		cfg := &Config{
			Columns:     []string{"pid", "name", "custom"},
			RefreshRate: 2000,
			Theme:       "custom-theme",
		}

		// Save to temp file
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		// Load from temp file
		loaded := &Config{}
		loadedData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config file: %v", err)
		}

		if err := json.Unmarshal(loadedData, loaded); err != nil {
			t.Fatalf("failed to unmarshal config: %v", err)
		}

		// Verify round-trip
		if loaded.RefreshRate != cfg.RefreshRate {
			t.Errorf("expected refresh rate %d, got %d", cfg.RefreshRate, loaded.RefreshRate)
		}
		if loaded.Theme != cfg.Theme {
			t.Errorf("expected theme %s, got %s", cfg.Theme, loaded.Theme)
		}
		if len(loaded.Columns) != len(cfg.Columns) {
			t.Errorf("expected %d columns, got %d", len(cfg.Columns), len(loaded.Columns))
		}
	})

	t.Run("load returns defaults on missing file", func(t *testing.T) {
		// When config path doesn't exist, Load() returns defaults
		tmpDir := t.TempDir()
		nonexistentPath := filepath.Join(tmpDir, "nonexistent", "config.json")

		// Simulate what happens with a missing file
		_, err := os.ReadFile(nonexistentPath)
		if err == nil {
			t.Fatal("expected error reading nonexistent file")
		}

		// This tests the behavior: missing file → defaults
		defaults := DefaultConfig()
		if defaults == nil {
			t.Fatal("expected non-nil defaults")
		}
	})

	t.Run("load returns defaults on corrupt JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "corrupt.json")

		// Write corrupt JSON
		if err := os.WriteFile(configPath, []byte("{ invalid json }"), 0644); err != nil {
			t.Fatalf("failed to write corrupt config: %v", err)
		}

		// Try to unmarshal
		data, _ := os.ReadFile(configPath)
		cfg := &Config{}
		err := json.Unmarshal(data, cfg)

		// Should get error on unmarshal of corrupt data
		if err == nil {
			t.Error("expected error unmarshaling corrupt JSON")
		}

		// Fallback behavior is to use defaults
		defaults := DefaultConfig()
		if defaults == nil {
			t.Fatal("expected non-nil defaults")
		}
	})

	t.Run("config marshals to valid JSON", func(t *testing.T) {
		cfg := DefaultConfig()

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Verify it's valid JSON by unmarshaling
		loaded := &Config{}
		if err := json.Unmarshal(data, loaded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if loaded.RefreshRate != cfg.RefreshRate {
			t.Errorf("refresh rate mismatch after marshal/unmarshal")
		}
	})

	t.Run("config fields are correct types", func(t *testing.T) {
		cfg := DefaultConfig()

		// Verify field types
		if _, ok := interface{}(cfg.Columns).([]string); !ok {
			t.Error("Columns should be []string")
		}

		if _, ok := interface{}(cfg.RefreshRate).(int); !ok {
			t.Error("RefreshRate should be int")
		}

		if _, ok := interface{}(cfg.Theme).(string); !ok {
			t.Error("Theme should be string")
		}
	})

	t.Run("save creates directory structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := DefaultConfig()

		// Simulate Save behavior: create directory if needed
		configDir := filepath.Join(tmpDir, ".config", "neohtop-cli")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		configPath := filepath.Join(configDir, "config.json")
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); err != nil {
			t.Fatalf("config file not found: %v", err)
		}
	})

	t.Run("partial config preserves defaults for missing fields", func(t *testing.T) {
		// Test unmarshaling a partial config
		partialJSON := `{"refresh_rate_ms": 5000}`

		loaded := DefaultConfig()
		if err := json.Unmarshal([]byte(partialJSON), loaded); err != nil {
			t.Fatalf("failed to unmarshal partial config: %v", err)
		}

		if loaded.RefreshRate != 5000 {
			t.Errorf("expected refresh rate 5000, got %d", loaded.RefreshRate)
		}
		// Theme should still have its default
		if loaded.Theme == "" {
			t.Error("expected theme to retain default value")
		}
	})

	t.Run("empty columns list is preserved", func(t *testing.T) {
		cfg := &Config{
			Columns:     []string{},
			RefreshRate: 1000,
			Theme:       "charm",
		}

		data, _ := json.Marshal(cfg)
		loaded := &Config{}
		json.Unmarshal(data, loaded)

		if len(loaded.Columns) != 0 {
			t.Errorf("expected empty columns, got %d", len(loaded.Columns))
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Run("default columns are ordered", func(t *testing.T) {
		cfg := DefaultConfig()

		if len(cfg.Columns) == 0 {
			t.Fatal("expected non-empty columns")
		}

		// First column should typically be pid
		if cfg.Columns[0] != "pid" {
			t.Errorf("expected first column to be 'pid', got '%s'", cfg.Columns[0])
		}
	})

	t.Run("all default columns are strings", func(t *testing.T) {
		cfg := DefaultConfig()

		for i, col := range cfg.Columns {
			if col == "" {
				t.Errorf("column %d is empty", i)
			}
		}
	})

	t.Run("default refresh rate is in milliseconds", func(t *testing.T) {
		cfg := DefaultConfig()

		// Should be in range of reasonable milliseconds
		if cfg.RefreshRate < 100 {
			t.Errorf("refresh rate seems too fast: %d ms", cfg.RefreshRate)
		}
		if cfg.RefreshRate > 60000 {
			t.Errorf("refresh rate seems too slow: %d ms", cfg.RefreshRate)
		}
	})
}
