package v1

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"golang.org/x/exp/ebnf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type operatorType string

const (
	setupExternalURLError = "external URL isn't setup yet, see https://www.bytebase.com/docs/get-started/install/external-url"

	comparatorTypeEqual        operatorType = "="
	comparatorTypeLess         operatorType = "<"
	comparatorTypeLessEqual    operatorType = "<="
	comparatorTypeGreater      operatorType = ">"
	comparatorTypeGreaterEqual operatorType = ">="
	comparatorTypeNotEqual     operatorType = "!="
)

var (
	resourceIDMatcher = regexp.MustCompile("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$")
	deletePatch       = true
	undeletePatch     = false
)

func isNumber(v string) (int, bool) {
	n, err := strconv.Atoi(v)
	if err == nil {
		return int(n), true
	}
	return 0, false
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

const filterExample = `project == "projects/abc"`

// getProjectFilter will parse the simple filter such as `project = "projects/abc"` to "projects/abc" .
func getProjectFilter(filter string) (string, error) {
	retErr := errors.Errorf("invalid filter %q, example %q", filter, filterExample)
	e, err := cel.NewEnv(cel.Variable("project", cel.StringType))
	if err != nil {
		return "", err
	}
	ast, issues := e.Compile(filter)
	if issues != nil {
		return "", status.Errorf(codes.InvalidArgument, issues.String())
	}
	expr := ast.Expr()
	if expr == nil {
		return "", retErr
	}
	callExpr := expr.GetCallExpr()
	if callExpr == nil {
		return "", retErr
	}
	if callExpr.Function != "_==_" {
		return "", retErr
	}
	if len(callExpr.Args) != 2 {
		return "", retErr
	}
	if callExpr.Args[0].GetIdentExpr() == nil || callExpr.Args[0].GetIdentExpr().Name != "project" {
		return "", retErr
	}
	constExpr := callExpr.Args[1].GetConstExpr()
	if constExpr == nil {
		return "", retErr
	}
	return constExpr.GetStringValue(), nil
}

// getEBNFTokens will parse the simple filter such as `project = "abc" | "def".` to {project: ["abc", "def"]} .
func getEBNFTokens(filter, filterKey string) ([]string, error) {
	grammar, err := ebnf.Parse("", strings.NewReader(filter))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid filter %q", filter)
	}
	productions, ok := grammar[filterKey]
	if !ok {
		return nil, nil
	}
	switch expr := productions.Expr.(type) {
	case *ebnf.Token:
		// filterKey = "abc".
		return []string{expr.String}, nil
	case ebnf.Alternative:
		// filterKey = "abc" | "def".
		var tokens []string
		for _, expr := range expr {
			token, ok := expr.(*ebnf.Token)
			if !ok {
				return nil, errors.Errorf("invalid filter %q", filter)
			}
			tokens = append(tokens, token.String)
		}
		return tokens, nil
	case *ebnf.Alternative:
		// filterKey = "abc" | "def".
		var tokens []string
		for _, expr := range *expr {
			token, ok := expr.(*ebnf.Token)
			if !ok {
				return nil, errors.Errorf("invalid filter %q", filter)
			}
			tokens = append(tokens, token.String)
		}
		return tokens, nil
	default:
		return nil, errors.Errorf("invalid filter %q", filter)
	}
}

type orderByKey struct {
	key      string
	isAscend bool
}

func parseOrderBy(orderBy string) ([]orderByKey, error) {
	if orderBy == "" {
		return nil, nil
	}

	var result []orderByKey
	re := regexp.MustCompile(`(\w+)\s*(asc|desc)?`)
	matches := re.FindAllStringSubmatch(orderBy, -1)
	for _, match := range matches {
		if len(match) > 3 {
			return nil, errors.Errorf("invalid order by %q", orderBy)
		}
		key := orderByKey{
			key:      match[1],
			isAscend: true,
		}
		if len(match) == 3 && match[2] == "desc" {
			key.isAscend = false
		}
		result = append(result, key)
	}
	return result, nil
}

type expression struct {
	key      string
	operator operatorType
	value    string
}

// parseFilter will parse the simple filter.
// TODO(rebelice): support more complex filter.
// Currently we support the following syntax:
//  1. for single expression:
//     i.   defined as `key comparator "val"`.
//     ii.  Comparator can be `=`, `!=`, `>`, `>=`, `<`, `<=`.
//     iii. If val doesn't contain space, we can omit the double quotes.
//  2. for multiple expressions:
//     i.  We only support && currently.
//     ii. defined as `key comparator "val" && key comparator "val" && ...`.
func parseFilter(filter string) ([]expression, error) {
	if filter == "" {
		return nil, nil
	}

	normalized, quotedString, err := normalizeFilter(filter)
	if err != nil {
		return nil, err
	}

	var result []expression
	nextStringPos := 0

	// Split the normalized filter by " && " to get the list of expressions.
	expressions := strings.Split(normalized, " && ")
	for _, expressionString := range expressions {
		expr, err := parseExpression(expressionString)
		if err != nil {
			return nil, err
		}
		if expr.value == "?" {
			if nextStringPos >= len(quotedString) {
				return nil, errors.Errorf("invalid filter %q", filter)
			}
			expr.value = quotedString[nextStringPos]
			nextStringPos++
		}
		result = append(result, expr)
	}

	return result, nil
}

