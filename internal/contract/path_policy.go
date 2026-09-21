package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateCwdContainment verifies that cwd conforms to the profile policy and
// remains contained within the assigned trusted worktree root.
func ValidateCwdContainment(cwd, worktreeRoot, cwdPolicy string) error {
	// 1. Canonical root procedure: trusted worktree root MUST be validated BEFORE
	// accepting either cwd = "." or worktree_contained paths.
	if strings.TrimSpace(worktreeRoot) == "" {
		return newPathError("worktree_root", "worktree root cannot be empty")
	}

	absRoot, err := filepath.Abs(worktreeRoot)
	if err != nil {
		return newPathError("worktree_root", fmt.Sprintf("failed to get absolute path for worktree root %q: %v", worktreeRoot, err))
	}
	cleanRoot := filepath.Clean(absRoot)

	fi, err := os.Stat(cleanRoot)
	if err != nil {
		return newPathError("worktree_root", fmt.Sprintf("worktree root %q does not exist or cannot be accessed: %v", cleanRoot, err))
	}
	if !fi.IsDir() {
		return newPathError("worktree_root", fmt.Sprintf("worktree root %q is not a directory", cleanRoot))
	}

	realRoot, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		// EvalSymlinks MUST succeed; NO fallback such as filepath.Abs allowed.
		return newPathError("worktree_root", fmt.Sprintf("failed to resolve symlinks for worktree root %q: %v", cleanRoot, err))
	}
	canonicalRealRoot := filepath.Clean(realRoot)

	fiReal, err := os.Stat(canonicalRealRoot)
	if err != nil {
		return newPathError("worktree_root", fmt.Sprintf("canonical worktree root %q cannot be accessed: %v", canonicalRealRoot, err))
	}
	if !fiReal.IsDir() {
		return newPathError("worktree_root", fmt.Sprintf("canonical worktree root %q is not a directory", canonicalRealRoot))
	}

	// 2. Policy validation
	if cwdPolicy == "" {
		cwdPolicy = "worktree_root"
	}
	if cwdPolicy != "worktree_root" && cwdPolicy != "worktree_contained" {
		return newPathError("cwd_policy", fmt.Sprintf("unsupported cwd_policy %q: allowed 'worktree_root' or 'worktree_contained'", cwdPolicy))
	}

	// 12. Path String Canonicality: leading/trailing whitespace rejected
	if cwd != strings.TrimSpace(cwd) {
		return newPathError("cwd", fmt.Sprintf("cwd %q contains forbidden leading or trailing whitespace", cwd))
	}

	trimmed := cwd

	// Lexical escape checks
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "\\") {
		return newPathError("cwd", fmt.Sprintf("absolute or leading slash path %q is forbidden", cwd))
	}

	// Volume / drive checks (e.g. C:, C:\, C:foo)
	if len(trimmed) >= 2 && trimmed[1] == ':' {
		return newPathError("cwd", fmt.Sprintf("drive-qualified path %q is forbidden", cwd))
	}
	if vol := filepath.VolumeName(trimmed); vol != "" {
		return newPathError("cwd", fmt.Sprintf("volume-qualified path %q is forbidden", cwd))
	}

	// UNC check
	if strings.HasPrefix(trimmed, "\\\\") || strings.HasPrefix(trimmed, "//") {
		return newPathError("cwd", fmt.Sprintf("UNC path %q is forbidden", cwd))
	}

	// Device path check (\\?\ or \??\)
	if strings.HasPrefix(trimmed, `\\?\`) || strings.HasPrefix(trimmed, `\??\`) {
		return newPathError("cwd", fmt.Sprintf("device path %q is forbidden", cwd))
	}

	// 8. Canonical Relative Path Representation
	nativeClean := filepath.Clean(
		filepath.FromSlash(
			strings.ReplaceAll(trimmed, "\\", "/"),
		),
	)
	canonicalSlash := filepath.ToSlash(nativeClean)

	// Lexical traversal check
	if canonicalSlash == ".." || strings.HasPrefix(canonicalSlash, "../") || strings.Contains(canonicalSlash, "/../") {
		return newPathError("cwd", fmt.Sprintf("parent traversal %q escaping root is forbidden", cwd))
	}

	// 4. Policy enforcement for "worktree_root"
	if cwdPolicy == "worktree_root" {
		if canonicalSlash != "." && canonicalSlash != "" {
			return newPathError("cwd", fmt.Sprintf("policy 'worktree_root' requires cwd to be '.' or empty, got %q", cwd))
		}
		return nil
	}

	// 5. Policy enforcement for "worktree_contained"
	if canonicalSlash == "." || canonicalSlash == "" {
		return nil
	}

	// Lexical containment against canonical real root
	targetLexical := filepath.Join(canonicalRealRoot, filepath.FromSlash(canonicalSlash))
	if !isWithinRoot(targetLexical, canonicalRealRoot) {
		return newPathError("cwd", fmt.Sprintf("target %q lexically escapes worktree root %q", cwd, canonicalRealRoot))
	}

	// 9. Component-aware existing prefix walk from canonical real root toward target
	components := strings.Split(canonicalSlash, "/")
	currentPath := canonicalRealRoot
	foundNonExistent := false

	for i, comp := range components {
		if comp == "" || comp == "." {
			continue
		}

		if foundNonExistent {
			continue
		}

		nextPath := filepath.Join(currentPath, filepath.FromSlash(comp))

		// Inspect component using os.Lstat to distinguish existing links/reparse entries from nonexistence
		_, err := os.Lstat(nextPath)
		if err != nil {
			if os.IsNotExist(err) {
				// First truly nonexistent component confirmed by os.IsNotExist
				foundNonExistent = true
				continue
			}
			// Fail closed on any other filesystem error
			return newPathError("cwd", fmt.Sprintf("filesystem error inspecting path component %q: %v", nextPath, err))
		}

		// Component physically exists on disk. Resolve symlinks/junctions.
		realNext, err := filepath.EvalSymlinks(nextPath)
		if err != nil {
			return newPathError("cwd", fmt.Sprintf("failed to resolve symlink or path component %q: %v", nextPath, err))
		}

		// Resolved component must remain root-or-descendant of canonical real root
		if !isWithinRoot(realNext, canonicalRealRoot) {
			return newPathError("cwd", fmt.Sprintf("resolved path component %q (%q) escapes worktree root %q", nextPath, realNext, canonicalRealRoot))
		}

		// If intermediate component (not leaf), it must be a directory
		if i < len(components)-1 {
			nextFi, err := os.Stat(realNext)
			if err != nil {
				return newPathError("cwd", fmt.Sprintf("failed to stat resolved intermediate component %q: %v", realNext, err))
			}
			if !nextFi.IsDir() {
				return newPathError("cwd", fmt.Sprintf("intermediate path component %q is not a directory", realNext))
			}
		}

		currentPath = realNext
	}

	return nil
}

// isWithinRoot checks if target is either root itself or a descendant of root.
func isWithinRoot(target, root string) bool {
	targetClean := strings.TrimPrefix(filepath.Clean(target), `\\?\`)
	rootClean := strings.TrimPrefix(filepath.Clean(root), `\\?\`)

	if strings.EqualFold(targetClean, rootClean) {
		return true
	}

	rel, err := filepath.Rel(rootClean, targetClean)
	if err != nil {
		return false
	}
	relClean := filepath.Clean(rel)
	if relClean == "." {
		return true
	}
	if relClean == ".." || strings.HasPrefix(relClean, ".."+string(filepath.Separator)) || strings.HasPrefix(relClean, "../") {
		return false
	}
	return true
}

// ValidateScopePatterns performs pre-dispatch lexical containment on scope glob patterns.
func ValidateScopePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if pattern != strings.TrimSpace(pattern) {
			return newScopeError("pattern", fmt.Sprintf("scope pattern %q contains forbidden leading or trailing whitespace", pattern))
		}
		if len(pattern) == 0 {
			return newScopeError("pattern", "scope pattern cannot be empty")
		}

		if strings.HasPrefix(pattern, "/") || strings.HasPrefix(pattern, "\\") {
			return newScopeError("pattern", fmt.Sprintf("absolute scope pattern %q is forbidden", pattern))
		}

		if len(pattern) >= 2 && pattern[1] == ':' {
			return newScopeError("pattern", fmt.Sprintf("drive-qualified scope pattern %q is forbidden", pattern))
		}
		if vol := filepath.VolumeName(pattern); vol != "" {
			return newScopeError("pattern", fmt.Sprintf("volume-qualified scope pattern %q is forbidden", pattern))
		}

		if strings.HasPrefix(pattern, "\\\\") || strings.HasPrefix(pattern, "//") {
			return newScopeError("pattern", fmt.Sprintf("UNC scope pattern %q is forbidden", pattern))
		}

		if strings.HasPrefix(pattern, `\\?\`) || strings.HasPrefix(pattern, `\??\`) {
			return newScopeError("pattern", fmt.Sprintf("device scope pattern %q is forbidden", pattern))
		}

		// Split by slash and backslash to verify no segment is '..'
		norm := strings.ReplaceAll(pattern, "\\", "/")
		parts := strings.Split(norm, "/")
		for _, part := range parts {
			if part == ".." {
				return newScopeError("pattern", fmt.Sprintf("scope pattern %q contains forbidden parent traversal '..'", pattern))
			}
		}
	}
	return nil
}
