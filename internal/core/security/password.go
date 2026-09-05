package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the plain password.
// When pepper is non-empty, the password is first keyed with it via
// HMAC-SHA256 so a leaked database cannot be attacked offline without the
// server-side secret. Keying before bcrypt also avoids the 72-byte truncation
// limit.
func HashPassword(plain, pepper string) ([]byte, error) {
	secret := []byte(plain)
	if pepper != "" {
		mac := hmac.New(sha256.New, []byte(pepper))
		_, _ = mac.Write([]byte(plain))
		secret = []byte(hex.EncodeToString(mac.Sum(nil)))
	}
	return bcrypt.GenerateFromPassword(secret, bcrypt.DefaultCost)
}

// VerifyPassword checks a plain password against a bcrypt hash, applying the
// same pepper keying as HashPassword when pepper is non-empty.
func VerifyPassword(hashed, plain, pepper string) bool {
	secret := []byte(plain)
	if pepper != "" {
		mac := hmac.New(sha256.New, []byte(pepper))
		_, _ = mac.Write([]byte(plain))
		secret = []byte(hex.EncodeToString(mac.Sum(nil)))
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), secret) == nil
}
