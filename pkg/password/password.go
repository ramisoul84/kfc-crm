package password

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// Bounds for bcrypt (72 is bcrypt's hard limit).
	MinLength = 8
	MaxLength = 72

	// bcrypt cost — 12 is the recommended production value.
	bcryptCost = 12

	lowercase = "abcdefghijklmnopqrstuvwxyz"
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits    = "0123456789"
	special   = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	allChars  = lowercase + uppercase + digits + special
)

// HashPassword hashes a plain password using bcrypt.
func HashPassword(plain string) (string, error) {
	if err := ValidateStrength(plain); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a plain password with its bcrypt hash.
func CheckPassword(plain, hash string) error {
	if plain == "" || hash == "" {
		return fmt.Errorf("password and hash are required")
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("invalid password")
		}
		return fmt.Errorf("failed to compare password: %w", err)
	}
	return nil
}

// GenerateRandomPassword returns a random password of the given length
// that satisfies all strength rules.
func GenerateRandomPassword(length int) (string, error) {
	if length < MinLength {
		length = 16
	}
	if length > MaxLength {
		length = MaxLength
	}

	buf := make([]byte, length)

	// Guarantee one character from each set.
	required := []string{lowercase, uppercase, digits, special}
	for i, set := range required {
		c, err := randomChar(set)
		if err != nil {
			return "", err
		}
		buf[i] = c
	}

	// Fill the rest from the combined set.
	for i := len(required); i < length; i++ {
		c, err := randomChar(allChars)
		if err != nil {
			return "", err
		}
		buf[i] = c
	}

	// Shuffle so positions aren't predictable.
	if err := shuffle(buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

// GenerateSecureToken returns a URL-safe random token of the given length
// (in bytes; the output is longer after base64).
func GenerateSecureToken(length int) (string, error) {
	if length < 32 {
		length = 32
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateStrength enforces the password policy.
func ValidateStrength(p string) error {
	if len(p) < MinLength {
		return fmt.Errorf("password must be at least %d characters", MinLength)
	}
	if len(p) > MaxLength {
		return fmt.Errorf("password must be at most %d characters", MaxLength)
	}

	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range p {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	switch {
	case !hasLower:
		return fmt.Errorf("password must contain a lowercase letter")
	case !hasUpper:
		return fmt.Errorf("password must contain an uppercase letter")
	case !hasDigit:
		return fmt.Errorf("password must contain a digit")
	case !hasSpecial:
		return fmt.Errorf("password must contain a special character")
	}

	return nil
}

// IsStrong reports whether the password satisfies the policy.
func IsStrong(p string) bool {
	return ValidateStrength(p) == nil
}

// ─────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────

func randomChar(set string) (byte, error) {
	if set == "" {
		return 0, fmt.Errorf("charset is empty")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, fmt.Errorf("random int: %w", err)
	}
	return set[n.Int64()], nil
}

func shuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("shuffle: %w", err)
		}
		b[i], b[j.Int64()] = b[j.Int64()], b[i]
	}
	return nil
}
