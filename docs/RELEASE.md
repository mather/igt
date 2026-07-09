# Release Guide

This document describes how to create a new release for `igt`.

## Prerequisites

1. **Create Homebrew Tap Repository**

   Create a new GitHub repository named `homebrew-igt` at https://github.com/mather/homebrew-igt

   This repository will be automatically populated by GoReleaser when you create a release.

2. **Set up GitHub Token for Homebrew Tap**

   GoReleaser needs a GitHub token with `repo` scope to push to the Homebrew tap repository.

   a. Generate a Personal Access Token at https://github.com/settings/tokens
      - Click "Generate new token (classic)"
      - Give it a name like "GoReleaser Homebrew Tap"
      - Select scope: `repo` (Full control of private repositories)
      - Click "Generate token"
      - Copy the token immediately (you won't see it again)

   b. Add the token as a repository secret:
      - Go to https://github.com/mather/igt/settings/secrets/actions
      - Click "New repository secret"
      - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
      - Value: Paste the token from step (a)
      - Click "Add secret"

## Creating a Release

1. **Ensure all changes are committed and pushed**

   ```bash
   git status  # Should show clean working directory
   ```

2. **Create and push a version tag**

   ```bash
   # For version 0.1.0
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

3. **GitHub Actions will automatically:**

   - Run tests
   - Build binaries for multiple platforms (Linux, macOS, Windows × amd64, arm64)
   - Create GitHub Release with binaries and checksums
   - Update the Homebrew tap repository with the new formula

4. **Verify the release**

   a. Check GitHub Release: https://github.com/mather/igt/releases

   b. Check Homebrew tap: https://github.com/mather/homebrew-igt

   c. Test installation:
      ```bash
      brew tap mather/igt
      brew install igt
      igt --version
      ```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v1.0.0, v2.0.0): Incompatible API changes
- **MINOR** version (v0.1.0, v0.2.0): New features, backwards compatible
- **PATCH** version (v0.1.1, v0.1.2): Bug fixes, backwards compatible

## Troubleshooting

### Release failed: "secret HOMEBREW_TAP_GITHUB_TOKEN not found"

The GitHub token secret is not set. Follow step 2 in Prerequisites above.

### Release succeeded but Homebrew tap not updated

Check that:
1. The `homebrew-igt` repository exists at https://github.com/mather/homebrew-igt
2. The `HOMEBREW_TAP_GITHUB_TOKEN` has `repo` scope
3. Check GitHub Actions logs for errors

### Cannot install via Homebrew

```bash
# Update Homebrew
brew update

# Try again
brew tap mather/igt
brew install igt
```

## Manual Release (if needed)

If GitHub Actions fails, you can create a release manually:

```bash
# Install GoReleaser
brew install goreleaser

# Create a release (requires GITHUB_TOKEN environment variable)
export GITHUB_TOKEN="your-github-token"
export HOMEBREW_TAP_GITHUB_TOKEN="your-homebrew-tap-token"
goreleaser release --clean
```
