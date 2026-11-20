package pg

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
		prober, probeErr := advisor.ProbeFor(checkCtx)
		if probeErr != nil {
			return nil, probeErr
		}
		currentRole, err = getCurrentUser(ctx, prober)
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
					Line:   contentStartLine(stmt.Text) + int32(stmt.BaseLine()),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}

func getCurrentUser(ctx context.Context, prober db.Prober) (string, error) {
	res, err := prober.Probe(ctx, db.ProbeContext{}, "SELECT CURRENT_USER")
	if err != nil {
		return "", err
	}
	if len(res.Rows) != 1 || len(res.Rows[0]) != 1 {
		return "", errors.Errorf("expected one row with one column but got %d rows", len(res.Rows))
	}
	user, ok := res.Rows[0][0].(string)
	if !ok {
		return "", errors.Errorf("expected a string but got %T", res.Rows[0][0])
	}
	return user, nil
}
