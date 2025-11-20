package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pablor21/gqlschemagen/generator"
)

// Watcher manages file system watching and regeneration
type Watcher struct {
	config        *generator.Config
	watcher       *fsnotify.Watcher
	debounceTimer *time.Timer
	debounceDelay time.Duration
}

// StartWatch initializes the watcher and monitors for changes
func StartWatch(cfg *generator.Config) error {
	// Get debounce delay from config or use default
	debounceMs := cfg.CLI.Watcher.DebounceMs
	if debounceMs <= 0 {
		debounceMs = 500
	}

	w := &Watcher{
		config:        cfg,
		debounceDelay: time.Duration(debounceMs) * time.Millisecond,
	}

	// Run initial generation
	fmt.Println("Running initial generation...")
	if err := generator.Generate(cfg); err != nil {
		log.Printf("Initial generation failed: %v", err)
	} else {
		fmt.Println("✓ Initial generation complete")
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()
	w.watcher = watcher

	// Add all package directories to watch
	for _, pkg := range cfg.Packages {
		if err := w.addRecursive(pkg); err != nil {
			return fmt.Errorf("failed to watch %s: %w", pkg, err)
		}
	}

	// Add additional paths from config
	for _, path := range cfg.CLI.Watcher.AdditionalPaths {
		if err := w.addRecursive(path); err != nil {
			log.Printf("Warning: failed to watch additional path %s: %v", path, err)
		}
	}

	fmt.Println("\n👀 Watching for changes... (Press Ctrl+C to stop)")

	// Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Watch for events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only watch Go files
			if !strings.HasSuffix(event.Name, ".go") {
				continue
			}

			// Ignore generated schema files
			if strings.Contains(event.Name, cfg.Output) {
				continue
			}

			// React to file changes
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				w.scheduleRegeneration(event.Name)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)

		case <-sigChan:
			fmt.Println("\n\nStopping watcher...")
			return nil
		}
	}
}

// scheduleRegeneration debounces file changes and triggers regeneration
func (w *Watcher) scheduleRegeneration(changedFile string) {
	// Reset timer if it exists
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	// Schedule regeneration after debounce delay
	w.debounceTimer = time.AfterFunc(w.debounceDelay, func() {
		w.regenerate(changedFile)
	})
}

// regenerate runs the schema generation and reports results
func (w *Watcher) regenerate(changedFile string) {
	// Clear console for clean output
	fmt.Print("\033[H\033[2J")

	// Show timestamp and changed file
	timestamp := time.Now().Format("15:04:05")
	relPath, _ := filepath.Rel(w.config.ConfigDir, changedFile)
	if relPath == "" {
		relPath = changedFile
	}
	fmt.Printf("[%s] 🔄 Change detected: %s\n", timestamp, relPath)
	fmt.Println("Regenerating schema...")

	// Run generation
	err := generator.Generate(w.config)
	if err != nil {
		fmt.Printf("\n❌ Generation failed: %v\n", err)
	} else {
		fmt.Printf("\n✓ Generation complete at %s\n", timestamp)
	}

	fmt.Println("\n👀 Watching for changes... (Press Ctrl+C to stop)")
}

// addRecursive adds a directory and all its subdirectories to the watcher
func (w *Watcher) addRecursive(path string) error {
	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	return filepath.Walk(absPath, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only watch directories
		if !info.IsDir() {
			return nil
		}

		// Skip certain directories
		if !w.shouldWatch(walkPath) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := w.watcher.Add(walkPath); err != nil {
			return err
		}

		return nil
	})
}

// shouldWatch determines if a directory should be watched
func (w *Watcher) shouldWatch(path string) bool {
	base := filepath.Base(path)

	// Skip hidden directories
	if strings.HasPrefix(base, ".") {
		return false
	}

	// Check against ignore patterns from config
	for _, pattern := range w.config.CLI.Watcher.IgnorePatterns {
		if base == pattern {
			return false
		}
	}

	// Skip the output directory to avoid watching generated files
	if w.config.Output != "" {
		absOutput, _ := filepath.Abs(w.config.Output)
		if strings.HasPrefix(path, absOutput) {
			return false
		}
	}

	return true
}
