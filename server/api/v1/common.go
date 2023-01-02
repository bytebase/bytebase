package v1

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/ebnf"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	resourceIDMatcher = regexp.MustCompile("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$")
	deletePatch       = true
	undeletePatch     = false
)

func convertDeletedToState(deleted bool) v1pb.State {
	if deleted {
		return v1pb.State_DELETED
	}
	return v1pb.State_ACTIVE
}

func isValidResourceID(resourceID string) bool {
	return resourceIDMatcher.MatchString(resourceID)
}

const filterExample = `project = "projects/abc".`

// getFilter will parse the simple filter such as `project = "abc".` to "project" and "abc" .
func getFilter(filter, filterKey string) (string, error) {
	retErr := errors.Errorf("invalid filter %q, example %q", filter, filterExample)
	grammar, err := ebnf.Parse("", strings.NewReader(filter))
	if err != nil {
		return "", retErr
	}
	if len(grammar) != 1 {
		return "", retErr
	}
	for key, production := range grammar {
		if filterKey != key {
			return "", errors.Errorf("support filter key %q only", filterKey)
		}
		token, ok := production.Expr.(*ebnf.Token)
		if !ok {
			return "", retErr
		}
		return token.String, nil
	}
	return "", retErr
}
