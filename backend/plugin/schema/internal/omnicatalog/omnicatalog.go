// Package omnicatalog holds the engine-neutral half of a metadata-to-metadata
// migration for the omni MySQL-family catalogs. The MySQL, MariaDB and OceanBase
// catalogs expose the same entry points under their own types, so an engine
// package supplies an Engine value and this package loads both snapshots, refuses
// a diff over an object a catalog could not install, and removes the synthetic
// database qualifier from the generated DDL.
package omnicatalog

import (
	"context"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/store/model"
)

// Database is the synthetic database both snapshots are loaded under. A synced
// snapshot is single-database and its view bodies and foreign keys are qualified
// with its own database name, so a database diffed against another database's
// changelog compares object by object only when both load under ONE name. The
// migration generator then emits that name as a qualifier, which
// StripDatabaseContext removes because Bytebase applies the migration within a
// connection already scoped to the target database.
const Database = "bbcatalog"

// catalogDiff is an omni SchemaDiff that reports whether a diff is empty.
type catalogDiff interface {
	IsEmpty() bool
}

// MigrationPlan is an omni MigrationPlan that renders SQL.
type MigrationPlan interface {
	SQL() string
}

// Engine is one omni catalog package as a metadata diff uses it. The type
// parameters are the catalog and its load report, which differ per engine.
type Engine[Catalog, Report any] struct {
	// New returns an empty catalog.
	New func() *Catalog
	// Load installs the snapshot into the catalog, under Database.
	Load func(*Catalog, *metadatapb.DatabaseSchemaMetadata) (*Report, error)
	// Problems lists the report's per-object problems for a refusal check.
	// When nil, the refusal check is skipped.
	Problems func(*Report) []map[string]error
	// Diff renders the migration between two loaded catalogs. empty reports
	// that there is nothing to migrate.
	Diff func(from, to *Catalog) (sql string, empty bool)
	// CheckCaseOnlyIdentities refuses snapshots whose relation names collide
	// under case-folding in DiffMetadataMigration.
	CheckCaseOnlyIdentities bool
	// SkipStripDatabaseContext disables StripDatabaseContext on generated SQL
	// for catalogs that do not load under the synthetic Database name.
	SkipStripDatabaseContext bool
	// PostProcess optionally transforms non-empty generated SQL before returning it.
	PostProcess func(string) string
}

// DiffSQL builds an Engine.Diff callback from a catalog's Diff and GenerateMigration functions.
func DiffSQL[Catalog any, D catalogDiff, P MigrationPlan](
	diffFn func(from, to *Catalog) D,
	migrateFn func(from, to *Catalog, diff D) P,
) func(from, to *Catalog) (string, bool) {
	return func(from, to *Catalog) (string, bool) {
		diff := diffFn(from, to)
		if diff.IsEmpty() {
			return "", true
		}
		return migrateFn(from, to, diff).SQL(), false
	}
}

// MySQLFamilyEngine builds an Engine for a MySQL-family omni catalog whose
// pointer type implements LoadMetadataInto. Optional afterLoad runs after each
// successful LoadMetadataInto call.
func MySQLFamilyEngine[C any, PC interface {
	*C
	LoadMetadataInto(context.Context, *metadatapb.DatabaseSchemaMetadata, string) (*Report, error)
}, Report any, D catalogDiff, P MigrationPlan](
	newCatalog func() PC,
	problems func(*Report) []map[string]error,
	diffFn func(from, to PC) D,
	migrateFn func(from, to PC, diff D) P,
	afterLoad func(PC, *metadatapb.DatabaseSchemaMetadata),
) Engine[C, Report] {
	return Engine[C, Report]{
		New: func() *C { return newCatalog() },
		Load: func(c *C, meta *metadatapb.DatabaseSchemaMetadata) (*Report, error) {
			report, err := PC(c).LoadMetadataInto(context.Background(), meta, Database)
			if err != nil {
				return report, err
			}
			if afterLoad != nil {
				afterLoad(PC(c), meta)
			}
			return report, nil
		},
		Problems: problems,
		Diff: DiffSQL(
			func(from, to *C) D { return diffFn(PC(from), PC(to)) },
			func(from, to *C, diff D) P { return migrateFn(PC(from), PC(to), diff) },
		),
		CheckCaseOnlyIdentities: true,
	}
}

// DatabaseProto returns d's underlying proto metadata, or nil when d is nil.
func DatabaseProto(d *model.DatabaseMetadata) *metadatapb.DatabaseSchemaMetadata {
	if d == nil {
		return nil
	}
	return d.GetProto()
}

// DiffMetadataMigration generates the migration between two synced snapshots
// using engine, checking case-only identities when engine requests it.
func DiffMetadataMigration[Catalog, Report any](engine Engine[Catalog, Report], oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	oldProto, newProto := DatabaseProto(oldSchema), DatabaseProto(newSchema)
	if engine.CheckCaseOnlyIdentities {
		oldCS := oldSchema != nil && oldSchema.GetIsObjectCaseSensitive()
		newCS := newSchema != nil && newSchema.GetIsObjectCaseSensitive()
		if err := rejectCaseOnlyIdentities(oldProto, newProto, oldCS, newCS); err != nil {
			return "", err
		}
	}
	return DiffMetadata(engine, oldProto, newProto)
}

