package template

import "strings"

// Category represents the category of a gitignore template
type Category string

const (
	CategoryLanguage  Category = "Language"
	CategoryGlobal    Category = "Global"
	CategoryCommunity Category = "Community"
)

// Template represents a single gitignore template
type Template struct {
	Name     string   // e.g., "Go", "Node"
	FileName string   // e.g., "Go.gitignore"
	Category Category
	Content  string
	Path     string // Path in the github/gitignore repo
}

// DetermineCategory determines the category based on the file path
func DetermineCategory(path string) Category {
	if strings.HasPrefix(path, "Global/") {
		return CategoryGlobal
	}
	if strings.HasPrefix(path, "community/") {
		return CategoryCommunity
	}
	return CategoryLanguage
}

// GetDisplayName returns the template name without the .gitignore extension
func (t *Template) GetDisplayName() string {
	return strings.TrimSuffix(t.Name, ".gitignore")
}

// CategoryOrder returns the sort order for categories
func CategoryOrder(c Category) int {
	switch c {
	case CategoryLanguage:
		return 0
	case CategoryGlobal:
		return 1
	case CategoryCommunity:
		return 2
	default:
		return 3
	}
}
