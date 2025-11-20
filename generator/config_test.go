package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Test 1: No config file exists - should return default config
	originalWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	err := os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory to tmpDir: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config to be returned, got nil")
	}

	// Should have default values
	if cfg.FieldCase != FieldCaseCamel {
		t.Errorf("Expected default FieldCase to be 'camel', got %s", cfg.FieldCase)
	}

	// Test 2: Config file exists - should load it
	configContent := `packages:
  - ./models
output: ./schema
field_case: snake
use_json_tag: false
`
	configPath := filepath.Join(tmpDir, "gqlschemagen.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.FieldCase != FieldCaseSnake {
		t.Errorf("Expected FieldCase to be 'snake', got %s", cfg.FieldCase)
	}

	if cfg.UseJsonTag != false {
		t.Errorf("Expected UseJsonTag to be false, got %v", cfg.UseJsonTag)
	}

	// Paths should now be resolved relative to config file location
	// Check that packages were resolved (should end with /models and be absolute)
	if len(cfg.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(cfg.Packages))
	}
	if !filepath.IsAbs(cfg.Packages[0]) {
		t.Errorf("Expected absolute path, got %s", cfg.Packages[0])
	}
	if filepath.Base(cfg.Packages[0]) != "models" {
		t.Errorf("Expected package path to end with 'models', got %s", cfg.Packages[0])
	}

	// ConfigDir should be set and absolute
	if cfg.ConfigDir == "" {
		t.Error("Expected ConfigDir to be set")
	}
	if !filepath.IsAbs(cfg.ConfigDir) {
		t.Errorf("Expected ConfigDir to be absolute, got %s", cfg.ConfigDir)
	}
}

func TestFindConfig(t *testing.T) {
	// Create a nested directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "nested")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested subdirectory: %v", err)
	}

	// Create config in parent directory
	configPath := filepath.Join(tmpDir, "gqlschemagen.yml")
	if err := os.WriteFile(configPath, []byte("packages: [./]"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Change to subdirectory
	originalWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(originalWd)
	}()
	err = os.Chdir(subDir)
	if err != nil {
		t.Fatalf("Failed to change directory to subDir: %v", err)
	}

	// Should find config in parent directory
	foundPath := FindConfig()
	if foundPath == "" {
		t.Fatal("Expected to find config file in parent directory")
	}

	if filepath.Base(foundPath) != "gqlschemagen.yml" {
		t.Errorf("Expected to find gqlschemagen.yml, got %s", filepath.Base(foundPath))
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `packages:
  - ./models
  - ./entities
output: ./graphql/schema
field_case: pascal
strategy: single
use_gqlgen_directives: true
`
	configPath := filepath.Join(tmpDir, "custom.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	if cfg.FieldCase != FieldCasePascal {
		t.Errorf("Expected FieldCase to be 'pascal', got %s", cfg.FieldCase)
	}

	if cfg.GenStrategy != GenStrategySingle {
		t.Errorf("Expected GenStrategy to be 'single', got %s", cfg.GenStrategy)
	}

	if !cfg.UseGqlGenDirectives {
		t.Error("Expected UseGqlGenDirectives to be true")
	}

	if len(cfg.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(cfg.Packages))
	}
}