func parseExpression(expr string) (expression, error) {
	// Split the expression by " " to get the key, comparator and val.
	re := regexp.MustCompile(`\s+`)
	words := re.Split(strings.TrimSpace(expr), -1)
	if len(words) != 3 {
		return expression{}, errors.Errorf("invalid expression %q", expr)
	}

	comparator, err := getComparatorType(words[1])
	if err != nil {
		return expression{}, err
	}

	return expression{
		key:      words[0],
		operator: comparator,
		value:    words[2],
	}, nil
}

func getComparatorType(op string) (operatorType, error) {
	switch op {
	case "=":
		return comparatorTypeEqual, nil
	case "!=":
		return comparatorTypeNotEqual, nil
	case ">":
		return comparatorTypeGreater, nil
	case ">=":
		return comparatorTypeGreaterEqual, nil
	case "<":
		return comparatorTypeLess, nil
	case "<=":
		return comparatorTypeLessEqual, nil
	default:
		return comparatorTypeEqual, errors.Errorf("invalid comparator %q", op)
	}
}

// normalizeFilter will replace all quoted string with ? and return the list of quoted strings.
func normalizeFilter(filter string) (string, []string, error) {
	var (
		normalizedFilter string
		quotedStrings    []string
	)
	inQuotes := false
	lastQuoteIndex := 0
	for i, s := range filter {
		if s == '"' {
			if inQuotes {
				quotedStrings = append(quotedStrings, filter[lastQuoteIndex+1:i])
				normalizedFilter += "?"
			} else {
				lastQuoteIndex = i
			}
			inQuotes = !inQuotes
		} else if !inQuotes {
			// If we are not in quotes, we need to normalize the filter.
			// We need to add space before and after the comparator.
			// For example, "a>b" should be normalized to "a > b".
			switch s {
			case '!':
				normalizedFilter += " "
				normalizedFilter += string(s)
			case '<', '>':
				normalizedFilter += " "
				normalizedFilter += string(s)
				if i+1 < len(filter) && filter[i+1] != '=' {
					normalizedFilter += " "
				}
			case '=':
				if i > 0 && (filter[i-1] != '!' && filter[i-1] != '<' && filter[i-1] != '>') {
					normalizedFilter += " "
				}
				normalizedFilter += string(s)
				normalizedFilter += " "
			default:
				normalizedFilter += string(s)
			}
		}
	}

	if inQuotes {
		return "", nil, errors.Errorf("invalid filter %q", filter)
	}

	return normalizedFilter, quotedStrings, nil
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
	case db.Redis:
		return v1pb.Engine_REDIS
	case db.Oracle:
		return v1pb.Engine_ORACLE
	case db.Spanner:
		return v1pb.Engine_SPANNER
	case db.MSSQL:
		return v1pb.Engine_MSSQL
	case db.Redshift:
		return v1pb.Engine_REDSHIFT
	case db.MariaDB:
		return v1pb.Engine_MARIADB
	case db.OceanBase:
		return v1pb.Engine_OCEANBASE
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
	case v1pb.Engine_REDIS:
		return db.Redis
	case v1pb.Engine_ORACLE:
		return db.Oracle
	case v1pb.Engine_SPANNER:
		return db.Spanner
	case v1pb.Engine_MSSQL:
		return db.MSSQL
	case v1pb.Engine_REDSHIFT:
		return db.Redshift
	case v1pb.Engine_MARIADB:
		return db.MariaDB
	case v1pb.Engine_OCEANBASE:
		return db.OceanBase
	}
	return db.UnknownType
}

func getPageToken(limit int, offset int) (string, error) {
	return marshalPageToken(&storepb.PageToken{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
}

func marshalPageToken(pageToken *storepb.PageToken) (string, error) {
	b, err := proto.Marshal(pageToken)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal page token")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func unmarshalPageToken(s string, pageToken *storepb.PageToken) error {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return errors.Wrapf(err, "failed to decode page token")
	}
	if err := proto.Unmarshal(b, pageToken); err != nil {
		return errors.Wrapf(err, "failed to unmarshal page token")
	}
	return nil
}
