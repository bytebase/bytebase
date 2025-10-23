package store

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"

	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type ChangelogStatus string

const (
	ChangelogStatusPending ChangelogStatus = "PENDING"
	ChangelogStatusDone    ChangelogStatus = "DONE"
	ChangelogStatusFailed  ChangelogStatus = "FAILED"
)

type ChangelogMessage struct {
	InstanceID   string
	DatabaseName string
	Payload      *storepb.ChangelogPayload

	PrevSyncHistoryUID *int64
	SyncHistoryUID     *int64
	Status             ChangelogStatus

	// output only
	UID       int64
	CreatedAt time.Time

	PrevSchema    string
	Schema        string
	Statement     string
	StatementSize int64
}

type FindChangelogMessage struct {
	UID          *int64
	InstanceID   *string
	DatabaseName *string

	TypeList        []string
	Status          *ChangelogStatus
	ResourcesFilter *string

	Limit  *int
	Offset *int

	// If false, PrevSchema, Schema are truncated
	ShowFull       bool
	HasSyncHistory bool
}

type UpdateChangelogMessage struct {
	UID int64

	SyncHistoryUID *int64
	RevisionUID    *int64
	Status         *ChangelogStatus
}

func (s *Store) CreateChangelog(ctx context.Context, create *ChangelogMessage) (int64, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	p, err := protojson.Marshal(create.Payload)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to marshal")
	}

	q := qb.Q().Space(`
		INSERT INTO changelog (
			instance,
			db_name,
			status,
			prev_sync_history_id,
			sync_history_id,
			payload
		) VALUES (
		 	?,
			?,
			?,
			?,
			?,
			?
		)
		RETURNING id
	`, create.InstanceID, create.DatabaseName, create.Status, create.PrevSyncHistoryUID, create.SyncHistoryUID, p)

	query, args, err := q.ToSQL()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to build sql")
	}

	var id int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "failed to insert")
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit tx")
	}

	return id, nil
}

func (s *Store) UpdateChangelog(ctx context.Context, update *UpdateChangelogMessage) error {
	set := qb.Q()
	if v := update.SyncHistoryUID; v != nil {
		set.Comma("sync_history_id = ?", *v)
	}
	if v := update.RevisionUID; v != nil {
		set.Comma("payload = payload || jsonb_build_object('revision', ?::BIGINT)", *v)
	}
	if v := update.Status; v != nil {
		set.Comma("status = ?", *v)
	}

	if set.Len() == 0 {
		return errors.Errorf("update nothing")
	}

	query, args, err := qb.Q().Space("UPDATE changelog SET ? WHERE id = ?", set, update.UID).ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to update")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	return nil
}

