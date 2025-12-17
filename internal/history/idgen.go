package history

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// adjectives is a list of descriptive words for memorable ID generation.
var adjectives = []string{
	"bold", "brave", "bright", "calm", "clever",
	"cool", "crisp", "daring", "eager", "fair",
	"fast", "firm", "fleet", "fresh", "gentle",
	"glad", "grand", "great", "happy", "hardy",
	"keen", "kind", "lively", "loyal", "lucid",
	"merry", "mighty", "noble", "patient", "proud",
	"quick", "quiet", "rapid", "ready", "sharp",
	"sleek", "smart", "smooth", "solid", "steady",
	"strong", "sturdy", "subtle", "sure", "swift",
	"true", "vivid", "warm", "wise", "witty",
}

// nouns is a list of concrete nouns for memorable ID generation.
var nouns = []string{
	"arrow", "beacon", "birch", "brook", "cedar",
	"cliff", "cloud", "coral", "crane", "creek",
	"delta", "eagle", "ember", "falcon", "fern",
	"flame", "flint", "forge", "frost", "glade",
	"grove", "hawk", "heron", "hill", "iris",
	"jade", "lake", "lark", "leaf", "maple",
	"marsh", "meadow", "oak", "ocean", "olive",
	"pebble", "pine", "pond", "raven", "reef",
	"river", "robin", "sage", "shade", "shore",
	"spark", "spruce", "stone", "swift", "vale",
}

// GenerateID creates a unique identifier in adjective_noun_YYYYMMDD_HHMMSS format.
// Uses crypto/rand for secure random word selection to prevent collisions.
func GenerateID() (string, error) {
	adj, err := randomWord(adjectives)
	if err != nil {
		return "", fmt.Errorf("selecting random adjective: %w", err)
	}

	noun, err := randomWord(nouns)
	if err != nil {
		return "", fmt.Errorf("selecting random noun: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s_%s", adj, noun, timestamp), nil
}

// randomWord selects a random word from the given slice using crypto/rand.
func randomWord(words []string) (string, error) {
	if len(words) == 0 {
		return "", fmt.Errorf("word list is empty")
	}

	max := big.NewInt(int64(len(words)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generating random number: %w", err)
	}

	return words[n.Int64()], nil
}
