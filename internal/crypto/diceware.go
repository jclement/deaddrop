package crypto

import (
	"fmt"
	"strings"

	"github.com/sethvargo/go-diceware/diceware"
)

const (
	// DefaultWordCount is the default number of diceware words.
	DefaultWordCount = 6

	// MinWordCount is the minimum allowed word count.
	MinWordCount = 5

	// WordSeparator is the character used between diceware words.
	WordSeparator = "-"
)

// GeneratePassphrase generates a diceware passphrase with the given number
// of words, separated by hyphens. Uses the EFF long wordlist and crypto/rand.
func GeneratePassphrase(wordCount int) (string, error) {
	if wordCount < MinWordCount {
		return "", fmt.Errorf("word count must be at least %d, got %d", MinWordCount, wordCount)
	}
	words, err := diceware.Generate(wordCount)
	if err != nil {
		return "", fmt.Errorf("generating passphrase: %w", err)
	}
	return strings.Join(words, WordSeparator), nil
}
