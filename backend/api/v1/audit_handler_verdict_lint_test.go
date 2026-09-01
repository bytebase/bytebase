package v1

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	iamCheckPattern    = regexp.MustCompile(`\.Check\w*Permission\(`)
	negativeResult     = regexp.MustCompile(`if !\w+ \{`)
	permissionDeniedRe = regexp.MustCompile(`connect\.CodePermissionDenied`)
)

// TestLintHandlerIAMVerdictsAreMarked holds the half of the audit rule that no
// interceptor can hold for itself.
//
// A method annotated auth_method = CUSTOM passes the ACL interceptor —
// doIAMPermissionCheck returns true for every non-IAM method — and makes its
// own IAM verdict inside the handler. That verdict is the same one ACL would
// have made, so it has to mark the same way: 22 of those methods declare
// audit = DENIALS, and an unmarked verdict makes that declaration false, which
// is the exact defect BYT-10124 is about.
//
// The predicate is deliberately narrow. It fires only where a handler calls
// iamManager.CheckPermission (or CheckProjectWidePermission), branches on the
// negative result, and answers CodePermissionDenied — the mechanical shape of
// an authorization verdict about the caller. Licence gates, workflow rules and
// "only the assignee may do this" refusals answer the same code and are NOT
// caught here, because they are not verdicts about a permission the caller
// holds. If one of those should be audited, the method's annotation is where
// to say so.
func TestLintHandlerIAMVerdictsAreMarked(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var unmarked []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// acl.go is the interceptor this lint exists to make handlers match.
		if name == "acl.go" {
			continue
		}
		content, err := os.ReadFile(filepath.Clean(name))
		require.NoError(t, err)
		lines := strings.Split(string(content), "\n")

		for i, line := range lines {
			if !permissionDeniedRe.MatchString(line) {
				continue
			}
			before := strings.Join(lines[max(0, i-14):i], "\n")
			if !iamCheckPattern.MatchString(before) || !negativeResult.MatchString(before) {
				continue
			}
			// The mark sits either on this line or on the return that follows
			// the error's detail being attached.
			after := strings.Join(lines[i:min(i+12, len(lines))], "\n")
			if !strings.Contains(after, "markPolicyDenied") {
				unmarked = append(unmarked, name+":"+strconv.Itoa(i+1))
			}
		}
	}

	slices.Sort(unmarked)
	require.Empty(t, unmarked,
		"a handler that makes the ACL interceptor's own IAM verdict must call markPolicyDenied, "+
			"or the method's audit = DENIALS annotation is false for the caller it just refused")
}
