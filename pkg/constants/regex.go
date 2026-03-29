package constants

import (
	"errors"
	"regexp"
)

var (
	// Password components (Go-compatible)
	hasUpper   = regexp.MustCompile(`[A-Z]`)
	hasLower   = regexp.MustCompile(`[a-z]`)
	hasNumber  = regexp.MustCompile(`\d`)
	hasSpecial = regexp.MustCompile(`[!@#$%^&*()_+{}|:<>?~-]`)

	// Standard Regexes
	PHONE_NUMBER_REGEX     = regexp.MustCompile(`^\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}$`)
	EMAIL_REGEX            = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	YOUTUBE_VIDEO_ID_REGEX = regexp.MustCompile(`(?:\/|v=|\/v\/|embed\/|watch\?v=|watch\?.+&v=)([\w-]{11})`)
	BANK_INPUT             = regexp.MustCompile(`[_＿]{2,}`)
)

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if !hasUpper.MatchString(password) {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower.MatchString(password) {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber.MatchString(password) {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial.MatchString(password) {
		return errors.New("password must contain at least one special character")
	}
	return nil
}