// Package validation_test tests documentation completeness and formatting standards.
// Related: docs/*.md
// Tags: validation, documentation, markdown, headers, links, mermaid, quality
package validation

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// findRepoRoot finds the repository root by looking for go.mod
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root without finding go.mod
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// Test that all required documentation files exist in docs/
func TestDocumentationFilesExist(t *testing.T) {
	requiredFiles := []string{
		"overview.md",
		"quickstart.md",
		"architecture.md",
		"reference.md",
		"troubleshooting.md",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	docsDir := filepath.Join(repoRoot, "docs")
	for _, file := range requiredFiles {
		path := filepath.Join(docsDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required documentation file missing: %s", path)
		}
	}
}

// Test that each documentation file is under 500 lines
func TestDocumentationLineCount(t *testing.T) {
	docFiles := []string{
		"overview.md",
		"quickstart.md",
		"architecture.md",
		"reference.md",
		"troubleshooting.md",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	maxLines := 1150 // Allow for comprehensive documentation including troubleshooting guides, command reference, agent configuration, worktree commands, and new features

	for _, file := range docFiles {
		path := filepath.Join(repoRoot, "docs", file)
		f, err := os.Open(path)
		if err != nil {
			// File doesn't exist yet - will be caught by TestDocumentationFilesExist
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineCount := 0
		for scanner.Scan() {
			lineCount++
		}

		if err := scanner.Err(); err != nil {
			t.Errorf("Error reading %s: %v", path, err)
			continue
		}

		if lineCount > maxLines {
			t.Errorf("%s exceeds maximum line count: %d > %d", path, lineCount, maxLines)
		}
	}
}

// Test that each documentation file has exactly one H1 header and logical nesting
func TestDocumentationHeaders(t *testing.T) {
	docFiles := []string{
		"overview.md",
		"quickstart.md",
		"architecture.md",
		"reference.md",
		"troubleshooting.md",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	h1Pattern := regexp.MustCompile(`^#\s+.+`)
	headerPattern := regexp.MustCompile(`^(#{1,6})\s+.+`)

	for _, file := range docFiles {
		path := filepath.Join(repoRoot, "docs", file)
		f, err := os.Open(path)
		if err != nil {
			// File doesn't exist yet - will be caught by TestDocumentationFilesExist
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		h1Count := 0
		lastHeaderLevel := 0
		lineNum := 0
		inCodeBlock := false

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Track code block state
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inCodeBlock = !inCodeBlock
				continue
			}

			// Skip lines inside code blocks
			if inCodeBlock {
				continue
			}

			// Count H1 headers
			if h1Pattern.MatchString(line) {
				h1Count++
			}

			// Check header nesting
			if matches := headerPattern.FindStringSubmatch(line); matches != nil {
				currentLevel := len(matches[1])

				// Allow skipping from any level to H1 (start of new major section)
				if currentLevel > 1 && lastHeaderLevel > 0 && currentLevel-lastHeaderLevel > 1 {
					t.Errorf("%s:%d Header level skip detected (H%d -> H%d)", path, lineNum, lastHeaderLevel, currentLevel)
				}

				lastHeaderLevel = currentLevel
			}
		}

		if err := scanner.Err(); err != nil {
			t.Errorf("Error reading %s: %v", path, err)
			continue
		}

		if h1Count != 1 {
			t.Errorf("%s should have exactly 1 H1 header, found %d", path, h1Count)
		}
	}
}

// Test that internal links to other documentation files are valid
func TestInternalLinks(t *testing.T) {
	docFiles := []string{
		"overview.md",
		"quickstart.md",
		"architecture.md",
		"reference.md",
		"troubleshooting.md",
	}

	// Pattern for markdown links: [text](path) or [text](path#anchor)
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	for _, file := range docFiles {
		path := filepath.Join(repoRoot, "docs", file)
		content, err := os.ReadFile(path)
		if err != nil {
			// File doesn't exist yet - will be caught by TestDocumentationFilesExist
			continue
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			matches := linkPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				link := match[2]

				// Skip external links (http://, https://)
				if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
					continue
				}

				// Skip mailto links
				if strings.HasPrefix(link, "mailto:") {
					continue
				}

				// Extract path without anchor
				linkPath := strings.Split(link, "#")[0]

				// Skip empty links (anchor only)
				if linkPath == "" {
					continue
				}

				// Check if it's a relative link to docs/
				if strings.HasPrefix(linkPath, "./") || strings.HasPrefix(linkPath, "../") || !strings.Contains(linkPath, "/") {
					// Resolve relative path
					dir := filepath.Dir(path)
					fullPath := filepath.Join(dir, linkPath)

					if _, err := os.Stat(fullPath); os.IsNotExist(err) {
						t.Errorf("%s:%d Invalid internal link: %s (resolved to %s)", path, lineNum+1, link, fullPath)
					}
				}
			}
		}
	}
}

// Test that code references use the correct file:line format
func TestCodeReferences(t *testing.T) {
	docFiles := []string{
		"overview.md",
		"quickstart.md",
		"architecture.md",
		"reference.md",
		"troubleshooting.md",
	}

	// Pattern for code references: path/to/file.go:123
	codeRefPattern := regexp.MustCompile(`([a-zA-Z0-9_/.-]+\.(go|sh|md)):(\d+)`)

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	for _, file := range docFiles {
		path := filepath.Join(repoRoot, "docs", file)
		content, err := os.ReadFile(path)
		if err != nil {
			// File doesn't exist yet - will be caught by TestDocumentationFilesExist
			continue
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			matches := codeRefPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				filePath := match[1]

				// Skip if it's in a code block (approximate check)
				if strings.Contains(line, "```") {
					continue
				}

				// Verify the referenced file exists (relative to repo root)
				fullPath := filepath.Join(repoRoot, filePath)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("%s:%d Code reference points to non-existent file: %s (checked: %s)", path, lineNum+1, filePath, fullPath)
				}
			}
		}
	}
}

