package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestHashPassword_ProducesValidPHCString(t *testing.T) {
	hash, err := HashPassword("correcthorsebatterystaple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected PHC string starting with $argon2id$, got %q",
			hash[:min(len(hash), 20)])
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("expected 6 parts in PHC string, got %d: %q", len(parts), hash)
	}
}

func TestHashPassword_IsNonDeterministic(t *testing.T) {
	h1, _ := HashPassword("samepassword")
	h2, _ := HashPassword("samepassword")
	if h1 == h2 {
		t.Error("two hashes of the same password must differ (different salts)")
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if !CheckPassword("mypassword123", hash) {
		t.Error("CheckPassword returned false for correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("rightpassword")
	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	hash, _ := HashPassword("somepassword")
	if CheckPassword("", hash) {
		t.Error("CheckPassword returned true for empty password")
	}
}

func TestCheckPassword_GarbageHash(t *testing.T) {
	cases := []string{
		"",
		"notahash",
		"$bcrypt$not$argon2id",
		"$argon2id$only-four-parts$here$ok",
		// Wrong version number — must be rejected
		"$argon2id$v=99$m=65536,t=3,p=4$bm90YmFzZTY0$bm90YmFzZTY0",
	}
	for _, bad := range cases {
		if CheckPassword("anything", bad) {
			t.Errorf("CheckPassword returned true for garbage hash %q", bad)
		}
	}
}

func TestCheckPassword_OldParametersStillWork(t *testing.T) {
	// PHC encoding means CheckPassword reads params from the stored string,
	// not the current package-level constants.
	// Simulate a hash produced with lower params (fast for tests).
	old := buildHashWithParams(t, "oldpassword", 4*1024, 1, 1)

	if !CheckPassword("oldpassword", old) {
		t.Error("CheckPassword failed to verify hash produced with different parameters")
	}
	if CheckPassword("wrongpassword", old) {
		t.Error("CheckPassword returned true for wrong password against low-param hash")
	}
}

func TestCheckPassword_RejectsBcryptHash(t *testing.T) {
	bcryptHash := "$2a$12$somehashedstuffhere"
	if CheckPassword("anything", bcryptHash) {
		t.Error("CheckPassword must reject bcrypt hashes")
	}
}

// buildHashWithParams produces a valid PHC string with custom parameters.
// Exists only to test that CheckPassword correctly reads params from the hash
// rather than from the package-level constants.
func buildHashWithParams(
	t *testing.T,
	plain string,
	memory, iterations uint32,
	parallelism uint8,
) string {
	t.Helper()
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("salt generation failed: %v", err)
	}
	hash := argon2.IDKey([]byte(plain), salt, iterations, memory, parallelism, 32)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, iterations, parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
