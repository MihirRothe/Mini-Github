package auth

import (
	"testing"
)

func TestArgon2idHashingAndVerification(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	// Use faster parameters for tests
	testParams := &HashParams{
		Memory:      16 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	hash, err := HashPasswordWithParams(password, testParams)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatalf("generated empty hash")
	}

	// Verify correct password
	matched, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("unexpected error during verification: %v", err)
	}
	if !matched {
		t.Errorf("expected password to match hash")
	}

	// Verify incorrect password
	wrongMatched, err := VerifyPassword("WrongPassword123!", hash)
	if err != nil {
		t.Fatalf("unexpected error verifying wrong password: %v", err)
	}
	if wrongMatched {
		t.Errorf("expected wrong password to NOT match")
	}

	// Test invalid hash format
	_, err = VerifyPassword(password, "invalid$hash$string")
	if err == nil {
		t.Errorf("expected error with invalid hash format")
	}
}