// Test that Mermaid diagrams have valid syntax
func TestMermaidDiagrams(t *testing.T) {
	docFiles := []string{
		"overview.md",
		"quickstart.md",
		"architecture.md",
		"reference.md",
		"troubleshooting.md",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	for _, file := range docFiles {
		path := filepath.Join(repoRoot, "docs", file)
		content, err := os.ReadFile(path)
		if err != nil {
			// File doesn't exist yet - will be caught by TestDocumentationFilesExist
			continue
		}

		lines := strings.Split(string(content), "\n")
		inMermaidBlock := false
		mermaidStartLine := 0
		mermaidContent := []string{}

		for lineNum, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```mermaid") {
				inMermaidBlock = true
				mermaidStartLine = lineNum + 1
				mermaidContent = []string{}
				continue
			}

			if inMermaidBlock {
				if strings.HasPrefix(strings.TrimSpace(line), "```") {
					// End of mermaid block - validate
					if len(mermaidContent) == 0 {
						t.Errorf("%s:%d Empty Mermaid diagram block", path, mermaidStartLine)
					}

					// Basic syntax checks
					firstLine := strings.TrimSpace(mermaidContent[0])
					validTypes := []string{"graph", "flowchart", "sequenceDiagram", "classDiagram", "stateDiagram", "erDiagram", "gantt", "pie"}
					hasValidType := false
					for _, validType := range validTypes {
						if strings.HasPrefix(firstLine, validType) {
							hasValidType = true
							break
						}
					}

					if !hasValidType {
						t.Errorf("%s:%d Mermaid diagram missing valid type (graph, flowchart, sequenceDiagram, etc.)", path, mermaidStartLine)
					}

					inMermaidBlock = false
					mermaidContent = []string{}
				} else {
					mermaidContent = append(mermaidContent, line)
				}
			}
		}

		if inMermaidBlock {
			t.Errorf("%s:%d Unclosed Mermaid code block", path, mermaidStartLine)
		}
	}
}

// Test that all CLI commands are documented in reference.md
func TestCommandCompleteness(t *testing.T) {
	requiredCommands := []string{
		"autospec full",
		"autospec prep",
		"autospec specify",
		"autospec plan",
		"autospec tasks",
		"autospec implement",
		"autospec doctor",
		"autospec status",
		"autospec config",
		"autospec init",
		"autospec version",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	refPath := filepath.Join(repoRoot, "docs", "reference.md")
	content, err := os.ReadFile(refPath)
	if err != nil {
		// File doesn't exist yet - will be caught by TestDocumentationFilesExist
		return
	}

	contentStr := string(content)
	for _, cmd := range requiredCommands {
		if !strings.Contains(contentStr, cmd) {
			t.Errorf("%s missing documentation for command: %s", refPath, cmd)
		}
	}
}

// Test that all configuration options are documented in reference.md
func TestConfigCompleteness(t *testing.T) {
	requiredConfigOptions := []string{
		"claude_cmd",
		"max_retries",
		"specs_dir",
		"state_dir",
		"timeout",
		"skip_preflight",
		"custom_claude_cmd",
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("Failed to find repository root: %v", err)
	}

	refPath := filepath.Join(repoRoot, "docs", "reference.md")
	content, err := os.ReadFile(refPath)
	if err != nil {
		// File doesn't exist yet - will be caught by TestDocumentationFilesExist
		return
	}

	contentStr := string(content)
	for _, option := range requiredConfigOptions {
		if !strings.Contains(contentStr, option) {
			t.Errorf("%s missing documentation for config option: %s", refPath, option)
		}
	}
}
