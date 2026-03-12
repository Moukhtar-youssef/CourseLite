package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Raise memory or iterations if your server can afford it.
// The encoded string stores all parameters

const (
	argonMemory     = 64 * 1024
	argonIterations = 3
	argonKeyLen     = 32
	argonSaltLen    = 16
)

var argonParallelism = uint8(runtime.NumCPU()) //nolint:gochecknoglobals

var argonSem = make(chan struct{}, runtime.NumCPU()) //nolint:gochecknoglobals

var (
	ErrInvalidHash         = errors.New("password: hash format is invalid")
	ErrIncompatibleVersion = errors.New("password: incompatible argon2 version")
)

func HashPassword(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: failed to generate salt: %w", err)
	}

	argonSem <- struct{}{}
	defer func() { <-argonSem }()

	hash := argon2.IDKey(
		[]byte(plain),
		salt,
		argonIterations,
		argonMemory,
		argonParallelism,
		argonKeyLen,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// CheckPassword verifies plain against an Argon2id PHC string produced by
// HashPassword. It reads all parameters (memory, iterations, parallelism)
// directly from the stored hash, so old hashes stay verifiable even after
// you tune the parameters for future passwords.
func CheckPassword(plain, encoded string) bool {
	p, salt, expectedHash, err := decodeHash(encoded)
	if err != nil {
		return false
	}

	argonSem <- struct{}{}
	defer func() { <-argonSem }()

	actualHash := argon2.IDKey(
		[]byte(plain),
		salt,
		p.iterations,
		p.memory,
		p.parallelism,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

// Format: $argon2id$v=<v>$m=<m>,t=<t>,p=<p>$<salt_b64>$<hash_b64>
func decodeHash(encoded string) (p argonParams, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	// parts[0] is empty (string starts with $), so valid split has 6 parts.
	if len(parts) != 6 {
		return p, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return p, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return p, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return p, nil, nil, ErrIncompatibleVersion
	}

	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&p.memory, &p.iterations, &p.parallelism); err != nil {
		return p, nil, nil, ErrInvalidHash
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return p, nil, nil, ErrInvalidHash
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return p, nil, nil, ErrInvalidHash
	}

	return p, salt, hash, nil
}
