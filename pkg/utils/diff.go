package utils

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/sergi/go-diff/diffmatchpatch"
	"k8s.io/klog/v2"
)

const (
	// MaxDiffSize limits the diff size to prevent memory issues
	MaxDiffSize = 5 * 1024 * 1024 // 5MB
	// MaxYAMLSize limits the YAML size before diffing
	MaxYAMLSize = 2 * 1024 * 1024 // 2MB per YAML
)

// GenerateUnifiedDiff creates a unified diff between two YAML strings
// Returns empty string if inputs are identical or if there's an error
// This is memory-safe and prevents injection attacks
func GenerateUnifiedDiff(oldYAML, newYAML string) string {
	// Validate input sizes to prevent memory exhaustion
	if len(oldYAML) > MaxYAMLSize || len(newYAML) > MaxYAMLSize {
		klog.Warningf("YAML content too large for diff: old=%d, new=%d bytes", len(oldYAML), len(newYAML))
		return ""
	}

	// Sanitize inputs - ensure they are valid UTF-8
	if !utf8.ValidString(oldYAML) || !utf8.ValidString(newYAML) {
		klog.Warning("Invalid UTF-8 in YAML content, skipping diff")
		return ""
	}

	// Quick check if they're identical
	if oldYAML == newYAML {
		return ""
	}

	// Use diffmatchpatch for efficient diffing
	dmp := diffmatchpatch.New()
	
	// Set timeout to prevent long-running diffs
	dmp.DiffTimeout = 2 // 2 seconds max
	
	// Compute line-based diff for better readability
	diffs := dmp.DiffMain(oldYAML, newYAML, true)
	
	// Check if diff is too large
	diffSize := 0
	for _, d := range diffs {
		diffSize += len(d.Text)
	}
	if diffSize > MaxDiffSize {
		klog.Warningf("Diff too large: %d bytes, truncating", diffSize)
		return "# Diff too large to store\n"
	}
	
	// Convert to unified diff format (similar to git diff)
	patch := dmp.PatchMake(oldYAML, diffs)
	unifiedDiff := dmp.PatchToText(patch)
	
	return unifiedDiff
}

// ApplyDiff applies a unified diff patch to old content to reconstruct new content
// Returns empty string on error
func ApplyDiff(oldYAML, diffPatch string) string {
	if diffPatch == "" {
		return oldYAML
	}

	// Validate input sizes
	if len(oldYAML) > MaxYAMLSize || len(diffPatch) > MaxDiffSize {
		klog.Warningf("Content too large for patch application: old=%d, diff=%d bytes", len(oldYAML), len(diffPatch))
		return ""
	}

	// Sanitize inputs
	if !utf8.ValidString(oldYAML) || !utf8.ValidString(diffPatch) {
		klog.Warning("Invalid UTF-8 in patch content")
		return ""
	}

	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(diffPatch)
	if err != nil {
		klog.Errorf("Failed to parse diff patch: %v", err)
		return ""
	}

	newText, applied := dmp.PatchApply(patches, oldYAML)
	
	// Check if all patches were applied successfully
	allApplied := true
	for _, success := range applied {
		if !success {
			allApplied = false
			break
		}
	}
	
	if !allApplied {
		klog.Warning("Some patches failed to apply")
	}

	return newText
}

// GenerateHumanReadableDiff creates a human-readable unified diff format
// Similar to `diff -u` output, safe for display
func GenerateHumanReadableDiff(oldYAML, newYAML, oldLabel, newLabel string) string {
	// Validate inputs
	if len(oldYAML) > MaxYAMLSize || len(newYAML) > MaxYAMLSize {
		return "# Content too large to diff\n"
	}

	if !utf8.ValidString(oldYAML) || !utf8.ValidString(newYAML) {
		return "# Invalid UTF-8 content\n"
	}

	// Sanitize labels to prevent injection
	oldLabel = sanitizeLabel(oldLabel)
	newLabel = sanitizeLabel(newLabel)

	// If identical, return empty
	if oldYAML == newYAML {
		return "# No changes\n"
	}

	// Generate unified diff header
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("--- %s\n", oldLabel))
	buf.WriteString(fmt.Sprintf("+++ %s\n", newLabel))

	// Use diffmatchpatch for line-level diff
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldYAML, newYAML, true)
	diffs = dmp.DiffCleanupSemantic(diffs)

	// Convert to unified diff format
	lineNum := 1
	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		for i, line := range lines {
			if i == len(lines)-1 && line == "" {
				continue
			}
			
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				buf.WriteString(fmt.Sprintf("-%s\n", line))
			case diffmatchpatch.DiffInsert:
				buf.WriteString(fmt.Sprintf("+%s\n", line))
			case diffmatchpatch.DiffEqual:
				// Only show context lines (not all equal lines)
				if i < 3 || i >= len(lines)-3 {
					buf.WriteString(fmt.Sprintf(" %s\n", line))
				} else if i == 3 {
					buf.WriteString("...\n")
				}
				lineNum++
			}
		}
	}

	result := buf.String()
	if len(result) > MaxDiffSize {
		return "# Diff too large to display\n"
	}

	return result
}

// sanitizeLabel removes potentially dangerous characters from labels
func sanitizeLabel(label string) string {
	// Remove control characters and limit length
	var buf bytes.Buffer
	for _, r := range label {
		if r < 32 || r == 127 { // Control characters
			continue
		}
		buf.WriteRune(r)
		if buf.Len() >= 100 {
			break
		}
	}
	result := buf.String()
	if result == "" {
		return "untitled"
	}
	return result
}
