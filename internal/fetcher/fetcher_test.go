package fetcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mather/igt/internal/template"
)

func TestNewGitHubFetcher(t *testing.T) {
	fetcher, err := NewGitHubFetcher()
	if err != nil {
		t.Fatalf("NewGitHubFetcher() failed: %v", err)
	}

	if fetcher.cacheDir == "" {
		t.Error("cacheDir should not be empty")
	}

	if fetcher.cacheTTL != defaultCacheTTL {
		t.Errorf("cacheTTL = %v, want %v", fetcher.cacheTTL, defaultCacheTTL)
	}

	if fetcher.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestIsValidTemplatePath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"Go.gitignore", true},
		{"Node.gitignore", true},
		{"Global/macOS.gitignore", true},
		{"Global/Windows.gitignore", true},
		{"community/Python.gitignore", true},
		{"community/subfolder/File.gitignore", true},
		{"docs/README.md", false},
		{"subfolder/nested/File.gitignore", false},
		{"Android.gitignore", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isValidTemplatePath(tt.path)
			if result != tt.valid {
				t.Errorf("isValidTemplatePath(%q) = %v, want %v", tt.path, result, tt.valid)
			}
		})
	}
}



func TestCacheOperations(t *testing.T) {
	// Create a temporary cache directory
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fetcher := &GitHubFetcher{
		cacheDir: tmpDir,
		cacheTTL: 1 * time.Hour,
	}

	// Test data
	testTemplates := []template.Template{
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe\n*.test\n",
			Path:     "Go.gitignore",
		},
		{
			Name:     "Node.gitignore",
			FileName: "Node.gitignore",
			Category: template.CategoryLanguage,
			Content:  "node_modules/\n*.log\n",
			Path:     "Node.gitignore",
		},
	}

	// Test saveToCache
	err = fetcher.saveToCache(testTemplates)
	if err != nil {
		t.Fatalf("saveToCache() failed: %v", err)
	}

	// Verify files were created
	cacheFile := filepath.Join(tmpDir, "templates.json")
	metaFile := filepath.Join(tmpDir, "meta.json")

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Error("cache file was not created")
	}
	if _, err := os.Stat(metaFile); os.IsNotExist(err) {
		t.Error("meta file was not created")
	}

	// Test loadFromCache
	loaded, err := fetcher.loadFromCache()
	if err != nil {
		t.Fatalf("loadFromCache() failed: %v", err)
	}

	if len(loaded) != len(testTemplates) {
		t.Errorf("loaded %d templates, want %d", len(loaded), len(testTemplates))
	}

	for i, tmpl := range loaded {
		if tmpl.Name != testTemplates[i].Name {
			t.Errorf("template[%d].Name = %q, want %q", i, tmpl.Name, testTemplates[i].Name)
		}
		if tmpl.Content != testTemplates[i].Content {
			t.Errorf("template[%d].Content = %q, want %q", i, tmpl.Content, testTemplates[i].Content)
		}
	}
}

func TestCacheExpiration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fetcher := &GitHubFetcher{
		cacheDir: tmpDir,
		cacheTTL: 1 * time.Millisecond, // Very short TTL for testing
	}

	testTemplates := []template.Template{
		{Name: "Test.gitignore", Content: "test"},
	}

	// Save to cache
	err = fetcher.saveToCache(testTemplates)
	if err != nil {
		t.Fatalf("saveToCache() failed: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	// Try to load - should fail due to expiration
	_, err = fetcher.loadFromCache()
	if err == nil {
		t.Error("loadFromCache() should fail on expired cache")
	}
}

