package crypto_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/jclement/deaddrop/internal/crypto"
)

func TestGeneratePassphrase(t *testing.T) {
	tests := []struct {
		name      string
		wordCount int
	}{
		{"5 words", 5},
		{"6 words", 6},
		{"7 words", 7},
		{"10 words", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, err := crypto.GeneratePassphrase(tt.wordCount)
			if err != nil {
				t.Fatalf("GeneratePassphrase(%d): %v", tt.wordCount, err)
			}
			words := strings.Split(pass, crypto.WordSeparator)
			if len(words) != tt.wordCount {
				t.Errorf("got %d words, want %d: %q", len(words), tt.wordCount, pass)
			}
		})
	}
}

func TestGeneratePassphraseMinWords(t *testing.T) {
	_, err := crypto.GeneratePassphrase(4)
	if err == nil {
		t.Fatal("expected error for wordCount < MinWordCount")
	}
	_, err = crypto.GeneratePassphrase(0)
	if err == nil {
		t.Fatal("expected error for wordCount 0")
	}
}

func TestGeneratePassphraseWordFormat(t *testing.T) {
	pass, err := crypto.GeneratePassphrase(6)
	if err != nil {
		t.Fatalf("GeneratePassphrase: %v", err)
	}
	wordPattern := regexp.MustCompile(`^[a-z]+$`)
	for _, word := range strings.Split(pass, crypto.WordSeparator) {
		if !wordPattern.MatchString(word) {
			t.Errorf("word %q does not match ^[a-z]+$", word)
		}
	}
}

func TestGeneratePassphraseUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pass, err := crypto.GeneratePassphrase(6)
		if err != nil {
			t.Fatalf("GeneratePassphrase: %v", err)
		}
		if seen[pass] {
			t.Fatalf("duplicate passphrase after %d generations: %q", i, pass)
		}
		seen[pass] = true
	}
}
