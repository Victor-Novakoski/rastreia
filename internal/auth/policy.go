package auth

import "strings"

const (
	MinPasswordLength = 10
	// bcrypt only uses the first 72 bytes and errors on longer input.
	MaxPasswordBytes = 72
)

// commonPasswords holds passwords that show up at the top of leaked lists
// and still pass the length rule.
var commonPasswords = map[string]bool{
	"1234567890": true, "0123456789": true, "0987654321": true, "1111111111": true,
	"0000000000": true, "1q2w3e4r5t": true, "qwertyuiop": true, "password12": true,
	"password123": true, "senha12345": true, "senha123456": true, "abcdefghij": true,
	"abc1234567": true, "a123456789": true, "123456789a": true, "qwerty1234": true,
	"iloveyou12": true, "admin12345": true, "administrator": true, "brasil1234": true,
	"mudar12345": true, "trocar1234": true, "1234567891": true, "12345678910": true,
}

// PasswordProblem explains why a password is too weak, or returns "" when it is fine.
func PasswordProblem(password string) string {
	if len([]rune(password)) < MinPasswordLength {
		return "must have at least 10 characters"
	}
	if len(password) > MaxPasswordBytes {
		return "must have at most 72 bytes"
	}
	if commonPasswords[strings.ToLower(password)] {
		return "is too common"
	}
	return ""
}
