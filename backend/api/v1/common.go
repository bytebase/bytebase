package v1

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/ebnf"

	"github.com/bytebase/bytebase/backend/plugin/db"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	projectNamePrefix          = "projects/"
	environmentNamePrefix      = "environments/"
	instanceNamePrefix         = "instances/"
	policyNamePrefix           = "policies/"
	databaseIDPrefix           = "databases/"
	instanceRolePrefix         = "roles/"
	userNamePrefix             = "users/"
	identityProviderNamePrefix = "idps/"
	settingNamePrefix          = "settings/"
)

var (
	resourceIDMatcher = regexp.MustCompile("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$")
	deletePatch       = true
	undeletePatch     = false
)

func getProjectID(name string) (string, error) {
	tokens, err := getNameParentTokens(name, projectNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func getEnvironmentID(name string) (string, error) {
	tokens, err := getNameParentTokens(name, environmentNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func getEnvironmentInstanceID(name string) (string, string, error) {
	// the instance request should be environments/{environment-id}/instances/{instance-id}
	tokens, err := getNameParentTokens(name, environmentNamePrefix, instanceNamePrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

func getEnvironmentInstanceRoleID(name string) (string, string, string, error) {
	// the instance request should be environments/{environment-id}/instances/{instance-id}/roles/{role-name}
	tokens, err := getNameParentTokens(name, environmentNamePrefix, instanceNamePrefix, instanceRolePrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

func getEnvironmentInstanceDatabaseID(name string) (string, string, string, error) {
	// the instance request should be environments/{environment-id}/instances/{instance-id}/databases/{database-id}
	tokens, err := getNameParentTokens(name, environmentNamePrefix, instanceNamePrefix, databaseIDPrefix)
	if err != nil {
		return "", "", "", err
	}
	return tokens[0], tokens[1], tokens[2], nil
}

func getUserID(name string) (int, error) {
	tokens, err := getNameParentTokens(name, userNamePrefix)
	if err != nil {
		return 0, err
	}
	userID, err := strconv.Atoi(tokens[0])
	if err != nil {
		return 0, errors.Errorf("invalid user ID %q", tokens[0])
	}
	return userID, nil
}

func getSettingName(name string) (string, error) {
	token, err := getNameParentTokens(name, settingNamePrefix)
	if err != nil {
		return "", err
	}
	return token[0], nil
}

func getIdentityProviderID(name string) (string, error) {
	tokens, err := getNameParentTokens(name, identityProviderNamePrefix)
	if err != nil {
		return "", err
	}
	return tokens[0], nil
}

func trimSuffix(name, suffix string) (string, error) {
	if !strings.HasSuffix(name, suffix) {
		return "", errors.Errorf("invalid request %q with suffix %q", name, suffix)
	}
	return strings.TrimRight(name, suffix), nil
}

func getNameParentTokens(name string, tokenPrefixes ...string) ([]string, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2*len(tokenPrefixes) {
		return nil, errors.Errorf("invalid request %q", name)
	}

	var tokens []string
	for i, tokenPrefix := range tokenPrefixes {
		if fmt.Sprintf("%s/", parts[2*i]) != tokenPrefix {
			return nil, errors.Errorf("invalid prefix %q in request %q", tokenPrefix, name)
		}
		if parts[2*i+1] == "" {
			return nil, errors.Errorf("invalid request %q with empty prefix %q", name, tokenPrefix)
		}
		tokens = append(tokens, parts[2*i+1])
	}
	return tokens, nil
}

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

func convertToEngine(engine db.Type) v1pb.Engine {
	switch engine {
	case db.ClickHouse:
		return v1pb.Engine_CLICKHOUSE
	case db.MySQL:
		return v1pb.Engine_MYSQL
	case db.Postgres:
		return v1pb.Engine_POSTGRES
	case db.Snowflake:
		return v1pb.Engine_SNOWFLAKE
	case db.SQLite:
		return v1pb.Engine_SQLITE
	case db.TiDB:
		return v1pb.Engine_TIDB
	case db.MongoDB:
		return v1pb.Engine_MONGODB
	}
	return v1pb.Engine_ENGINE_UNSPECIFIED
}

func convertEngine(engine v1pb.Engine) db.Type {
	switch engine {
	case v1pb.Engine_CLICKHOUSE:
		return db.ClickHouse
	case v1pb.Engine_MYSQL:
		return db.MySQL
	case v1pb.Engine_POSTGRES:
		return db.Postgres
	case v1pb.Engine_SNOWFLAKE:
		return db.Snowflake
	case v1pb.Engine_SQLITE:
		return db.SQLite
	case v1pb.Engine_TIDB:
		return db.TiDB
	case v1pb.Engine_MONGODB:
		return db.MongoDB
	}
	return db.UnknownType
}
