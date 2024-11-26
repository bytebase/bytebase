package v1

import (
	"encoding/base64"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"golang.org/x/exp/ebnf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type OperatorType string

const (
	setupExternalURLError = "external URL isn't setup yet, see https://www.bytebase.com/docs/get-started/install/external-url"

	ComparatorTypeEqual        OperatorType = "="
	ComparatorTypeLess         OperatorType = "<"
	ComparatorTypeLessEqual    OperatorType = "<="
	ComparatorTypeGreater      OperatorType = ">"
	ComparatorTypeGreaterEqual OperatorType = ">="
	ComparatorTypeNotEqual     OperatorType = "!="
)

var (
	resourceIDMatcher = regexp.MustCompile("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$")
	// https://datatracker.ietf.org/doc/html/rfc4122#section-4.1
	uuidMatcher   = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
	deletePatch   = true
	undeletePatch = false
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
		return "", status.Error(codes.InvalidArgument, issues.String())
	}
	parsedExpr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return "", retErr
	}
	expr := parsedExpr.Expr
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

type Expression struct {
	Key      string
	Operator OperatorType
	Value    string
}

// ParseFilter will parse the simple filter.
// TODO(rebelice): support more complex filter.
// Currently we support the following syntax:
//  1. for single expression:
//     i.   defined as `key comparator "val"`.
//     ii.  Comparator can be `=`, `!=`, `>`, `>=`, `<`, `<=`.
//     iii. If val doesn't contain space, we can omit the double quotes.
//  2. for multiple expressions:
//     i.  We only support && currently.
//     ii. defined as `key comparator "val" && key comparator "val" && ...`.
func ParseFilter(filter string) ([]Expression, error) {
	if filter == "" {
		return nil, nil
	}

	normalized, quotedString, err := normalizeFilter(filter)
	if err != nil {
		return nil, err
	}

	var result []Expression
	nextStringPos := 0

	// Split the normalized filter by " && " to get the list of expressions.
	expressions := strings.Split(normalized, " && ")
	for _, expressionString := range expressions {
		expr, err := parseExpression(expressionString)
		if err != nil {
			return nil, err
		}
		if expr.Value == "?" {
			if nextStringPos >= len(quotedString) {
				return nil, errors.Errorf("invalid filter %q", filter)
			}
			expr.Value = quotedString[nextStringPos]
			nextStringPos++
		}
		result = append(result, expr)
	}

	return result, nil
}

func parseExpression(expr string) (Expression, error) {
	// Split the expression by " " to get the key, comparator and val.
	re := regexp.MustCompile(`\s+`)
	words := re.Split(strings.TrimSpace(expr), -1)
	if len(words) != 3 {
		return Expression{}, errors.Errorf("invalid expression %q", expr)
	}

	comparator, err := getComparatorType(words[1])
	if err != nil {
		return Expression{}, err
	}

	return Expression{
		Key:      words[0],
		Operator: comparator,
		Value:    words[2],
	}, nil
}