// MetadataMigrationFunc returns a schema.Engine.DiffMetadataMigration callback
// backed by engine.
func MetadataMigrationFunc[Catalog, Report any](engine Engine[Catalog, Report]) func(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	return func(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
		return DiffMetadataMigration(engine, oldSchema, newSchema)
	}
}

// DiffMetadata loads both snapshots into one catalog each and renders the
// migration between them.
//
// An object the catalog cannot install as the snapshot defines it (a stand-in, a
// missing routine, or a table without a clause that did not parse) is safe to
// leave in place only when it is the same on both sides: the two loads then agree
// on it and the diff skips it. Where it differs, the diff would compare stand-ins,
// so the migration is refused rather than guessed.
func DiffMetadata[Catalog, Report any](engine Engine[Catalog, Report], oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata) (string, error) {
	if oldMeta == nil {
		oldMeta = &metadatapb.DatabaseSchemaMetadata{}
	}
	if newMeta == nil {
		newMeta = &metadatapb.DatabaseSchemaMetadata{}
	}
	from, fromReport, err := load(engine, oldMeta)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, toReport, err := load(engine, newMeta)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	if engine.Problems != nil {
		if err := unloadableChanges(oldMeta, newMeta, engine.Problems(fromReport), engine.Problems(toReport)); err != nil {
			return "", err
		}
	}
	sql, empty := engine.Diff(from, to)
	if empty {
		return "", nil
	}
	if !engine.SkipStripDatabaseContext {
		sql = StripDatabaseContext(sql)
	}
	if engine.PostProcess != nil {
		sql = engine.PostProcess(sql)
	}
	return sql, nil
}

func load[Catalog, Report any](engine Engine[Catalog, Report], meta *metadatapb.DatabaseSchemaMetadata) (*Catalog, *Report, error) {
	c := engine.New()
	report, err := engine.Load(c, meta)
	if err != nil {
		return nil, nil, err
	}
	return c, report, nil
}

// unloadableChanges returns an error naming each object that either load could
// not install as defined and that differs between the two snapshots.
func unloadableChanges(oldMeta, newMeta *metadatapb.DatabaseSchemaMetadata, reports ...[]map[string]error) error {
	var changed []string
	reasons := make(map[string]error)
	for _, report := range reports {
		for _, problems := range report {
			for key, err := range problems {
				if _, seen := reasons[key]; !seen {
					reasons[key] = err
				}
			}
		}
	}
	for key, err := range reasons {
		if !proto.Equal(snapshotObject(oldMeta, key), snapshotObject(newMeta, key)) {
			changed = append(changed, key+": "+err.Error())
		}
	}
	if len(changed) == 0 {
		return nil
	}
	slices.Sort(changed)
	return errors.Errorf("cannot generate a migration for objects whose definitions could not be loaded: %s", strings.Join(changed, "; "))
}

// snapshotObject finds the object a LoadMetadataReport key names, or nil.
func snapshotObject(meta *metadatapb.DatabaseSchemaMetadata, key string) proto.Message {
	kind, name, _ := strings.Cut(key, ":")
	for _, s := range meta.GetSchemas() {
		switch kind {
		case "table":
			for _, t := range s.GetTables() {
				if t.GetName() == name {
					return tableDefinition(t)
				}
			}
		case "view":
			for _, v := range s.GetViews() {
				if v.GetName() == name {
					return v
				}
			}
		case "sequence":
			for _, sequence := range s.GetSequences() {
				if sequence.GetName() == name {
					clone := proto.CloneOf(sequence)
					clone.LastValue = ""
					return clone
				}
			}
		case "function":
			for _, f := range s.GetFunctions() {
				if f.GetName() == name {
					return f
				}
			}
		case "procedure":
			for _, p := range s.GetProcedures() {
				if p.GetName() == name {
					return p
				}
			}
		case "trigger":
			table, trigger, _ := strings.Cut(name, ".")
			for _, t := range s.GetTables() {
				if t.GetName() != table {
					continue
				}
				for _, trg := range t.GetTriggers() {
					if trg.GetName() == trigger {
						return trg
					}
				}
			}
		case "event":
			for _, e := range s.GetEvents() {
				if e.GetName() == name {
					return e
				}
			}
		default:
		}
	}
	return nil
}

// tableDefinition is t without the statistics sync records, which change with
// the data rather than the definition.
func tableDefinition(t *metadatapb.TableMetadata) *metadatapb.TableMetadata {
	t = proto.CloneOf(t)
	t.RowCount, t.DataSize, t.IndexSize, t.DataFree = 0, 0, 0, 0
	return t
}

