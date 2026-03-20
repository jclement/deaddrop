package crypto

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"

	"filippo.io/age"
)

// MagicHeader is the 4-byte prefix identifying a Dead Drop payload.
var MagicHeader = []byte("DD01")

// ErrInvalidHeader is returned when the payload does not start with the expected magic bytes.
var ErrInvalidHeader = errors.New("invalid dead drop header: expected DD01")

// DefaultWorkFactor is the scrypt work factor (log2 N) for age encryption.
const DefaultWorkFactor = 18

// MinWorkFactor is the minimum recommended scrypt work factor.
// Values below this are trivially brute-forceable.
const MinWorkFactor = 14

// Compress applies zlib compression to data.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("compressing: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing compressor: %w", err)
	}
	return buf.Bytes(), nil
}

// Decompress reverses zlib compression.
func Decompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompressing: %w", err)
	}
	defer r.Close()
	result, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading decompressed data: %w", err)
	}
	return result, nil
}

// Encrypt compresses the plaintext, encrypts it with age scrypt using the
// given passphrase and work factor, and prepends the DD01 magic header.
// Returns the complete payload: DD01 || age_ciphertext.
func Encrypt(plaintext []byte, passphrase string, workFactor int) ([]byte, error) {
	compressed, err := Compress(plaintext)
	if err != nil {
		return nil, err
	}

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating scrypt recipient: %w", err)
	}
	recipient.SetWorkFactor(workFactor)

	var ageBuf bytes.Buffer
	w, err := age.Encrypt(&ageBuf, recipient)
	if err != nil {
		return nil, fmt.Errorf("initializing encryption: %w", err)
	}
	if _, err := w.Write(compressed); err != nil {
		return nil, fmt.Errorf("writing encrypted data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalizing encryption: %w", err)
	}

	var payload bytes.Buffer
	payload.Write(MagicHeader)
	payload.Write(ageBuf.Bytes())
	return payload.Bytes(), nil
}

// Decrypt strips the DD01 header, decrypts the age ciphertext with the
// given passphrase, and decompresses the result.
func Decrypt(payload []byte, passphrase string) ([]byte, error) {
	if !bytes.HasPrefix(payload, MagicHeader) {
		return nil, ErrInvalidHeader
	}
	ageCiphertext := payload[len(MagicHeader):]

	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating scrypt identity: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(ageCiphertext), identity)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	compressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading decrypted data: %w", err)
	}

	return Decompress(compressed)
}
