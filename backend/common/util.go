package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nyaruka/phonenumbers"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// MaxSheetSize is the maximum size (2M) of a sheet for displaying.
	MaxSheetSize = 2 * 1024 * 1024
	// MaxSheetCheckSize is the maximum size of a sheet for checking changes.
	MaxSheetCheckSize = 2 * 1024 * 1024
	// The maximum number of bytes for sql results in response body.
	// 100 MB.
	DefaultMaximumSQLResultSize = 100 * 1024 * 1024
	// MaximumCommands is the maximum number of commands that can be executed in a single transaction.
	MaximumCommands = 200
	// MaximumAdvicePerStatus is the maximum number of advice that can be returned per status.
	MaximumAdvicePerStatus = 50
	MaximumLintExplainSize = 10

	// MinimumCompletedPlanCheckRun is the minimum number required to generate approval flow.
	MinimumCompletedPlanCheckRun = 5

	// ExternalURLPlaceholder is the docs link to configure --external-url.
	ExternalURLPlaceholder = "https://www.bytebase.com/docs/get-started/install/external-url"
)

var (
	StatementAdviseEngines = map[storepb.Engine]bool{
		storepb.Engine_MYSQL:            true,
		storepb.Engine_TIDB:             true,
		storepb.Engine_POSTGRES:         true,
		storepb.Engine_ORACLE:           true,
		storepb.Engine_OCEANBASE_ORACLE: true,
		storepb.Engine_OCEANBASE:        true,
		storepb.Engine_SNOWFLAKE:        true,
		storepb.Engine_MSSQL:            true,
		storepb.Engine_DYNAMODB:         true,
		storepb.Engine_COCKROACHDB:      true,
	}
	StatementReportEngines = map[storepb.Engine]bool{
		storepb.Engine_POSTGRES:         true,
		storepb.Engine_MYSQL:            true,
		storepb.Engine_TIDB:             true,
		storepb.Engine_OCEANBASE:        true,
		storepb.Engine_ORACLE:           true,
		storepb.Engine_OCEANBASE_ORACLE: true,
		storepb.Engine_MSSQL:            true,
	}
)

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

// HasPrefixes returns true if the string s has any of the given prefixes.
func HasPrefixes(src string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(src, prefix) {
			return true
		}
	}
	return false
}

// GetPostgresSocketDir returns the postgres socket directory of Bytebase.
func GetPostgresSocketDir() string {
	return "/tmp"
}

// GetResourceDir returns the resource directory of Bytebase.
func GetResourceDir(dataDir string) string {
	return path.Join(dataDir, "resources")
}

// DefaultMigrationVersion returns the default migration version string.
// Use the current time in second to guarantee uniqueness in a monotonic increasing way.
func DefaultMigrationVersion() model.Version {
	return model.Version{Version: time.Now().Format("20060102150405")}
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

// TruncateStringWithDescription tries to truncate the string and append "... (view details in Bytebase)" if truncated.
func TruncateStringWithDescription(str string) string {
	const limit = 450
	if truncatedStr, truncated := TruncateString(str, limit); truncated {
		return fmt.Sprintf("%s... (view details in Bytebase)", truncatedStr)
	}
	return str
}

// GetBinlogAbsDir gets the binary log directory for an instance.
func GetBinlogAbsDir(dataDir string, instanceID int) string {
	return filepath.Join(dataDir, "backup", "instance", fmt.Sprintf("%d", instanceID))
}

// Obfuscate obfuscates a string with a seed string.
func Obfuscate(src, seed string) string {
	srcBytes, seedBytes := []byte(src), []byte(seed)
	obfuscated := make([]byte, len(srcBytes))
	for i, b := range srcBytes {
		obfuscated[i] = b ^ seedBytes[i%len(seedBytes)]
	}
	return base64.StdEncoding.EncodeToString(obfuscated)
}

// Unobfuscate unobfuscates a string with a seed string.
func Unobfuscate(dst, seed string) (string, error) {
	obfuscated, err := base64.StdEncoding.DecodeString(dst)
	if err != nil {
		return "", err
	}
	unobfuscated, seedBytes := make([]byte, len(obfuscated)), []byte(seed)
	for i, b := range obfuscated {
		unobfuscated[i] = b ^ seedBytes[i%len(seedBytes)]
	}
	return string(unobfuscated), nil
}

// NormalizeExternalURL will format the external url.
func NormalizeExternalURL(url string) (string, error) {
	r := strings.TrimSpace(url)
	r = strings.TrimSuffix(r, "/")
	if !HasPrefixes(r, "http://", "https://") {
		return "", errors.Errorf("%s must start with http:// or https://", url)
	}
	parts := strings.Split(r, ":")
	if len(parts) > 3 {
		return "", errors.Errorf("%s malformed", url)
	}
	if len(parts) == 3 {
		port, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", errors.Errorf("%s has non integer port", url)
		}
		// The external URL is used as the redirectURL in the get token process of OAuth, and the
		// RedirectURL needs to be consistent with the RedirectURL in the get code process.
		// The frontend gets it through window.location.origin in the get code
		// process, so port 80/443 need to be cropped.
		if port == 80 || port == 443 {
			r = strings.Join(parts[0:2], ":")
		}
	}
	return r, nil
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
			_, _ = b.WriteString(fmt.Sprintf("\\x%02x", c))
			continue
		}
		_, _ = b.WriteString(s[i : i+wid])
		i += wid
	}

	return b.String()
}

func FormatMaximumSQLResultSizeMessage(limit int64) string {
	return fmt.Sprintf("Output of query exceeds max allowed output size of %dMB", limit/1024/1024)
}