func getComparatorType(op string) (OperatorType, error) {
	switch op {
	case "=", "eq":
		return ComparatorTypeEqual, nil
	case "!=":
		return ComparatorTypeNotEqual, nil
	case ">":
		return ComparatorTypeGreater, nil
	case ">=":
		return ComparatorTypeGreaterEqual, nil
	case "<":
		return ComparatorTypeLess, nil
	case "<=":
		return ComparatorTypeLessEqual, nil
	default:
		return ComparatorTypeEqual, errors.Errorf("invalid comparator %q", op)
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

func convertToEngine(engine storepb.Engine) v1pb.Engine {
	switch engine {
	case storepb.Engine_CLICKHOUSE:
		return v1pb.Engine_CLICKHOUSE
	case storepb.Engine_MYSQL:
		return v1pb.Engine_MYSQL
	case storepb.Engine_POSTGRES:
		return v1pb.Engine_POSTGRES
	case storepb.Engine_SNOWFLAKE:
		return v1pb.Engine_SNOWFLAKE
	case storepb.Engine_SQLITE:
		return v1pb.Engine_SQLITE
	case storepb.Engine_TIDB:
		return v1pb.Engine_TIDB
	case storepb.Engine_MONGODB:
		return v1pb.Engine_MONGODB
	case storepb.Engine_REDIS:
		return v1pb.Engine_REDIS
	case storepb.Engine_ORACLE:
		return v1pb.Engine_ORACLE
	case storepb.Engine_SPANNER:
		return v1pb.Engine_SPANNER
	case storepb.Engine_MSSQL:
		return v1pb.Engine_MSSQL
	case storepb.Engine_REDSHIFT:
		return v1pb.Engine_REDSHIFT
	case storepb.Engine_MARIADB:
		return v1pb.Engine_MARIADB
	case storepb.Engine_OCEANBASE:
		return v1pb.Engine_OCEANBASE
	case storepb.Engine_OCEANBASE_ORACLE:
		return v1pb.Engine_OCEANBASE_ORACLE
	case storepb.Engine_DM:
		return v1pb.Engine_DM
	case storepb.Engine_RISINGWAVE:
		return v1pb.Engine_RISINGWAVE
	case storepb.Engine_STARROCKS:
		return v1pb.Engine_STARROCKS
	case storepb.Engine_DORIS:
		return v1pb.Engine_DORIS
	case storepb.Engine_HIVE:
		return v1pb.Engine_HIVE
	case storepb.Engine_ELASTICSEARCH:
		return v1pb.Engine_ELASTICSEARCH
	case storepb.Engine_BIGQUERY:
		return v1pb.Engine_BIGQUERY
	case storepb.Engine_DYNAMODB:
		return v1pb.Engine_DYNAMODB
	case storepb.Engine_DATABRICKS:
		return v1pb.Engine_DATABRICKS
	case storepb.Engine_COCKROACHDB:
		return v1pb.Engine_COCKROACHDB
	}
	return v1pb.Engine_ENGINE_UNSPECIFIED
}

func convertEngine(engine v1pb.Engine) storepb.Engine {
	switch engine {
	case v1pb.Engine_CLICKHOUSE:
		return storepb.Engine_CLICKHOUSE
	case v1pb.Engine_MYSQL:
		return storepb.Engine_MYSQL
	case v1pb.Engine_POSTGRES:
		return storepb.Engine_POSTGRES
	case v1pb.Engine_SNOWFLAKE:
		return storepb.Engine_SNOWFLAKE
	case v1pb.Engine_SQLITE:
		return storepb.Engine_SQLITE
	case v1pb.Engine_TIDB:
		return storepb.Engine_TIDB
	case v1pb.Engine_MONGODB:
		return storepb.Engine_MONGODB
	case v1pb.Engine_REDIS:
		return storepb.Engine_REDIS
	case v1pb.Engine_ORACLE:
		return storepb.Engine_ORACLE
	case v1pb.Engine_SPANNER:
		return storepb.Engine_SPANNER
	case v1pb.Engine_MSSQL:
		return storepb.Engine_MSSQL
	case v1pb.Engine_REDSHIFT:
		return storepb.Engine_REDSHIFT
	case v1pb.Engine_MARIADB:
		return storepb.Engine_MARIADB
	case v1pb.Engine_OCEANBASE:
		return storepb.Engine_OCEANBASE
	case v1pb.Engine_OCEANBASE_ORACLE:
		return storepb.Engine_OCEANBASE_ORACLE
	case v1pb.Engine_DM:
		return storepb.Engine_DM
	case v1pb.Engine_RISINGWAVE:
		return storepb.Engine_RISINGWAVE
	case v1pb.Engine_STARROCKS:
		return storepb.Engine_STARROCKS
	case v1pb.Engine_DORIS:
		return storepb.Engine_DORIS
	case v1pb.Engine_HIVE:
		return storepb.Engine_HIVE
	case v1pb.Engine_ELASTICSEARCH:
		return storepb.Engine_ELASTICSEARCH
	case v1pb.Engine_BIGQUERY:
		return storepb.Engine_BIGQUERY
	case v1pb.Engine_DYNAMODB:
		return storepb.Engine_DYNAMODB
	case v1pb.Engine_DATABRICKS:
		return storepb.Engine_DATABRICKS
	case v1pb.Engine_COCKROACHDB:
		return storepb.Engine_COCKROACHDB
	}
	return storepb.Engine_ENGINE_UNSPECIFIED
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

type pageSize struct {
	token   string
	limit   int
	maximum int
}

type pageOffset struct {
	limit  int
	offset int
}

func (p *pageOffset) getNextPageToken() (string, error) {
	return marshalPageToken(&storepb.PageToken{
		Limit:  int32(p.limit),
		Offset: int32(p.offset + p.limit),
	})
}

func parseLimitAndOffset(size *pageSize) (*pageOffset, error) {
	offset := &pageOffset{}
	if size.token != "" {
		var token storepb.PageToken
		if err := unmarshalPageToken(size.token, &token); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", err)
		}
		if token.Limit < 0 {
			return nil, status.Errorf(codes.InvalidArgument, "page size cannot be negative")
		}
		offset.limit = int(token.Limit)
		offset.offset = int(token.Offset)
	} else {
		offset.limit = int(size.limit)
	}
	if offset.limit <= 0 {
		offset.limit = 10
	}
	if offset.limit > size.maximum {
		offset.limit = size.maximum
	}
	return offset, nil
}

// isValidUUID validates that the id is the valid UUID format.
// https://datatracker.ietf.org/doc/html/rfc4122#section-4.1
func isValidUUID(id string) bool {
	return uuidMatcher.MatchString(id)
}

func convertExportFormat(format storepb.ExportFormat) v1pb.ExportFormat {
	switch format {
	case storepb.ExportFormat_CSV:
		return v1pb.ExportFormat_CSV
	case storepb.ExportFormat_JSON:
		return v1pb.ExportFormat_JSON
	case storepb.ExportFormat_SQL:
		return v1pb.ExportFormat_SQL
	case storepb.ExportFormat_XLSX:
		return v1pb.ExportFormat_XLSX
	}
	return v1pb.ExportFormat_FORMAT_UNSPECIFIED
}

func convertToExportFormat(format v1pb.ExportFormat) storepb.ExportFormat {
	switch format {
	case v1pb.ExportFormat_CSV:
		return storepb.ExportFormat_CSV
	case v1pb.ExportFormat_JSON:
		return storepb.ExportFormat_JSON
	case v1pb.ExportFormat_SQL:
		return storepb.ExportFormat_SQL
	case v1pb.ExportFormat_XLSX:
		return storepb.ExportFormat_XLSX
	}
	return storepb.ExportFormat_FORMAT_UNSPECIFIED
}
