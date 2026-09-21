package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateCwdContainment verifies that cwd conforms to the profile policy and
// remains contained within the assigned worktree root.
func ValidateCwdContainment(cwd, worktreeRoot, cwdPolicy string) error {
	// Default policy if omitted
	if cwdPolicy == "" {
		cwdPolicy = "worktree_root"
	}

	if cwdPolicy != "worktree_root" && cwdPolicy != "worktree_contained" {
		return newPathError("cwd_policy", fmt.Sprintf("unsupported cwd_policy %q: allowed 'worktree_root' or 'worktree_contained'", cwdPolicy))
	}

	trimmed := strings.TrimSpace(cwd)

	// Lexical escape checks
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "\\") {
		return newPathError("cwd", fmt.Sprintf("absolute or leading slash path %q is forbidden", cwd))
	}

	// Volume / drive checks (e.g. C:, C:, C:foo)
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
	if strings.HasPrefix(trimmed, `\\?\`) || strings.HasPrefix(trimmed, `\\??\`) {
		return newPathError("cwd", fmt.Sprintf("device path %q is forbidden", cwd))
	}

	// Normalized clean relative path
	cleanCwd := filepath.Clean(filepath.ToSlash(trimmed))
	if cleanCwd == ".." || strings.HasPrefix(cleanCwd, "../") || strings.Contains(cleanCwd, "/../") {
		return newPathError("cwd", fmt.Sprintf("parent traversal %q escaping root is forbidden", cwd))
	}

	// Policy enforcement
	if cwdPolicy == "worktree_root" {
		if cleanCwd != "." && cleanCwd != "" {
			return newPathError("cwd", fmt.Sprintf("policy 'worktree_root' requires cwd to be '.' or empty, got %q", cwd))
		}
		return nil
	}

	// "worktree_contained"
	if cleanCwd == "." || cleanCwd == "" {
		return nil
	}

	// If worktreeRoot is specified, evaluate physical containment & symlinks
	if worktreeRoot != "" {
		cleanRoot := filepath.Clean(worktreeRoot)
		realRoot, err := filepath.EvalSymlinks(cleanRoot)
		if err != nil {
			realRoot, _ = filepath.Abs(cleanRoot)
		}

		target := filepath.Join(cleanRoot, filepath.FromSlash(cleanCwd))

		// Check if target physically exists
		if _, err := os.Stat(target); err == nil {
			realTarget, err := filepath.EvalSymlinks(target)
			if err != nil {
				return newPathError("cwd", fmt.Sprintf("failed to resolve symlinks for %q: %v", target, err))
			}
			if !isWithinRoot(realTarget, realRoot) {
				return newPathError("cwd", fmt.Sprintf("resolved target %q escapes worktree root %q", realTarget, realRoot))
			}
		} else {
			// Nonexistent target: lexical containment + check longest existing ancestor
			if !isWithinRoot(target, cleanRoot) {
				return newPathError("cwd", fmt.Sprintf("target %q lexically escapes worktree root %q", target, cleanRoot))
			}

			// Find longest existing ancestor
			existingAncestor := target
			for {
				parent := filepath.Dir(existingAncestor)
				if parent == existingAncestor {
					break
				}
				existingAncestor = parent
				if _, err := os.Stat(existingAncestor); err == nil {
					break
				}
			}

			if _, err := os.Stat(existingAncestor); err == nil {
				realAncestor, err := filepath.EvalSymlinks(existingAncestor)
				if err == nil && !isWithinRoot(realAncestor, realRoot) {
					return newPathError("cwd", fmt.Sprintf("resolved ancestor %q escapes worktree root %q", realAncestor, realRoot))
				}
			}
		}
	}

	return nil
}

// isWithinRoot checks if target is either root itself or a descendant of root.
func isWithinRoot(target, root string) bool {
	targetClean := filepath.Clean(target)
	rootClean := filepath.Clean(root)

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
		trimmed := strings.TrimSpace(pattern)
		if len(trimmed) == 0 {
			return newScopeError("pattern", "scope pattern cannot be empty")
		}

		if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "\\") {
			return newScopeError("pattern", fmt.Sprintf("absolute scope pattern %q is forbidden", pattern))
		}

		if len(trimmed) >= 2 && trimmed[1] == ':' {
			return newScopeError("pattern", fmt.Sprintf("drive-qualified scope pattern %q is forbidden", pattern))
		}
		if vol := filepath.VolumeName(trimmed); vol != "" {
			return newScopeError("pattern", fmt.Sprintf("volume-qualified scope pattern %q is forbidden", pattern))
		}

		if strings.HasPrefix(trimmed, "\\\\") || strings.HasPrefix(trimmed, "//") {
			return newScopeError("pattern", fmt.Sprintf("UNC scope pattern %q is forbidden", pattern))
		}

		if strings.HasPrefix(trimmed, `\\?\`) || strings.HasPrefix(trimmed, `\\??\`) {
			return newScopeError("pattern", fmt.Sprintf("device scope pattern %q is forbidden", pattern))
		}

		// Split by slash and backslash to verify no segment is '..'
		norm := strings.ReplaceAll(trimmed, "\\", "/")
		parts := strings.Split(norm, "/")
		for _, part := range parts {
			if part == ".." {
				return newScopeError("pattern", fmt.Sprintf("scope pattern %q contains forbidden parent traversal '..'", pattern))
			}
		}
	}
	return nil
}
