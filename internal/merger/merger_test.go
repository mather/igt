package merger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mather/igt/internal/template"
)

func TestExtractMarkerName(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"### igt: Go ###", "Go"},
		{"### igt: Node ###", "Node"},
		{"### igt: macOS ###", "macOS"},
		{"### igt: Python 3 ###", "Python 3"},
		{"  ### igt: Spaced ###  ", "Spaced"},
		{"### igt: AL.gitignore ###", "AL.gitignore"},
		{"### Go ###", ""}, // Old format should not match
		{"### AL ###", ""}, // This could be in template content
		{"### igt: ###", ""},
		{"## igt: Go ##", ""},
		{"Go", ""},
		{"", ""},
		{"### igt: Unclosed", ""},
		{"Unclosed ###", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := extractMarkerName(tt.line)
			if result != tt.expected {
				t.Errorf("extractMarkerName(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestFormatMarker(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Go", "### igt: Go ###"},
		{"Node", "### igt: Node ###"},
		{"Python 3", "### igt: Python 3 ###"},
		{"AL.gitignore", "### igt: AL.gitignore ###"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMarker(tt.name)
			if result != tt.expected {
				t.Errorf("formatMarker(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestParseGitignore_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	merger := NewMerger(gitignorePath)
	sections, unmanaged, err := merger.ParseGitignore()

	if err != nil {
		t.Errorf("ParseGitignore() failed on non-existent file: %v", err)
	}
	if sections != nil || unmanaged != nil {
		t.Error("ParseGitignore() should return nil for non-existent file")
	}
}

func TestParseGitignore_UnmanagedOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content := `*.log
*.tmp
/dist/
node_modules/
`
	err = os.WriteFile(gitignorePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	merger := NewMerger(gitignorePath)
	sections, unmanaged, err := merger.ParseGitignore()

	if err != nil {
		t.Fatalf("ParseGitignore() failed: %v", err)
	}

	if len(sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(sections))
	}

	expectedLines := []string{"*.log", "*.tmp", "/dist/", "node_modules/"}
	if len(unmanaged) != len(expectedLines) {
		t.Errorf("expected %d unmanaged lines, got %d", len(expectedLines), len(unmanaged))
	}
}

func TestParseGitignore_WithManagedSections(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content := `# Custom rules
*.log

### igt: Go ###
*.exe
*.test
### igt: Go ###

### igt: Node ###
node_modules/
*.log
### igt: Node ###

# More custom
/dist/
`
	err = os.WriteFile(gitignorePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	merger := NewMerger(gitignorePath)
	sections, unmanaged, err := merger.ParseGitignore()

	if err != nil {
		t.Fatalf("ParseGitignore() failed: %v", err)
	}

	if len(sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sections))
	}

	// Check Go section
	if sections[0].Name != "Go" {
		t.Errorf("section[0].Name = %q, want %q", sections[0].Name, "Go")
	}
	if !strings.Contains(sections[0].Content, "*.exe") {
		t.Error("Go section should contain *.exe")
	}

	// Check Node section
	if sections[1].Name != "Node" {
		t.Errorf("section[1].Name = %q, want %q", sections[1].Name, "Node")
	}
	if !strings.Contains(sections[1].Content, "node_modules/") {
		t.Error("Node section should contain node_modules/")
	}

	// Check unmanaged lines
	unmanagedStr := strings.Join(unmanaged, "\n")
	if !strings.Contains(unmanagedStr, "# Custom rules") {
		t.Error("unmanaged should contain '# Custom rules'")
	}
	if !strings.Contains(unmanagedStr, "/dist/") {
		t.Error("unmanaged should contain '/dist/'")
	}
}

func TestMergeTemplates_NewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	templates := []template.Template{
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe\n*.test",
			Path:     "Go.gitignore",
		},
	}

	merger := NewMerger(gitignorePath)
	err = merger.MergeTemplates(templates)
	if err != nil {
		t.Fatalf("MergeTemplates() failed: %v", err)
	}

	// Read the created file
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "### igt: Go ###") {
		t.Error("content should contain igt Go markers")
	}
	if !strings.Contains(contentStr, "*.exe") {
		t.Error("content should contain *.exe")
	}
	if !strings.Contains(contentStr, "*.test") {
		t.Error("content should contain *.test")
	}
}

func TestMergeTemplates_UpdateExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	// Create initial .gitignore with Go section
	initialContent := `# Custom
*.log

### igt: Go ###
*.exe
### igt: Go ###
`
	err = os.WriteFile(gitignorePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	// Merge with updated Go template
	templates := []template.Template{
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe\n*.test\n*.dll",
			Path:     "Go.gitignore",
		},
	}

	merger := NewMerger(gitignorePath)
	err = merger.MergeTemplates(templates)
	if err != nil {
		t.Fatalf("MergeTemplates() failed: %v", err)
	}

	// Read the updated file
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	// Check that custom content is preserved
	if !strings.Contains(contentStr, "# Custom") {
		t.Error("custom content should be preserved")
	}
	if !strings.Contains(contentStr, "*.log") {
		t.Error("*.log should be preserved")
	}

	// Check that Go section is updated
	if !strings.Contains(contentStr, "*.dll") {
		t.Error("updated content should contain *.dll")
	}
}

func TestMergeTemplates_AddMultiple(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	templates := []template.Template{
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe\n*.test",
			Path:     "Go.gitignore",
		},
		{
			Name:     "Node.gitignore",
			FileName: "Node.gitignore",
			Category: template.CategoryLanguage,
			Content:  "node_modules/\n*.log",
			Path:     "Node.gitignore",
		},
	}

	merger := NewMerger(gitignorePath)
	err = merger.MergeTemplates(templates)
	if err != nil {
		t.Fatalf("MergeTemplates() failed: %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	// Check both sections exist
	if !strings.Contains(contentStr, "### igt: Go ###") {
		t.Error("should contain igt Go section")
	}
	if !strings.Contains(contentStr, "### igt: Node ###") {
		t.Error("should contain igt Node section")
	}
	if !strings.Contains(contentStr, "node_modules/") {
		t.Error("should contain node_modules/")
	}
}

func TestMergeTemplates_Deduplication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	// Same template twice
	templates := []template.Template{
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe",
			Path:     "Go.gitignore",
		},
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe",
			Path:     "Go.gitignore",
		},
	}

	merger := NewMerger(gitignorePath)
	err = merger.MergeTemplates(templates)
	if err != nil {
		t.Fatalf("MergeTemplates() failed: %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	// Count occurrences of the marker
	count := strings.Count(string(content), "### igt: Go ###")
	if count != 2 { // Start and end marker
		t.Errorf("expected 2 occurrences of igt Go marker (start+end), got %d", count)
	}
}

func TestPreviewChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "igt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	// Create initial file
	initialContent := `# Custom
*.log
`
	err = os.WriteFile(gitignorePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	templates := []template.Template{
		{
			Name:     "Go.gitignore",
			FileName: "Go.gitignore",
			Category: template.CategoryLanguage,
			Content:  "*.exe",
			Path:     "Go.gitignore",
		},
	}

	merger := NewMerger(gitignorePath)
	preview, err := merger.PreviewChanges(templates)
	if err != nil {
		t.Fatalf("PreviewChanges() failed: %v", err)
	}

	if !strings.Contains(preview, "+ Added: Go") {
		t.Error("preview should indicate Go was added")
	}
	if !strings.Contains(preview, ".gitignore") {
		t.Error("preview should mention the file path")
	}
}