func (s *Store) ListChangelogs(ctx context.Context, find *FindChangelogMessage) ([]*ChangelogMessage, error) {
	truncateSize := 512
	if common.IsDev() {
		truncateSize = 4
	}
	shPreField := fmt.Sprintf("LEFT(sh_pre.raw_dump, %d)", truncateSize)
	if find.ShowFull {
		shPreField = "sh_pre.raw_dump"
	}
	shCurField := fmt.Sprintf("LEFT(sh_cur.raw_dump, %d)", truncateSize)
	if find.ShowFull {
		shCurField = "sh_cur.raw_dump"
	}
	sheetField := fmt.Sprintf("LEFT(sheet_blob.content, %d)", truncateSize)
	if find.ShowFull {
		sheetField = "sheet_blob.content"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			changelog.id,
			changelog.created_at,
			changelog.instance,
			changelog.db_name,
			changelog.status,
			changelog.prev_sync_history_id,
			changelog.sync_history_id,
			COALESCE(%s, ''),
			COALESCE(%s, ''),
			COALESCE(%s, ''),
			COALESCE(OCTET_LENGTH(sheet_blob.content), 0),
			changelog.payload
		FROM changelog
		LEFT JOIN sync_history sh_pre ON sh_pre.id = changelog.prev_sync_history_id
		LEFT JOIN sync_history sh_cur ON sh_cur.id = changelog.sync_history_id
		LEFT JOIN sheet ON sheet.id::text = split_part(changelog.payload->>'sheet', '/', 4)
		LEFT JOIN sheet_blob ON sheet.sha256 = sheet_blob.sha256
		WHERE TRUE
	`,
		shPreField,
		shCurField,
		sheetField,
	))

	if v := find.UID; v != nil {
		q.And("changelog.id = ?", *v)
	}
	if v := find.InstanceID; v != nil {
		q.And("changelog.instance = ?", *v)
	}
	if v := find.DatabaseName; v != nil {
		q.And("changelog.db_name = ?", *v)
	}
	if v := find.ResourcesFilter; v != nil {
		text, err := generateResourceFilter(*v, "changelog.payload")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate resource filter from %q", *v)
		}
		if text != "" {
			q.And(text)
		}
	}
	if v := find.Status; v != nil {
		q.And("changelog.status = ?", string(*v))
	}
	if find.HasSyncHistory {
		q.And("changelog.sync_history_id IS NOT NULL")
	}
	if len(find.TypeList) > 0 {
		q.And("changelog.payload->>'type' = ANY(?)", find.TypeList)
	}

	q.Space("ORDER BY changelog.id DESC")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	var changelogs []*ChangelogMessage
	for rows.Next() {
		c := ChangelogMessage{
			Payload: &storepb.ChangelogPayload{},
		}
		var payload []byte

		if err := rows.Scan(
			&c.UID,
			&c.CreatedAt,
			&c.InstanceID,
			&c.DatabaseName,
			&c.Status,
			&c.PrevSyncHistoryUID,
			&c.SyncHistoryUID,
			&c.PrevSchema,
			&c.Schema,
			&c.Statement,
			&c.StatementSize,
			&payload,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, c.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}

		changelogs = append(changelogs, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return changelogs, nil
}

func (s *Store) GetChangelog(ctx context.Context, find *FindChangelogMessage) (*ChangelogMessage, error) {
	changelogs, err := s.ListChangelogs(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(changelogs) == 0 {
		return nil, nil
	}
	if len(changelogs) > 1 {
		return nil, errors.Errorf("found %d changelogs with find %v, expect 1", len(changelogs), *find)
	}
	return changelogs[0], nil
}

type resourceDatabase struct {
	name    string
	schemas schemaMap
}

type databaseMap map[string]*resourceDatabase

type resourceSchema struct {
	name   string
	tables tableMap
}

type schemaMap map[string]*resourceSchema

type resourceTable struct {
	name string
}

type tableMap map[string]*resourceTable

// The CEL filter MUST be a Disjunctive Normal Form (DNF) expression.
// In other words, the CEL expression consists of several parts connected by OR operators.
// For example, the following expression is valid:
// (
//
//	tableExists("db", "public", "table1") &&
//	tableExists("db", "public", "table2")
//
// ) || (
//
//	tableExists("db", "public", "table3")
//
// )
// .
func generateResourceFilter(filter string, jsonbFieldName string) (string, error) {
	env, err := cel.NewEnv(
		cel.Function("tableExists",
			cel.Overload("tableExists_string",
				[]*cel.Type{cel.StringType, cel.StringType, cel.StringType},
				cel.BoolType,
			),
		),
	)
	if err != nil {
		return "", err
	}

	ast, iss := env.Compile(filter)
	if iss != nil && iss.Err() != nil {
		return "", iss.Err()
	}

	rewriter := &expressionRewriter{
		metaMap: make(databaseMap),
	}

	parsedExpr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return "", err
	}
	if err := rewriter.rewriteExpression(parsedExpr.Expr); err != nil {
		return "", err
	}

	if len(rewriter.metaMap) != 0 {
		if err := rewriter.appendDNFPart(); err != nil {
			return "", err
		}
	}

	if len(rewriter.dnfParts) == 0 {
		return "", nil
	}

	var buf strings.Builder
	if len(rewriter.dnfParts) > 1 {
		if _, err := buf.WriteString("("); err != nil {
			return "", err
		}
	}
	for i, part := range rewriter.dnfParts {
		if i > 0 {
			if _, err := buf.WriteString(" OR "); err != nil {
				return "", err
			}
		}
		if _, err := buf.WriteString(fmt.Sprintf("(%s @> '", jsonbFieldName)); err != nil {
			return "", err
		}
		if _, err := buf.WriteString(part); err != nil {
			return "", err
		}
		if _, err := buf.WriteString("'::jsonb)"); err != nil {
			return "", err
		}
	}
	if len(rewriter.dnfParts) > 1 {
		if _, err := buf.WriteString(")"); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

type expressionRewriter struct {
	metaMap  databaseMap
	dnfParts []string
}

func (r *expressionRewriter) appendDNFPart() error {
	if r.metaMap == nil {
		return nil
	}

	defer func() {
		r.metaMap = make(databaseMap)
	}()

	var meta storepb.ChangedResources
	for _, dbMeta := range r.metaMap {
		db := &storepb.ChangedResourceDatabase{
			Name: dbMeta.name,
		}
		for _, schemaMeta := range dbMeta.schemas {
			schema := &storepb.ChangedResourceSchema{
				Name: schemaMeta.name,
			}
			for _, tableMeta := range schemaMeta.tables {
				table := &storepb.ChangedResourceTable{
					Name: tableMeta.name,
				}
				schema.Tables = append(schema.Tables, table)
			}
			slices.SortFunc(schema.Tables, func(a, b *storepb.ChangedResourceTable) int {
				if a.Name < b.Name {
					return -1
				} else if a.Name > b.Name {
					return 1
				}
				return 0
			})
			db.Schemas = append(db.Schemas, schema)
		}
		slices.SortFunc(db.Schemas, func(a, b *storepb.ChangedResourceSchema) int {
			if a.Name < b.Name {
				return -1
			} else if a.Name > b.Name {
				return 1
			}
			return 0
		})
		meta.Databases = append(meta.Databases, db)
	}
	slices.SortFunc(meta.Databases, func(a, b *storepb.ChangedResourceDatabase) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})

	text, err := protojson.Marshal(&storepb.InstanceChangeHistoryPayload{
		ChangedResources: &meta,
	})
	if err != nil {
		return err
	}
	r.dnfParts = append(r.dnfParts, string(text))
	return nil
}

func (r *expressionRewriter) rewriteExpression(expr *exprpb.Expr) error {
	switch e := expr.ExprKind.(type) {
	case *exprpb.Expr_CallExpr:
		switch e.CallExpr.Function {
		case "_||_":
			for _, arg := range e.CallExpr.Args {
				if err := r.rewriteExpression(arg); err != nil {
					return err
				}
				if err := r.appendDNFPart(); err != nil {
					return err
				}
			}
		case "_&&_":
			for _, arg := range e.CallExpr.Args {
				if err := r.rewriteExpression(arg); err != nil {
					return err
				}
			}
		case "tableExists":
			if len(e.CallExpr.Args) != 3 {
				return errors.Errorf("invalid tableExists function call: %v, expected three arguments buf got %d", e.CallExpr, len(e.CallExpr.Args))
			}
			var args []string
			for _, arg := range e.CallExpr.Args {
				switch a := arg.ExprKind.(type) {
				case *exprpb.Expr_ConstExpr:
					switch a.ConstExpr.ConstantKind.(type) {
					case *exprpb.Constant_StringValue:
						args = append(args, a.ConstExpr.GetStringValue())
					default:
						return errors.Errorf("invalid tableExists function call: %v, expected string arguments buf got %v", e.CallExpr, arg)
					}
				default:
					return errors.Errorf("invalid tableExists function call: %v, expected constant arguments buf got %v", e.CallExpr, arg)
				}
			}
			database, ok := r.metaMap[args[0]]
			if !ok {
				database = &resourceDatabase{
					name:    args[0],
					schemas: make(schemaMap),
				}
				r.metaMap[args[0]] = database
			}
			schema, ok := database.schemas[args[1]]
			if !ok {
				schema = &resourceSchema{
					name:   args[1],
					tables: make(tableMap),
				}
				database.schemas[args[1]] = schema
			}
			schema.tables[args[2]] = &resourceTable{
				name: args[2],
			}
		default:
			// Ignore other function calls
		}
	default:
		return errors.Errorf("invalid expression: %v", expr)
	}
	return nil
}
