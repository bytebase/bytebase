package oracle

import (
	"context"
	"regexp"
	"strconv"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/bytebase/omni/oracle/ast"
	oraclecompletion "github.com/bytebase/omni/oracle/completion"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

func init() {
	engine := base.OmniEngine(analysis.Engine, oraclecompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.QuerySpan = GetQuerySpan
	base.Register(engine, storepb.Engine_ORACLE)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// Version is an Oracle release, such as 12.2.
type Version = analysis.Version

func ParseVersion(banner string) (*Version, error) {
	match := versionRe.FindStringSubmatch(banner)
	if len(match) >= 3 {
		firstVersion, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, errors.Errorf("failed to parse first version from banner: %s", banner)
		}
		secondVersion, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, errors.Errorf("failed to parse second version from banner: %s", banner)
		}
		return &Version{First: firstVersion, Second: secondVersion}, nil
	}
	return nil, errors.Errorf("failed to parse version from banner: %s", banner)
}

// GetStatementTypes returns the distinct types of the statements, including
// STATEMENT_TYPE_UNSPECIFIED for one omni does not classify.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected Oracle omni AST", analysis.StatementTypeOf)
}

func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		Instance:        gCtx.InstanceID,
		DefaultDatabase: database,
		Catalog:         newCatalog(ctx, gCtx),
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

// newCatalog reads synced metadata for omni. A nil metadata function stays
// nil, so omni fails where the extractor did.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(instance, name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, instance, name)
			if err != nil {
				return nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return nil, nil
			}
			return &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	if gCtx.GetLinkedDatabaseMetadataFunc != nil {
		catalog.GetLinkedDatabase = func(instance, link, schema string) (string, string, analysis.Database, error) {
			linkedInstance, name, database, err := gCtx.GetLinkedDatabaseMetadataFunc(ctx, instance, link, schema)
			if err != nil {
				return "", "", nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return linkedInstance, name, nil, nil
			}
			return linkedInstance, name, &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	return catalog
}
