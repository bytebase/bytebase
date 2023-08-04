package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"
	"github.com/pkg/errors"
)

const (
	// MaxSheetSize is the maximum size (1M) of a sheet for displaying.
	MaxSheetSize = 1024 * 1024
	// MaxSheetSizeForTaskCheck is the maximum size of a sheet for task check to run.
	MaxSheetSizeForTaskCheck = 10 * 1024 * 1024
	// MaxSheetSizeForRollback is the maximum size of a sheet for rollback generator to run.
	MaxSheetSizeForRollback = 8 * 1024 * 1024

	// ExternalURLPlaceholder is the docs link to configure --external-url.
	ExternalURLPlaceholder = "https://www.bytebase.com/docs/get-started/install/external-url"
)

// FindString returns the search index of sorted strings.
func FindString(stringList []string, search string) int {
	sort.Strings(stringList)
	i := sort.SearchStrings(stringList, search)
	if i == len(stringList) {
		return -1
	}
	return i
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

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

// GetPostgresDataDir returns the postgres data directory of Bytebase.
func GetPostgresDataDir(dataDir string, demoName string) string {
	// If demo is specified, we will use demo specific directory to store the demo data. Because
	// we reset the demo data when starting Bytebase and this can prevent accidentally removing the
	// production data.
	if demoName != "" {
		return path.Join(dataDir, "pgdata-demo", demoName)
	}
	return path.Join(dataDir, "pgdata")
}

// GetPostgresSampleDataDir returns the data directory of postgres sample instance.
func GetPostgresSampleDataDir(dataDir string, subDir string) string {
	return path.Join(dataDir, "pgdata-sample", subDir)
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
// We cannot add task ID because tenant mode databases should use the same migration version string when applying a schema update.
func DefaultMigrationVersion() string {
	return time.Now().Format("20060102150405")
}

// ParseTemplateTokens parses the template and returns template tokens and their delimiters.
// For example, if the template is "{{DB_NAME}}_hello_{{LOCATION}}", then the tokens will be ["{{DB_NAME}}", "{{LOCATION}}"],
// and the delimiters will be ["_hello_"].
// The caller will usually replace the tokens with a normal string, or a regexp. In the latter case, it will be a problem
// if there are special regexp characters such as "$" in the delimiters. The caller should escape the delimiters in such cases.
func ParseTemplateTokens(template string) ([]string, []string) {
	r := regexp.MustCompile(`{{[^{}]+}}`)
	tokens := r.FindAllString(template, -1)
	if len(tokens) > 0 {
		split := r.Split(template, -1)
		var delimiters []string
		for _, s := range split {
			if s != "" {
				delimiters = append(delimiters, s)
			}
		}
		return tokens, delimiters
	}
	return nil, nil
}

// GetFileSizeSum calculates the sum of file sizes for file names in the list.
func GetFileSizeSum(fileNameList []string) (int64, error) {
	var sum int64
	for _, fileName := range fileNameList {
		stat, err := os.Stat(fileName)
		if err != nil {
			return 0, err
		}
		sum += stat.Size()
	}
	return sum, nil
}

// GetBinlogRelativeDir composes the relative directory for binlog.
// It's useful to convert a local absolute binlog directory path to the cloud path.
func GetBinlogRelativeDir(binlogDir string) string {
	instanceID := filepath.Base(binlogDir)
	return filepath.Join("backup", "instance", instanceID)
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
