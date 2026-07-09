# igt - Interactive gitignore CLI

`igt` (**i**gnore + **T**UI) is an interactive CLI tool for managing `.gitignore` templates from the official [github/gitignore](https://github.com/github/gitignore) repository.

## Features

- 🎨 **Interactive TUI** - Beautiful terminal UI with fuzzy search and multi-select
- 📦 **Smart Caching** - Local cache with configurable TTL (default 7 days)
- 🔄 **Section Management** - Update templates without losing custom rules
- 🌐 **GitHub Integration** - Fetch templates directly from github/gitignore
- 🚀 **Fast & Lightweight** - Single binary with no dependencies
- 🔍 **Fuzzy Search** - Quickly find templates with incremental filtering
- 📋 **Preview Mode** - Dry-run to see changes before applying

## Installation

```bash
go install github.com/mather/igt/cmd/igt@latest
```

Or build from source:

```bash
git clone https://github.com/mather/igt.git
cd igt
go build -o igt ./cmd/igt
```

## Usage

### Interactive Mode

Simply run `igt` to launch the interactive template selector:

```bash
igt
```

Use the arrow keys to navigate, `space` or `tab` to toggle selection, `enter` to confirm, and `esc` to cancel.

### Non-Interactive Mode

Pass template names as arguments to skip the TUI:

```bash
igt Go Node Python
```

### List Available Templates

Show all available templates grouped by category:

```bash
igt --list
```

Search for specific templates:

```bash
igt --list | grep -i python
```

### Preview Changes

Use dry-run mode to preview what would be added:

```bash
igt --dry-run Go Node
```

### Refresh Cache

Force refresh the template cache from GitHub:

```bash
igt --refresh
```

### Custom Output Path

Specify a custom output file:

```bash
igt --output path/to/.gitignore Go
```

## How It Works

### Template Sections

`igt` manages templates using marked sections in your `.gitignore`:

```gitignore
# Your custom rules
*.log
/dist/

### igt: Go ###
*.exe
*.test
### igt: Go ###

### igt: Node ###
node_modules/
npm-debug.log
### igt: Node ###
```

When you re-run `igt` and select templates:
- Existing managed sections (between `### igt: Name ###` markers) are updated
- Your custom rules (outside markers) are preserved
- New templates are added at the end
- The `igt:` prefix prevents conflicts with template content

### Caching

Templates are cached locally in `~/.cache/igt/templates/` with a 7-day TTL:
- First run downloads the entire repository as a zip archive (one HTTP request)
- Subsequent runs use cached templates
- Cache automatically refreshes after TTL expires
- Use `--refresh` to force immediate update

### Offline Support

Since `igt` downloads the entire repository once and caches it locally:
- No rate limiting issues
- Works offline after initial download
- Fast subsequent operations

## Command-Line Options

```
Usage:
  igt [flags] [<template>...]

Flags:
  -o, --output string   Output file path (default: .gitignore)
  -r, --refresh         Force refresh cache
  -l, --list            List all templates
  -n, --dry-run         Dry run mode (preview changes)
  -h, --help            Show help
  -v, --version         Show version

Examples:
  igt                   # Interactive mode
  igt Go Node           # Non-interactive mode
  igt -l | grep -i go   # List and search templates
  igt -n Go             # Preview changes
  igt -r                # Refresh cache and select
```

## Template Categories

Templates are organized into three categories:

- **Language** - Programming language-specific templates (e.g., Go, Python, Java)
- **Global** - OS and editor-specific templates (e.g., macOS, Windows, Vim)
- **Community** - Community-contributed templates

## Development

### Project Structure

```
igt/
├── cmd/igt/           # CLI entry point
├── internal/
│   ├── fetcher/       # GitHub API & caching
│   ├── merger/        # .gitignore parser & merger
│   ├── template/      # Data models
│   └── ui/            # Bubble Tea TUI
├── go.mod
└── README.md
```

### Running Tests

```bash
go test ./internal/... -v
```

### Building

```bash
go build -o igt ./cmd/igt
```

## Similar Tools

- [gibo](https://github.com/simonwhitaker/gibo) - Command-line gitignore template manager
- [gi](https://github.com/edouard-lopez/gi) - CLI for generating .gitignore files

`igt` differentiates itself with:
- Interactive TUI with fuzzy search
- Managed sections that preserve custom rules
- Built-in caching with TTL

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
