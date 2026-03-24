package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	SaltSize    = 32
	NonceSize   = 12 // AES-256-GCM standard nonce
	KeySize     = 32 // AES-256
	HeaderSize  = SaltSize + 4 + 4 + 1 + NonceSize // salt + time + memory + threads + nonce = 53 bytes

	DefaultArgonTime    = 3
	DefaultArgonMemory  = 64 * 1024 // 64 MB
	DefaultArgonThreads = 4
)

// ArgonParams holds the Argon2id parameters stored in the vault header.
type ArgonParams struct {
	Time    uint32
	Memory  uint32
	Threads uint8
}

// DefaultArgonParams returns the default Argon2id parameters.
func DefaultArgonParams() ArgonParams {
	return ArgonParams{
		Time:    DefaultArgonTime,
		Memory:  DefaultArgonMemory,
		Threads: DefaultArgonThreads,
	}
}

// DeriveKey derives an AES-256 key from a passphrase and salt using Argon2id.
func DeriveKey(passphrase []byte, salt []byte, params ArgonParams) []byte {
	return argon2.IDKey(passphrase, salt, params.Time, params.Memory, params.Threads, KeySize)
}

// GenerateSalt creates a cryptographically random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts plaintext with AES-256-GCM and returns the full vault file bytes:
// [salt:32][time:4][memory:4][threads:1][nonce:12][ciphertext+tag:...]
func Encrypt(plaintext, passphrase []byte, salt []byte, params ArgonParams) ([]byte, error) {
	key := DeriveKey(passphrase, salt, params)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Build header + ciphertext
	header := make([]byte, HeaderSize)
	copy(header[0:SaltSize], salt)
	binary.BigEndian.PutUint32(header[SaltSize:SaltSize+4], params.Time)
	binary.BigEndian.PutUint32(header[SaltSize+4:SaltSize+8], params.Memory)
	header[SaltSize+8] = params.Threads
	copy(header[SaltSize+9:], nonce)

	return append(header, ciphertext...), nil
}

// Decrypt decrypts a vault file. Returns the plaintext JSON.
func Decrypt(data, passphrase []byte) ([]byte, error) {
	if len(data) < HeaderSize {
		return nil, errors.New("vault file too short")
	}

	salt := data[0:SaltSize]
	params := ArgonParams{
		Time:    binary.BigEndian.Uint32(data[SaltSize : SaltSize+4]),
		Memory:  binary.BigEndian.Uint32(data[SaltSize+4 : SaltSize+8]),
		Threads: data[SaltSize+8],
	}
	nonce := data[SaltSize+9 : HeaderSize]
	ciphertext := data[HeaderSize:]

	key := DeriveKey(passphrase, salt, params)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed — wrong passphrase or corrupted vault")
	}

	return plaintext, nil
}

// DecryptWithKey decrypts a vault file using a pre-derived key (from session).
func DecryptWithKey(data, key []byte) ([]byte, error) {
	if len(data) < HeaderSize {
		return nil, errors.New("vault file too short")
	}

	nonce := data[SaltSize+9 : HeaderSize]
	ciphertext := data[HeaderSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed — wrong key or corrupted vault")
	}

	return plaintext, nil
}

// EncryptWithKey encrypts plaintext using a pre-derived key (from session).
// Reads the existing salt and params from the current vault data to preserve them.
func EncryptWithKey(plaintext, key, existingVaultData []byte) ([]byte, error) {
	if len(existingVaultData) < HeaderSize {
		return nil, errors.New("existing vault data too short to extract header")
	}

	// Preserve salt and params from existing vault
	salt := existingVaultData[0:SaltSize]
	params := ArgonParams{
		Time:    binary.BigEndian.Uint32(existingVaultData[SaltSize : SaltSize+4]),
		Memory:  binary.BigEndian.Uint32(existingVaultData[SaltSize+4 : SaltSize+8]),
		Threads: existingVaultData[SaltSize+8],
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	// Fresh nonce for every write
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	header := make([]byte, HeaderSize)
	copy(header[0:SaltSize], salt)
	binary.BigEndian.PutUint32(header[SaltSize:SaltSize+4], params.Time)
	binary.BigEndian.PutUint32(header[SaltSize+4:SaltSize+8], params.Memory)
	header[SaltSize+8] = params.Threads
	copy(header[SaltSize+9:], nonce)

	return append(header, ciphertext...), nil
}
