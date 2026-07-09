package merger

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mather/igt/internal/template"
)

const markerFormat = "### igt: %s ###"

// Section represents a managed section in .gitignore
type Section struct {
	Name       string
	StartLine  int
	EndLine    int
	Content    string
	IsManaged  bool
}

// Merger handles merging templates into .gitignore
type Merger struct {
	gitignorePath string
}

// NewMerger creates a new Merger
func NewMerger(gitignorePath string) *Merger {
	return &Merger{
		gitignorePath: gitignorePath,
	}
}

// ParseGitignore parses the .gitignore file and extracts managed sections
func (m *Merger) ParseGitignore() ([]Section, []string, error) {
	file, err := os.Open(m.gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil // File doesn't exist yet
		}
		return nil, nil, fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	var sections []Section
	var unmanagedLines []string
	var currentSection *Section
	var lineNum int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Check if this is a marker line
		if name := extractMarkerName(line); name != "" {
			if currentSection == nil {
				// Start of a new section
				currentSection = &Section{
					Name:      name,
					StartLine: lineNum,
					IsManaged: true,
				}
			} else if currentSection.Name == name {
				// End of the current section
				currentSection.EndLine = lineNum
				sections = append(sections, *currentSection)
				currentSection = nil
			} else {
				// Mismatched marker - treat as unmanaged
				if currentSection != nil {
					// This shouldn't happen in well-formed files
					unmanagedLines = append(unmanagedLines, formatMarker(currentSection.Name))
					unmanagedLines = append(unmanagedLines, strings.Split(currentSection.Content, "\n")...)
					currentSection = nil
				}
				unmanagedLines = append(unmanagedLines, line)
			}
		} else if currentSection != nil {
			// Inside a managed section
			if currentSection.Content != "" {
				currentSection.Content += "\n"
			}
			currentSection.Content += line
		} else {
			// Unmanaged line
			unmanagedLines = append(unmanagedLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// Handle unclosed section
	if currentSection != nil {
		unmanagedLines = append(unmanagedLines, formatMarker(currentSection.Name))
		if currentSection.Content != "" {
			unmanagedLines = append(unmanagedLines, strings.Split(currentSection.Content, "\n")...)
		}
	}

	return sections, unmanagedLines, nil
}

// MergeTemplates merges selected templates into .gitignore
func (m *Merger) MergeTemplates(templates []template.Template) error {
	// Parse existing .gitignore
	existingSections, unmanagedLines, err := m.ParseGitignore()
	if err != nil {
		return err
	}

	// Build a map of existing sections for quick lookup
	existingMap := make(map[string]Section)
	for _, section := range existingSections {
		existingMap[section.Name] = section
	}

	// Deduplicate templates
	templateMap := make(map[string]template.Template)
	for _, tmpl := range templates {
		name := tmpl.GetDisplayName()
		templateMap[name] = tmpl
	}

	// Build the new .gitignore content
	var lines []string

	// Add unmanaged content
	for _, line := range unmanagedLines {
		lines = append(lines, line)
	}

	// Trim trailing empty lines from unmanaged content
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// Add new/updated sections
	for _, tmpl := range templateMap {
		name := tmpl.GetDisplayName()

		// Add spacing before new section if there's existing content
		if len(lines) > 0 {
			lines = append(lines, "")
		}

		// Add section with markers
		lines = append(lines, formatMarker(name))
		content := strings.TrimSpace(tmpl.Content)
		if content != "" {
			lines = append(lines, content)
		}
		lines = append(lines, formatMarker(name))
	}

	// Write to file
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n" // Ensure file ends with newline
	}

	if err := os.WriteFile(m.gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

// PreviewChanges returns a diff-like preview of what would change
func (m *Merger) PreviewChanges(templates []template.Template) (string, error) {
	existingSections, unmanagedLines, err := m.ParseGitignore()
	if err != nil {
		return "", err
	}

	existingMap := make(map[string]Section)
	for _, section := range existingSections {
		existingMap[section.Name] = section
	}

	templateMap := make(map[string]template.Template)
	for _, tmpl := range templates {
		name := tmpl.GetDisplayName()
		templateMap[name] = tmpl
	}

	var preview strings.Builder
	preview.WriteString("Changes to ")
	preview.WriteString(m.gitignorePath)
	preview.WriteString(":\n\n")

	// Show sections that will be removed/updated
	for name := range existingMap {
		if _, exists := templateMap[name]; exists {
			preview.WriteString(fmt.Sprintf("~ Updated: %s\n", name))
		} else {
			// Section exists but not selected - will be preserved
		}
	}

	// Show new sections
	for name := range templateMap {
		if _, exists := existingMap[name]; !exists {
			preview.WriteString(fmt.Sprintf("+ Added: %s\n", name))
		}
	}

	if preview.Len() == 0 {
		preview.WriteString("No changes\n")
	}

	// Show detailed diff
	preview.WriteString("\n--- Preview ---\n")

	// Unmanaged content (unchanged)
	for _, line := range unmanagedLines {
		if line != "" {
			preview.WriteString("  " + line + "\n")
		}
	}

	// New sections
	for _, tmpl := range templateMap {
		name := tmpl.GetDisplayName()
		if section, exists := existingMap[name]; exists {
			// Show removal of old section
			preview.WriteString("- " + formatMarker(name) + "\n")
			for _, line := range strings.Split(section.Content, "\n") {
				if line != "" {
					preview.WriteString("- " + line + "\n")
				}
			}
			preview.WriteString("- " + formatMarker(name) + "\n")
		}

		// Show addition of new section
		preview.WriteString("+ " + formatMarker(name) + "\n")
		content := strings.TrimSpace(tmpl.Content)
		for _, line := range strings.Split(content, "\n") {
			if line != "" {
				preview.WriteString("+ " + line + "\n")
			}
		}
		preview.WriteString("+ " + formatMarker(name) + "\n")
	}

	return preview.String(), nil
}

func formatMarker(name string) string {
	return fmt.Sprintf(markerFormat, name)
}

func extractMarkerName(line string) string {
	line = strings.TrimSpace(line)

	// Check for igt marker format: "### igt: <name> ###"
	if !strings.HasPrefix(line, "### igt:") || !strings.HasSuffix(line, "###") {
		return ""
	}

	// Extract name between "### igt:" and "###"
	// Remove "### igt:" prefix and "###" suffix
	withoutPrefix := strings.TrimPrefix(line, "### igt:")
	withoutSuffix := strings.TrimSuffix(withoutPrefix, "###")
	return strings.TrimSpace(withoutSuffix)
}
