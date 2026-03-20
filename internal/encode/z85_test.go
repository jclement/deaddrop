package encode_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/jclement/deaddrop/internal/encode"
)

func TestZ85RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x42}},
		{"2 bytes", []byte{0x01, 0x02}},
		{"3 bytes", []byte{0x01, 0x02, 0x03}},
		{"4 bytes", []byte{0x01, 0x02, 0x03, 0x04}},
		{"5 bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05}},
		{"hello", []byte("hello world")},
		{"100 bytes", bytes.Repeat([]byte{0xAB}, 100)},
		{"1000 bytes", bytes.Repeat([]byte("test"), 250)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := encode.EncodeZ85(tt.data)
			if err != nil {
				t.Fatalf("EncodeZ85: %v", err)
			}
			decoded, err := encode.DecodeZ85(encoded)
			if err != nil {
				t.Fatalf("DecodeZ85: %v", err)
			}
			if !bytes.Equal(decoded, tt.data) {
				t.Errorf("round-trip failed:\n  got  %x\n  want %x", decoded, tt.data)
			}
		})
	}
}

func TestZ85RoundTripRandomBinary(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 4, 7, 15, 16, 31, 32, 63, 64, 127, 128, 255, 256, 500, 1000, 2000} {
		t.Run("", func(t *testing.T) {
			data := make([]byte, size)
			if size > 0 {
				if _, err := rand.Read(data); err != nil {
					t.Fatal(err)
				}
			}
			encoded, err := encode.EncodeZ85(data)
			if err != nil {
				t.Fatalf("EncodeZ85 (size=%d): %v", size, err)
			}
			decoded, err := encode.DecodeZ85(encoded)
			if err != nil {
				t.Fatalf("DecodeZ85 (size=%d): %v", size, err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("round-trip failed for size %d", size)
			}
		})
	}
}

func TestZ85EncodeDeterministic(t *testing.T) {
	data := []byte("deterministic test input")
	enc1, _ := encode.EncodeZ85(data)
	enc2, _ := encode.EncodeZ85(data)
	if enc1 != enc2 {
		t.Errorf("encoding not deterministic:\n  %q\n  %q", enc1, enc2)
	}
}

func TestZ85DecodeInvalidInput(t *testing.T) {
	_, err := encode.DecodeZ85("!!!invalid!!!")
	if err == nil {
		t.Fatal("expected error for invalid Z85 input")
	}
}

func TestFormatZ85Block(t *testing.T) {
	encoded := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lines := encode.FormatZ85Block(encoded, 20)

	// Check no line exceeds width
	for i, line := range lines {
		if len(line) > 20 {
			t.Errorf("line %d exceeds width: %d chars", i, len(line))
		}
	}

	// Check concatenation equals original
	joined := ""
	for _, line := range lines {
		joined += line
	}
	if joined != encoded {
		t.Errorf("concatenated lines don't match original")
	}
}