// StripDatabaseContext removes the synthetic database qualifier from generated DDL.
// Both snapshots are loaded under Database so unqualified references resolve, which
// makes the omni migration generator emit fully-qualified identifiers (e.g. ALTER
// TABLE `bbcatalog`.`t`). Bytebase applies the migration within a connection already
// scoped to the target database, so the qualifier must be removed.
//
// The scan is literal- and comment-aware: a '…' or "…" string literal — say a view
// body or CHECK expression whose text happens to contain the byte sequence
// `bbcatalog`.`x` — is copied opaquely (scanViewBodyToken, a quote/identifier
// scanner), so only identifier-position qualifiers are stripped, and the identifier
// following a stripped qualifier is itself copied opaquely so even an object literally
// named "bbcatalog" keeps its own quoting. Comments matter because omni emits routine
// bodies VERBATIM into create ops: a user comment ("-- don't …") carries unbalanced
// quotes that would otherwise open a phantom literal swallowing later qualifiers, so
// #…, "-- …", and non-executable /*…*/ comments are copied opaquely, while executable
// /*!…*/ comment interiors are scanned as the live SQL they are.
func StripDatabaseContext(sql string) string {
	quoted := "`" + Database + "`"
	if !strings.Contains(sql, quoted+".") {
		return sql
	}
	var out strings.Builder
	out.Grow(len(sql))
	for i := 0; i < len(sql); {
		c := sql[i]
		if isViewBodySpaceByte(c) {
			out.WriteByte(c)
			i++
			continue
		}
		if end, isComment := scanMySQLComment(sql, i); isComment {
			out.WriteString(sql[i:end])
			i = end
			continue
		}
		end := scanViewBodyToken(sql, i)
		tok := sql[i:end]
		if tok == quoted && end < len(sql) && sql[end] == '.' && end+1 < len(sql) && sql[end+1] == '`' {
			identEnd := scanViewBodyToken(sql, end+1)
			out.WriteString(sql[end+1 : identEnd])
			i = identEnd
			continue
		}
		out.WriteString(tok)
		i = end
	}
	return out.String()
}

// scanMySQLComment reports whether a NON-EXECUTABLE MySQL comment starts at i and, if
// so, the offset just past it: "#…" and "-- …" (double dash followed by whitespace or
// end of input, per MySQL's rule) run to end of line; "/*…*/" runs to its terminator.
// Version-gated executable comments ("/*!…*/") are NOT comments — their interior is
// live SQL — so they are left to the normal token scan.
func scanMySQLComment(sql string, i int) (int, bool) {
	switch sql[i] {
	case '#':
		return mysqlLineCommentEnd(sql, i), true
	case '-':
		if i+1 < len(sql) && sql[i+1] == '-' && (i+2 >= len(sql) || isViewBodySpaceByte(sql[i+2])) {
			return mysqlLineCommentEnd(sql, i), true
		}
	case '/':
		if i+1 < len(sql) && sql[i+1] == '*' && (i+2 >= len(sql) || sql[i+2] != '!') {
			if idx := strings.Index(sql[i+2:], "*/"); idx >= 0 {
				return i + 2 + idx + 2, true
			}
			return len(sql), true
		}
	default:
	}
	return 0, false
}

// mysqlLineCommentEnd returns the offset of the newline terminating a line comment
// starting at i (or end of input); the newline itself is left to the whitespace copy.
func mysqlLineCommentEnd(sql string, i int) int {
	for i < len(sql) && sql[i] != '\n' {
		i++
	}
	return i
}

// isViewBodySpaceByte reports whether b is whitespace between tokens.
func isViewBodySpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// scanViewBodyToken returns the end offset of the token starting at i (a non-space
// byte): a whole '…'/"…" string literal, a whole `…` identifier, a word run, or a
// single punctuation byte. String literals honor backslash escapes and doubled
// delimiters; backtick identifiers only doubled backticks. An unterminated literal or
// identifier runs to the end. It copies the scanner omni's SDL dump uses for view bodies.
func scanViewBodyToken(body string, i int) int {
	switch c := body[i]; c {
	case '\'', '"':
		for j := i + 1; j < len(body); {
			switch {
			case body[j] == '\\' && j+1 < len(body):
				j += 2
			case body[j] == c:
				if j+1 < len(body) && body[j+1] == c {
					j += 2
					continue
				}
				return j + 1
			default:
				j++
			}
		}
		return len(body)
	case '`':
		for j := i + 1; j < len(body); {
			if body[j] == '`' {
				if j+1 < len(body) && body[j+1] == '`' {
					j += 2
					continue
				}
				return j + 1
			}
			j++
		}
		return len(body)
	default:
		if !isViewBodyWordByte(body[i]) {
			return i + 1
		}
		j := i + 1
		for j < len(body) && isViewBodyWordByte(body[j]) {
			j++
		}
		return j
	}
}

// isViewBodyWordByte reports whether b can be part of a bare word. Multibyte (>= 0x80)
// bytes are treated as word bytes so an unquoted multibyte identifier stays one token.
func isViewBodyWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '_' || b == '$':
		return true
	case b >= 0x80:
		return true
	default:
		return false
	}
}
