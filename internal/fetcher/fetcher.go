package fetcher

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mather/igt/internal/template"
)

const (
	defaultCacheTTL = 7 * 24 * time.Hour
	repoOwner       = "github"
	repoName        = "gitignore"
	zipURL          = "https://github.com/github/gitignore/archive/refs/heads/main.zip"
)

// Fetcher defines the interface for fetching gitignore templates
type Fetcher interface {
	FetchTemplates(ctx context.Context, forceRefresh bool) ([]template.Template, error)
}

// GitHubFetcher implements Fetcher for GitHub API
type GitHubFetcher struct {
	cacheDir  string
	cacheTTL  time.Duration
	token     string
	httpClient *http.Client
}

// NewGitHubFetcher creates a new GitHub fetcher
func NewGitHubFetcher() (*GitHubFetcher, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	cacheDir = filepath.Join(cacheDir, "igt", "templates")

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &GitHubFetcher{
		cacheDir:   cacheDir,
		cacheTTL:   defaultCacheTTL,
		token:      os.Getenv("GITHUB_TOKEN"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}


// FetchTemplates fetches all gitignore templates from GitHub or cache
func (f *GitHubFetcher) FetchTemplates(ctx context.Context, forceRefresh bool) ([]template.Template, error) {
	// Check cache first unless force refresh is requested
	if !forceRefresh {
		if templates, err := f.loadFromCache(); err == nil {
			return templates, nil
		}
	}

	// Fetch from GitHub
	templates, err := f.fetchFromGitHub(ctx)
	if err != nil {
		// Try to fall back to cache on error
		if cached, cacheErr := f.loadFromCache(); cacheErr == nil {
			fmt.Fprintf(os.Stderr, "Warning: GitHub API error, using cached templates: %v\n", err)
			return cached, nil
		}
		return nil, fmt.Errorf("failed to fetch from GitHub and no cache available: %w", err)
	}

	// Save to cache
	if err := f.saveToCache(templates); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
	}

	return templates, nil
}

func (f *GitHubFetcher) fetchFromGitHub(ctx context.Context) ([]template.Template, error) {
	// Download the zip archive
	req, err := http.NewRequestWithContext(ctx, "GET", zipURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download zip: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download zip: status %d", resp.StatusCode)
	}

	// Read the entire zip file into memory
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip data: %w", err)
	}

	// Open the zip archive
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	// Extract .gitignore templates
	var templates []template.Template
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Remove the leading directory name (e.g., "gitignore-main/")
		path := strings.SplitN(file.Name, "/", 2)
		if len(path) < 2 {
			continue
		}
		relativePath := path[1]

		// Only process .gitignore files
		if !strings.HasSuffix(relativePath, ".gitignore") {
			continue
		}

		// Filter: root level, Global/, or community/
		if !isValidTemplatePath(relativePath) {
			continue
		}

		// Read file content
		rc, err := file.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open %s: %v\n", relativePath, err)
			continue
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", relativePath, err)
			continue
		}

		name := filepath.Base(relativePath)
		templates = append(templates, template.Template{
			Name:     name,
			FileName: name,
			Category: template.DetermineCategory(relativePath),
			Content:  string(content),
			Path:     relativePath,
		})
	}

	if len(templates) == 0 {
		return nil, fmt.Errorf("no templates found in zip archive")
	}

	return templates, nil
}

func isValidTemplatePath(path string) bool {
	// Root level .gitignore files
	if !strings.Contains(path, "/") {
		return true
	}
	// Global/ directory
	if strings.HasPrefix(path, "Global/") {
		return true
	}
	// community/ directory
	if strings.HasPrefix(path, "community/") {
		return true
	}
	return false
}

func (f *GitHubFetcher) loadFromCache() ([]template.Template, error) {
	cacheFile := filepath.Join(f.cacheDir, "templates.json")
	metaFile := filepath.Join(f.cacheDir, "meta.json")

	// Check if cache is expired
	meta, err := os.Stat(metaFile)
	if err != nil {
		return nil, fmt.Errorf("cache meta not found: %w", err)
	}

	if time.Since(meta.ModTime()) > f.cacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	// Load templates
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var templates []template.Template
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, fmt.Errorf("failed to decode cache: %w", err)
	}

	return templates, nil
}

func (f *GitHubFetcher) saveToCache(templates []template.Template) error {
	cacheFile := filepath.Join(f.cacheDir, "templates.json")
	metaFile := filepath.Join(f.cacheDir, "meta.json")

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode templates: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	// Write meta file with current timestamp
	metaData := []byte(fmt.Sprintf(`{"updated_at":"%s"}`, time.Now().Format(time.RFC3339)))
	if err := os.WriteFile(metaFile, metaData, 0644); err != nil {
		return fmt.Errorf("failed to write meta: %w", err)
	}

	return nil
}

