//nolint:revive
package common

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nyaruka/phonenumbers/v2"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// MaxSheetSize is the maximum size (2M) of a sheet for displaying.
	MaxSheetSize = 2 * 1024 * 1024
	// MaxSheetCheckSize is the maximum size of a sheet for checking changes.
	MaxSheetCheckSize = 2 * 1024 * 1024
	// The maximum number of bytes for sql results in response body.
	// 100 MB.
	DefaultMaximumSQLResultSize = int64(100 * 1024 * 1024)
	// MaximumCommands is the maximum number of commands that can be executed in a single transaction.
	MaximumCommands = 200
	// MaximumAdvicePerStatus is the maximum number of advice that can be returned per status.
	MaximumAdvicePerStatus = 50
	MaximumLintExplainSize = 10
)

// RoundRows rounds a planner's row estimate to int64, saturating at math.MaxInt64 instead of
// overflowing to a negative count.
func RoundRows(rows float64) int64 {
	if rows >= math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(math.Round(rows))
}

// AddRows adds row counts, saturating at math.MaxInt64 instead of wrapping negative.
func AddRows(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// ProtojsonMarshaler is a global protojson marshaler with DiscardUnknown set to true.
//
//nolint:forbidigo
var ProtojsonUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}

// RandomString returns a random string with length n.
func RandomString(n int) (string, error) {
	var sb strings.Builder
	sb.Grow(n)
	for i := 0; i < n; i++ {
		// The reason for using crypto/rand instead of math/rand is that
		// the former relies on hardware to generate random numbers and
		// thus has a stronger source of random numbers.
		randNum, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		if _, err := sb.WriteRune(letters[randNum.Uint64()]); err != nil {
			return "", err
		}
	}
	return sb.String(), nil
}

// TruncateString truncates the string to have a maximum length of `limit` characters.
func TruncateString(str string, limit int) (string, bool) {
	chars := 0
	// The string may contain unicode characters, so we iterate here.
	for i := range str {
		if chars >= limit {
			return str[:i], true
		}
		chars++
	}
	return str, false
}

// ValidatePhone validates the phone number.
func ValidatePhone(phone string) error {
	phoneNumber, err := phonenumbers.Parse(phone, "")
	if err != nil {
		return err
	}
	if !phonenumbers.IsValidNumber(phoneNumber) {
		return errors.New("invalid phone number")
	}
	return nil
}

// emailRegex is based on the WHATWG HTML spec for valid email addresses.
// https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
// Modified to only allow lowercase letters since Bytebase requires lowercase emails.
var emailRegex = regexp.MustCompile(`^[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)

// ValidateEmail validates the email address using WHATWG HTML spec.
// Only lowercase ASCII letters, digits, and standard email symbols are allowed.
func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email address")
	}
	return nil
}

// IsValidEmail returns true if the email is valid per WHATWG HTML spec.
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// SanitizeUTF8String returns a copy of the string s with each run of invalid or unprintable UTF-8 byte sequences
// replaced by its hexadecimal representation string.
func SanitizeUTF8String(s string) string {
	var b strings.Builder

	for i, c := range s {
		if c != utf8.RuneError {
			continue
		}

		_, wid := utf8.DecodeRuneInString(s[i:])
		if wid == 1 {
			b.Grow(len(s))
			_, _ = b.WriteString(s[:i])
			s = s[i:]
			break
		}
	}

	// Fast path for unchanged input
	if b.Cap() == 0 { // didn't call b.Grow above
		return s
	}

	for i := 0; i < len(s); {
		c := s[i]
		// U+0000-U+0019 are control characters
		if 0x20 <= c && c < utf8.RuneSelf {
			i++
			_ = b.WriteByte(c)
			continue
		}
		_, wid := utf8.DecodeRuneInString(s[i:])
		if wid == 1 {
			i++
			_, _ = fmt.Fprintf(&b, "\\x%02x", c)
			continue
		}
		_, _ = b.WriteString(s[i : i+wid])
		i += wid
	}

	return b.String()
}

func IsNil(val any) bool {
	if val == nil {
		return true
	}

	v := reflect.ValueOf(val)
	k := v.Kind()
	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		// Other types cannot be nil
	}

	return false
}

// Uniq returns a new slice with duplicate elements removed, preserving order.
func Uniq[T comparable](array []T) []T {
	res := make([]T, 0, len(array))
	seen := make(map[T]struct{}, len(array))

	for _, e := range array {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		res = append(res, e)
	}

	return res
}
