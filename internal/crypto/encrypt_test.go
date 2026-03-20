package crypto_test

import (
	"bytes"
	"testing"

	"github.com/jclement/deaddrop/internal/crypto"
)

func TestCompressDecompressRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello world")},
		{"long", bytes.Repeat([]byte("abcdefghij"), 1000)},
		{"binary", []byte{0x00, 0x01, 0xFF, 0xFE, 0x80, 0x7F}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := crypto.Compress(tt.data)
			if err != nil {
				t.Fatalf("Compress: %v", err)
			}
			decompressed, err := crypto.Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress: %v", err)
			}
			if !bytes.Equal(decompressed, tt.data) {
				t.Errorf("round-trip failed: got %d bytes, want %d", len(decompressed), len(tt.data))
			}
		})
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		plaintext  []byte
		passphrase string
	}{
		{"short secret", []byte("my seed phrase"), "correct-horse-battery-staple"},
		{"empty", []byte{}, "passphrase"},
		{"binary data", []byte{0x00, 0x01, 0xFF, 0xFE, 0x80}, "binary-pass"},
		{"long secret", bytes.Repeat([]byte("secret data "), 100), "long-secret-pass"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := crypto.Encrypt(tt.plaintext, tt.passphrase, 1)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			result, err := crypto.Decrypt(payload, tt.passphrase)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !bytes.Equal(result, tt.plaintext) {
				t.Errorf("round-trip failed: got %d bytes, want %d", len(result), len(tt.plaintext))
			}
		})
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	payload, err := crypto.Encrypt([]byte("secret"), "correct", 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	_, err = crypto.Decrypt(payload, "wrong")
	if err == nil {
		t.Fatal("expected error decrypting with wrong passphrase")
	}
}

func TestDecryptInvalidHeader(t *testing.T) {
	_, err := crypto.Decrypt([]byte("XXXX garbage"), "pass")
	if err != crypto.ErrInvalidHeader {
		t.Fatalf("expected ErrInvalidHeader, got: %v", err)
	}
}

func TestDecryptCorruptedPayload(t *testing.T) {
	payload := append([]byte(nil), crypto.MagicHeader...)
	payload = append(payload, []byte("not-valid-age-ciphertext")...)
	_, err := crypto.Decrypt(payload, "pass")
	if err == nil {
		t.Fatal("expected error decrypting corrupted payload")
	}
}

func TestEncryptProducesDD01Header(t *testing.T) {
	payload, err := crypto.Encrypt([]byte("test"), "pass", 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !bytes.HasPrefix(payload, crypto.MagicHeader) {
		t.Errorf("payload does not start with DD01: got %q", payload[:4])
	}
}
