package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRelativePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file in a subdirectory
	configDir := filepath.Join(tmpDir, "config")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `packages:
  - ./models
  - ./entities
output: ./schema
`
	configPath := filepath.Join(configDir, "gqlschemagen.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config from file
	cfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	// ConfigDir should be set to the directory containing the config
	if cfg.ConfigDir != configDir {
		t.Errorf("Expected ConfigDir to be %s, got %s", configDir, cfg.ConfigDir)
	}

	// Paths should be resolved relative to config directory
	expectedModels := filepath.Join(configDir, "models")
	if cfg.Packages[0] != expectedModels {
		t.Errorf("Expected packages[0] to be %s, got %s", expectedModels, cfg.Packages[0])
	}

	expectedEntities := filepath.Join(configDir, "entities")
	if cfg.Packages[1] != expectedEntities {
		t.Errorf("Expected packages[1] to be %s, got %s", expectedEntities, cfg.Packages[1])
	}

	expectedOutput := filepath.Join(configDir, "schema")
	if cfg.Output != expectedOutput {
		t.Errorf("Expected output to be %s, got %s", expectedOutput, cfg.Output)
	}
}

func TestResolveRelativePathsWithAbsolute(t *testing.T) {
	tmpDir := t.TempDir()

	configDir := filepath.Join(tmpDir, "config")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	absPath := filepath.Join(tmpDir, "absolute", "models")

	configContent := `packages:
  - ./relative/models
  - ` + absPath + `
output: ` + tmpDir + `/output
`
	configPath := filepath.Join(configDir, "gqlschemagen.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	// Relative path should be resolved relative to config dir
	expectedRelative := filepath.Join(configDir, "relative", "models")
	if cfg.Packages[0] != expectedRelative {
		t.Errorf("Expected packages[0] to be %s, got %s", expectedRelative, cfg.Packages[0])
	}

	// Absolute path should remain unchanged
	if cfg.Packages[1] != absPath {
		t.Errorf("Expected packages[1] to be %s (unchanged), got %s", absPath, cfg.Packages[1])
	}

	// Absolute output should remain unchanged
	expectedOutput := filepath.Join(tmpDir, "output")
	if cfg.Output != expectedOutput {
		t.Errorf("Expected output to be %s (unchanged), got %s", expectedOutput, cfg.Output)
	}
}

func TestLoadConfigFromDifferentDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   project/
	//     gqlschemagen.yml (with relative paths)
	//   workdir/ (where we run from)

	projectDir := filepath.Join(tmpDir, "project")
	workDir := filepath.Join(tmpDir, "workdir")
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}
	err = os.MkdirAll(workDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}

	configContent := `packages:
  - ./models
output: ./schema
`
	configPath := filepath.Join(projectDir, "gqlschemagen.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Change to work directory (different from config location)
	originalWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(originalWd)
	}()
	err = os.Chdir(workDir)
	if err != nil {
		t.Fatalf("Failed to change directory to workDir: %v", err)
	}

	// Load config from project directory
	cfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromFile failed: %v", err)
	}

	// Paths should be relative to config file location, not current directory
	expectedModels := filepath.Join(projectDir, "models")
	if cfg.Packages[0] != expectedModels {
		t.Errorf("Expected packages[0] to be %s, got %s", expectedModels, cfg.Packages[0])
	}

	expectedOutput := filepath.Join(projectDir, "schema")
	if cfg.Output != expectedOutput {
		t.Errorf("Expected output to be %s, got %s", expectedOutput, cfg.Output)
	}
}
