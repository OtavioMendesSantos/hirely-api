package security

import "testing"

func TestHashAndVerifyPassword_WithPepper(t *testing.T) {
	hash, err := HashPassword("password123", "test-pepper")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !VerifyPassword(string(hash), "password123", "test-pepper") {
		t.Error("expected matching password to verify")
	}
	if VerifyPassword(string(hash), "wrong-password", "test-pepper") {
		t.Error("expected wrong password not to verify")
	}
	if VerifyPassword(string(hash), "password123", "wrong-pepper") {
		t.Error("expected password with wrong pepper not to verify")
	}
}

func TestHashPassword_DifferentPepersProduceDifferentHashes(t *testing.T) {
	hashA, err := HashPassword("password123", "pepper-a")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	hashB, err := HashPassword("password123", "pepper-b")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(hashA) == string(hashB) {
		t.Error("expected different peppers to produce different hashes")
	}
}

func TestHashAndVerifyPassword_NoPepper(t *testing.T) {
	hash, err := HashPassword("password123", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !VerifyPassword(string(hash), "password123", "") {
		t.Error("expected matching password to verify")
	}
	if VerifyPassword(string(hash), "password123", "some-pepper") {
		t.Error("expected password with pepper not to verify against unpeppered hash")
	}
}
