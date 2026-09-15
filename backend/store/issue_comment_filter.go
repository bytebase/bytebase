package store

import (
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celtypes "github.com/google/cel-go/common/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// GetIssueCommentListFilter accepts only the documented root predicates. An
// empty filter is the timeline: events and root comments, so a reply never
// consumes a page slot for clients that predate threads.
func GetIssueCommentListFilter(filter, projectID string, issueUID int64) (*FindIssueCommentMessage, error) {
	find := &FindIssueCommentMessage{ProjectID: projectID, IssueUID: &issueUID}
	if filter == "" {
		find.TopLevelOnly = true
		return find, nil
	}
	ast, err := common.ParseCELFilter(filter)
	if err != nil {
		return nil, err
	}
	expr := ast.NativeRep().Expr()
	if expr.Kind() != celast.CallKind {
		return nil, errors.New("expected a root comparison")
	}
	call := expr.AsCall()
	args := call.Args()
	if call.IsMemberFunction() || len(args) != 2 || args[0].Kind() != celast.IdentKind || args[0].AsIdent() != "root" {
		return nil, errors.New("expected root == null, root == a comment name, or root in a list of comment names")
	}
	var values []celast.Expr
	switch call.FunctionName() {
	case celoperators.Equals:
		if args[1].Kind() == celast.LiteralKind && args[1].AsLiteral() == celtypes.NullValue {
			find.TopLevelOnly = true
			return find, nil
		}
		values = []celast.Expr{args[1]}
	case celoperators.In:
		if args[1].Kind() != celast.ListKind {
			return nil, errors.New("root in requires a list of comment names")
		}
		values = args[1].AsList().Elements()
	default:
		return nil, errors.New("unsupported root filter operator")
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if value.Kind() != celast.LiteralKind {
			return nil, errors.New("root must be a literal comment name")
		}
		name, ok := value.AsLiteral().Value().(string)
		if !ok {
			return nil, errors.New("root must be a string comment name")
		}
		p, i, id, err := common.GetProjectIDIssueUIDIssueCommentID(name)
		if err != nil || p != projectID || i != issueUID {
			return nil, errors.Errorf("root %q must name a comment in the parent issue", name)
		}
		ids = append(ids, id)
	}
	find.ParentIDs = &ids
	return find, nil
}
