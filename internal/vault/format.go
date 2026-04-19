package vault

import (
	"encoding/binary"
	"errors"
)

// KDFMode selects how a vault's file-level key is derived from mount material.
type KDFMode uint8

const (
	// KDFArgon2id derives the AES key from a passphrase via Argon2id using the
	// salt and parameters stored in the vault header. This is the original Hort
	// format and the default for the primary vault.
	KDFArgon2id KDFMode = 0

	// KDFRawKey uses the supplied 32-byte material directly as the AES key,
	// skipping any KDF. Used for machine-mounted sources that carry their own
	// key (e.g. a Fachwerk instance's FACHWERK_SECRETS_MASTER_KEY).
	KDFRawKey KDFMode = 1
)

const (
	// V1 layout: [salt:32][time:4][memory:4][threads:1][nonce:12][ciphertext+tag:...]
	V1HeaderSize = SaltSize + 4 + 4 + 1 + NonceSize

	// V2 layout: [magic:4][version:1][kdf:1][salt:32][time:4][memory:4][threads:1][nonce:12][ciphertext+tag:...]
	V2HeaderSize = len(V2Magic) + 1 + 1 + SaltSize + 4 + 4 + 1 + NonceSize
)

// V2Magic is the 4-byte prefix that marks a v2 vault file. Legacy v1 files have
// no magic prefix and are detected by the absence of this signature.
const V2Magic = "HRT\x02"

// FormatInfo describes the header of a vault file after parsing.
type FormatInfo struct {
	Version  int
	KDF      KDFMode
	Salt     []byte
	Argon    ArgonParams
	Nonce    []byte
	CipherAt int
}

// ParseFormat inspects the vault file header and returns layout information.
// It works for both v1 (legacy, no magic) and v2 vaults.
func ParseFormat(data []byte) (FormatInfo, error) {
	if len(data) >= len(V2Magic) && string(data[0:len(V2Magic)]) == V2Magic {
		if len(data) < V2HeaderSize {
			return FormatInfo{}, errors.New("vault file too short for v2 header")
		}
		off := len(V2Magic)
		version := data[off]
		off++
		kdf := KDFMode(data[off])
		off++
		salt := data[off : off+SaltSize]
		off += SaltSize
		params := ArgonParams{
			Time:    binary.BigEndian.Uint32(data[off : off+4]),
			Memory:  binary.BigEndian.Uint32(data[off+4 : off+8]),
			Threads: data[off+8],
		}
		off += 9
		nonce := data[off : off+NonceSize]
		off += NonceSize

		if version != 2 {
			return FormatInfo{}, errors.New("unsupported vault format version")
		}
		return FormatInfo{
			Version:  2,
			KDF:      kdf,
			Salt:     salt,
			Argon:    params,
			Nonce:    nonce,
			CipherAt: V2HeaderSize,
		}, nil
	}

	if len(data) < V1HeaderSize {
		return FormatInfo{}, errors.New("vault file too short")
	}
	return FormatInfo{
		Version: 1,
		KDF:     KDFArgon2id,
		Salt:    data[0:SaltSize],
		Argon: ArgonParams{
			Time:    binary.BigEndian.Uint32(data[SaltSize : SaltSize+4]),
			Memory:  binary.BigEndian.Uint32(data[SaltSize+4 : SaltSize+8]),
			Threads: data[SaltSize+8],
		},
		Nonce:    data[SaltSize+9 : V1HeaderSize],
		CipherAt: V1HeaderSize,
	}, nil
}

// buildV1Header serializes a v1 header (legacy format, Argon2id only).
func buildV1Header(salt []byte, params ArgonParams, nonce []byte) []byte {
	h := make([]byte, V1HeaderSize)
	copy(h[0:SaltSize], salt)
	binary.BigEndian.PutUint32(h[SaltSize:SaltSize+4], params.Time)
	binary.BigEndian.PutUint32(h[SaltSize+4:SaltSize+8], params.Memory)
	h[SaltSize+8] = params.Threads
	copy(h[SaltSize+9:], nonce)
	return h
}

// buildV2Header serializes a v2 header.
func buildV2Header(kdf KDFMode, salt []byte, params ArgonParams, nonce []byte) []byte {
	h := make([]byte, V2HeaderSize)
	off := copy(h, V2Magic)
	h[off] = 2
	off++
	h[off] = byte(kdf)
	off++
	copy(h[off:off+SaltSize], salt)
	off += SaltSize
	binary.BigEndian.PutUint32(h[off:off+4], params.Time)
	binary.BigEndian.PutUint32(h[off+4:off+8], params.Memory)
	h[off+8] = params.Threads
	off += 9
	copy(h[off:off+NonceSize], nonce)
	return h
}
