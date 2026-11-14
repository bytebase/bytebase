package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*StatementObjectOwnerCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK, &StatementObjectOwnerCheckAdvisor{})
}

// StatementObjectOwnerCheckAdvisor checks that statements change only
// objects the current role owns. This adapter reads the current role from the
// database; omni (pg/v1review.StatementObjectOwnerCheck) judges the statements
// against the synced schema's owners.
type StatementObjectOwnerCheckAdvisor struct {
}

func (*StatementObjectOwnerCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	currentRole := checkCtx.DBSchema.Owner
	if !checkCtx.TenantMode {
		currentRole, err = getCurrentUser(ctx, checkCtx.Driver)
		if err != nil {
			slog.Debug("Failed to get current user", log.BBError(err))
			currentRole = checkCtx.DBSchema.Owner
		}
	}
	schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := pgparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range pgv1review.StatementObjectOwnerCheck([]ast.Node{node}, schema, currentRole) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: fmt.Sprintf("%s \"%s\" is owned by \"%s\", but the current role is \"%s\".", f.Name, f.Table, f.Value, currentRole),
				Code:    code.StatementObjectOwnerCheck.Int32(),
				StartPosition: &storepb.Position{
					Line:   (&BaseRule{StmtText: stmt.Text}).ContentStartLine() + int32(stmt.BaseLine()),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}

// extractSchemaFromFuncname extracts the schema name from a function's qualified name list.
func extractSchemaFromFuncname(funcname *ast.List) string {
	parts := listStrings(funcname)
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

func getCurrentUser(ctx context.Context, driver *sql.DB) (string, error) {
	var user string
	err := driver.QueryRowContext(ctx, "SELECT CURRENT_USER").Scan(&user)
	if err != nil {
		return "", err
	}
	return user, nil
}
