package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ariel-frischer/autospec/internal/git"
)

// MaxBranchLength is GitHub's maximum branch name length in bytes
const MaxBranchLength = 244

// StopWords are common words filtered from feature descriptions when generating branch names
var StopWords = map[string]bool{
	"i": true, "a": true, "an": true, "the": true, "to": true,
	"for": true, "of": true, "in": true, "on": true, "at": true,
	"by": true, "with": true, "from": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "should": true, "could": true,
	"can": true, "may": true, "might": true, "must": true, "shall": true,
	"this": true, "that": true, "these": true, "those": true,
	"my": true, "your": true, "our": true, "their": true,
	"want": true, "need": true, "add": true, "get": true, "set": true,
}

// branchNumberPattern matches feature branch numbers like "001", "002", etc.
var branchNumberPattern = regexp.MustCompile(`^(\d{3})-`)

// GenerateBranchName generates a branch name suffix from a feature description
// It filters stop words and keeps only meaningful words (3+ characters)
func GenerateBranchName(description string) string {
	// Convert to lowercase and extract words
	lower := strings.ToLower(description)
	// Replace non-alphanumeric characters with spaces
	cleaned := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(lower, " ")
	words := strings.Fields(cleaned)

	var meaningfulWords []string
	for _, word := range words {
		if word == "" {
			continue
		}
		// Skip stop words
		if StopWords[word] {
			continue
		}
		// Keep words that are 3+ characters
		if len(word) >= 3 {
			meaningfulWords = append(meaningfulWords, word)
		} else {
			// Check if it might be an acronym (appears uppercase in original)
			upper := strings.ToUpper(word)
			if strings.Contains(description, upper) && len(upper) >= 2 {
				meaningfulWords = append(meaningfulWords, word)
			}
		}
	}

	// Use first 3-4 meaningful words
	maxWords := 3
	if len(meaningfulWords) == 4 {
		maxWords = 4
	}
	if len(meaningfulWords) > maxWords {
		meaningfulWords = meaningfulWords[:maxWords]
	}

	// If no meaningful words found, fall back to cleaned description
	if len(meaningfulWords) == 0 {
		return CleanBranchName(description)
	}

	return strings.Join(meaningfulWords, "-")
}

// CleanBranchName sanitizes a string for use as a git branch name
// It converts to lowercase, replaces invalid characters with hyphens,
// and handles edge cases like leading/trailing/consecutive hyphens
func CleanBranchName(name string) string {
	// Convert to lowercase
	lower := strings.ToLower(name)

	// Replace any non-alphanumeric character with hyphen
	cleaned := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(lower, "-")

	// Remove consecutive hyphens
	cleaned = regexp.MustCompile(`-+`).ReplaceAllString(cleaned, "-")

	// Remove leading and trailing hyphens
	cleaned = strings.Trim(cleaned, "-")

	return cleaned
}

// TruncateBranchName ensures a branch name doesn't exceed GitHub's 244-byte limit
// It truncates at word boundaries when possible and emits a warning to stderr
func TruncateBranchName(branchName string) string {
	if len(branchName) <= MaxBranchLength {
		return branchName
	}

	original := branchName

	// Account for number prefix (e.g., "001-")
	// We'll truncate the suffix part only
	parts := strings.SplitN(branchName, "-", 2)
	if len(parts) != 2 {
		// No hyphen found, just truncate
		truncated := branchName[:MaxBranchLength]
		fmt.Fprintf(os.Stderr, "[specify] Warning: Branch name exceeded GitHub's 244-byte limit\n")
		fmt.Fprintf(os.Stderr, "[specify] Original: %s (%d bytes)\n", original, len(original))
		fmt.Fprintf(os.Stderr, "[specify] Truncated to: %s (%d bytes)\n", truncated, len(truncated))
		return truncated
	}

	prefix := parts[0]
	suffix := parts[1]

	// Calculate max suffix length: MaxBranchLength - prefix length - hyphen
	maxSuffixLen := MaxBranchLength - len(prefix) - 1
	if maxSuffixLen <= 0 {
		// Edge case: prefix is too long
		truncated := branchName[:MaxBranchLength]
		fmt.Fprintf(os.Stderr, "[specify] Warning: Branch name exceeded GitHub's 244-byte limit\n")
		fmt.Fprintf(os.Stderr, "[specify] Original: %s (%d bytes)\n", original, len(original))
		fmt.Fprintf(os.Stderr, "[specify] Truncated to: %s (%d bytes)\n", truncated, len(truncated))
		return truncated
	}

	// Truncate suffix
	if len(suffix) > maxSuffixLen {
		suffix = suffix[:maxSuffixLen]
	}

	// Remove trailing hyphen if truncation created one
	suffix = strings.TrimSuffix(suffix, "-")

	truncated := prefix + "-" + suffix

	fmt.Fprintf(os.Stderr, "[specify] Warning: Branch name exceeded GitHub's 244-byte limit\n")
	fmt.Fprintf(os.Stderr, "[specify] Original: %s (%d bytes)\n", original, len(original))
	fmt.Fprintf(os.Stderr, "[specify] Truncated to: %s (%d bytes)\n", truncated, len(truncated))

	return truncated
}

// GetNextBranchNumber scans git branches and spec directories to find the next available number
// It returns a zero-padded three-digit string (e.g., "004")
func GetNextBranchNumber(specsDir string) (string, error) {
	highest := 0

	// Scan spec directories
	if info, err := os.Stat(specsDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(specsDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				name := entry.Name()
				if match := branchNumberPattern.FindStringSubmatch(name); match != nil {
					num, err := strconv.Atoi(match[1])
					if err == nil && num > highest {
						highest = num
					}
				}
			}
		}
	}

	// Scan git branches if available
	if git.IsGitRepository() {
		branches, err := git.GetBranchNames()
		if err == nil {
			for _, branch := range branches {
				if match := branchNumberPattern.FindStringSubmatch(branch); match != nil {
					num, err := strconv.Atoi(match[1])
					if err == nil && num > highest {
						highest = num
					}
				}
			}
		}
	}

	// Return next number, zero-padded to 3 digits
	next := highest + 1
	return fmt.Sprintf("%03d", next), nil
}

// FormatBranchName creates a full branch name from a number and suffix
func FormatBranchName(number, suffix string) string {
	return fmt.Sprintf("%s-%s", number, suffix)
}

// GetFeatureDirectory returns the path to a feature's spec directory
func GetFeatureDirectory(specsDir, branchName string) string {
	return filepath.Join(specsDir, branchName)
}
