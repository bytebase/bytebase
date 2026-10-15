package aireview

import (
	"fmt"
	"strings"
	"unicode"

	metadatapb "github.com/bytebase/omni/metadata"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// index lists every object of the database on the first call.
func (c *catalogTools) index() []*catalogObject {
	if !c.indexed {
		c.indexed = true
		for _, s := range c.schema.GetSchemas() {
			c.indexSchema(s)
		}
		c.indexDatabase()
	}
	return c.objects
}

// indexDatabase lists the objects that belong to the database, not to one of
// its schemas.
func (c *catalogTools) indexDatabase() {
	for _, trigger := range c.schema.GetEventTriggers() {
		c.objects = append(c.objects, &catalogObject{kind: kindEventTrigger, name: trigger.GetName(), render: func() string {
			return join(plainEventTrigger(trigger), extensionMember(trigger.GetSkipDump()), plainComment("", trigger.GetComment()))
		}})
	}
	for _, extension := range c.schema.GetExtensions() {
		c.objects = append(c.objects, &catalogObject{kind: kindExtension, schema: extension.GetSchema(), name: extension.GetName(), render: func() string {
			definition := fmt.Sprintf("EXTENSION %s VERSION %s", qualify(extension.GetSchema(), extension.GetName()), extension.GetVersion())
			return join(definition, plainComment("", extension.GetDescription()))
		}})
	}
}

func (c *catalogTools) indexSchema(s *metadatapb.SchemaMetadata) {
	schemaName := s.GetName()
	// The prompt names every schema already, so a schema is an object of its
	// own only when the sync holds its owner, its comment, or that an
	// extension owns it.
	if schemaName != "" && (s.GetOwner() != "" || s.GetComment() != "" || s.GetSkipDump()) {
		c.objects = append(c.objects, &catalogObject{kind: kindSchema, name: schemaName, render: func() string {
			definition := "SCHEMA " + schemaName
			if owner := s.GetOwner(); owner != "" {
				definition += " OWNER " + owner
			}
			return join(definition, extensionMember(s.GetSkipDump()), plainComment("", s.GetComment()))
		}})
	}
	add := func(kind string, name string, signature string, statistics *tableStatistics, render func() string) {
		c.objects = append(c.objects, &catalogObject{kind: kind, schema: schemaName, name: name, signature: signature, statistics: statistics, render: render})
	}
	// addMember is add for a kind PostgreSQL marks when an extension owns the
	// object. The schema renderers do not write the mark, so it follows the
	// definition.
	addMember := func(kind string, name string, signature string, statistics *tableStatistics, skipDump bool, render func() string) {
		if skipDump {
			definition := render
			render = func() string { return join(definition(), extensionMember(true)) }
		}
		add(kind, name, signature, statistics, render)
	}
	owned := make(map[string][]*metadatapb.SequenceMetadata)
	for _, sequence := range s.GetSequences() {
		if table := sequence.GetOwnerTable(); table != "" {
			owned[table] = append(owned[table], sequence)
		}
	}
	for _, table := range s.GetTables() {
		addMember(kindTable, table.GetName(), "", statisticsOf(c.syncedStatistics(), table), table.GetSkipDump(), func() string {
			return c.tableDefinition(schemaName, table, owned[table.GetName()])
		})
	}
	for _, table := range s.GetExternalTables() {
		add(kindExternalTable, table.GetName(), "", nil, func() string {
			return join(plainExternalTable(c.engine, schemaName, table), columnComments(table.GetColumns()))
		})
	}
	for _, view := range s.GetViews() {
		addMember(kindView, view.GetName(), "", nil, view.GetSkipDump(), func() string { return c.viewDefinition(schemaName, view) })
	}
	for _, view := range s.GetMaterializedViews() {
		addMember(kindMaterializedView, view.GetName(), "", nil, view.GetSkipDump(), func() string { return c.materializedViewDefinition(schemaName, view) })
	}
	for _, function := range s.GetFunctions() {
		addMember(kindFunction, function.GetName(), function.GetSignature(), nil, function.GetSkipDump(), func() string {
			render := func(function *metadatapb.FunctionMetadata) (string, error) {
				return schema.GetFunctionDefinition(c.engine, schemaName, function)
			}
			type functionpb = metadatapb.FunctionMetadata
			// Each creation setting is its own part, so one a renderer
			// leaves out, such as MySQL's database collation, is still shown.
			return complete(function, render, function.GetDefinition,
				extra[*functionpb]{strip: func(f *functionpb) { f.SqlMode = "" }, text: contextLine("sql_mode", function.GetSqlMode())},
				extra[*functionpb]{strip: func(f *functionpb) { f.CharacterSetClient = "" }, text: contextLine("character_set_client", function.GetCharacterSetClient())},
				extra[*functionpb]{strip: func(f *functionpb) { f.CollationConnection = "" }, text: contextLine("collation_connection", function.GetCollationConnection())},
				extra[*functionpb]{strip: func(f *functionpb) { f.DatabaseCollation = "" }, text: contextLine("database_collation", function.GetDatabaseCollation())},
				extra[*functionpb]{strip: func(f *functionpb) { f.Comment = "" }, text: plainComment("", function.GetComment())},
			)
		})
	}
	for _, procedure := range s.GetProcedures() {
		addMember(kindProcedure, procedure.GetName(), procedure.GetSignature(), nil, procedure.GetSkipDump(), func() string {
			render := func(procedure *metadatapb.ProcedureMetadata) (string, error) {
				return schema.GetProcedureDefinition(c.engine, schemaName, procedure)
			}
			type procedurepb = metadatapb.ProcedureMetadata
			return complete(procedure, render, procedure.GetDefinition,
				extra[*procedurepb]{strip: func(p *procedurepb) { p.SqlMode = "" }, text: contextLine("sql_mode", procedure.GetSqlMode())},
				extra[*procedurepb]{strip: func(p *procedurepb) { p.CharacterSetClient = "" }, text: contextLine("character_set_client", procedure.GetCharacterSetClient())},
				extra[*procedurepb]{strip: func(p *procedurepb) { p.CollationConnection = "" }, text: contextLine("collation_connection", procedure.GetCollationConnection())},
				extra[*procedurepb]{strip: func(p *procedurepb) { p.DatabaseCollation = "" }, text: contextLine("database_collation", procedure.GetDatabaseCollation())},
				extra[*procedurepb]{strip: func(p *procedurepb) { p.Comment = "" }, text: plainComment("", procedure.GetComment())},
			)
		})
	}
	for _, sequence := range s.GetSequences() {
		addMember(kindSequence, sequence.GetName(), "", nil, sequence.GetSkipDump(), func() string {
			render := func(sequence *metadatapb.SequenceMetadata) (string, error) {
				return schema.GetSequenceDefinition(c.engine, schemaName, sequence)
			}
			plain := func() string { return plainSequence(schemaName, sequence) }
			return complete(sequence, render, plain,
				extra[*metadatapb.SequenceMetadata]{
					strip: func(sequence *metadatapb.SequenceMetadata) { sequence.Comment = "" },
					text:  plainComment("", sequence.GetComment()),
				},
				// The position as of the sync, which the definition does not
				// carry.
				extra[*metadatapb.SequenceMetadata]{
					strip: func(sequence *metadatapb.SequenceMetadata) { sequence.LastValue = "" },
					text:  plainFact("LAST VALUE AT SYNC", sequence.GetLastValue()),
				},
			)
		})
	}
	for _, pkg := range s.GetPackages() {
		add(kindPackage, pkg.GetName(), "", nil, pkg.GetDefinition)
	}
	for _, event := range s.GetEvents() {
		add(kindEvent, event.GetName(), "", nil, func() string {
			context := creationContext{sqlMode: event.GetSqlMode(), characterSetClient: event.GetCharacterSetClient(), collationConnection: event.GetCollationConnection(), timeZone: event.GetTimeZone()}
			return join(event.GetDefinition(), context.text(), plainComment("", event.GetComment()))
		})
	}
	for _, stream := range s.GetStreams() {
		add(kindStream, stream.GetName(), "", nil, func() string {
			// The definition is the stream's DDL, which says neither who owns
			// it nor whether it went stale.
			var stale string
			if stream.GetStale() {
				stale = "STALE: the stream can no longer return the changes since its offset"
			}
			return join(stream.GetDefinition(), plainFact("OWNER", stream.GetOwner()), stale, plainComment("", stream.GetComment()))
		})
	}
	for _, task := range s.GetTasks() {
		add(kindTask, task.GetName(), "", nil, func() string {
			// The definition is the task's DDL, which says neither who owns
			// it nor whether it is running.
			var state string
			switch task.GetState() {
			case metadatapb.TaskMetadata_STATE_STARTED:
				state = "STATE: started"
			case metadatapb.TaskMetadata_STATE_SUSPENDED:
				state = "STATE: suspended"
			default:
			}
			return join(task.GetDefinition(), plainFact("OWNER", task.GetOwner()), state, plainComment("", task.GetComment()))
		})
	}
	for _, enum := range s.GetEnumTypes() {
		addMember(kindEnumType, enum.GetName(), "", nil, enum.GetSkipDump(), func() string {
			definition := fmt.Sprintf("ENUM TYPE %s: %s", qualify(schemaName, enum.GetName()), sqlStrings(enum.GetValues()))
			return join(definition, plainComment("", enum.GetComment()))
		})
	}
	for _, composite := range s.GetCompositeTypes() {
		addMember(kindCompositeType, composite.GetName(), "", nil, composite.GetSkipDump(), func() string {
			attributes := make([]string, 0, len(composite.GetAttributes()))
			comments := []string{plainComment("", composite.GetComment())}
			for _, attribute := range composite.GetAttributes() {
				text := ident(attribute.GetName()) + " " + attribute.GetType()
				if collation := attribute.GetCollation(); collation != "" {
					text += " COLLATE " + collation
				}
				attributes = append(attributes, text)
				comments = append(comments, plainComment("ATTRIBUTE "+attribute.GetName(), attribute.GetComment()))
			}
			definition := fmt.Sprintf("COMPOSITE TYPE %s (%s)", qualify(schemaName, composite.GetName()), strings.Join(attributes, ", "))
			return join(definition, lines(comments))
		})
	}
}

// statisticsOf returns the statistics the sync holds of a table. A value the
// engine's sync reads is shown even when it reads 0. Another value is shown
// only when it is not 0, because 0 is also what an unread value holds.
func statisticsOf(synced statisticsSynced, table *metadatapb.TableMetadata) *tableStatistics {
	rows, data, index := synced.rows, synced.data, synced.index
	value := func(synced bool, v int64) *int64 {
		if !synced && v == 0 {
			return nil
		}
		return &v
	}
	statistics := &tableStatistics{
		Rows:       value(rows, table.GetRowCount()),
		DataBytes:  value(data, table.GetDataSize()),
		IndexBytes: value(index, table.GetIndexSize()),
	}
	if statistics.Rows == nil && statistics.DataBytes == nil && statistics.IndexBytes == nil {
		return nil
	}
	return statistics
}

// statisticsSynced says which table statistics a sync reads.
type statisticsSynced struct{ rows, data, index bool }

// syncedStatistics reports which table statistics the sync of this database
// reads. Redshift reads the row count and writes 0 for the sizes, and reads
// nothing for a datashare database.
func (c *catalogTools) syncedStatistics() statisticsSynced {
	switch c.engine {
	case storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE,
		storepb.Engine_TIDB, storepb.Engine_STARROCKS, storepb.Engine_DORIS, storepb.Engine_MONGODB:
		return statisticsSynced{rows: true, data: true, index: true}
	case storepb.Engine_SNOWFLAKE, storepb.Engine_CLICKHOUSE, storepb.Engine_BIGQUERY, storepb.Engine_HIVE, storepb.Engine_ELASTICSEARCH:
		return statisticsSynced{rows: true, data: true}
	case storepb.Engine_REDSHIFT:
		if c.schema.GetDatashare() {
			return statisticsSynced{}
		}
		return statisticsSynced{rows: true}
	case storepb.Engine_MSSQL, storepb.Engine_ORACLE, storepb.Engine_COCKROACHDB, storepb.Engine_DYNAMODB:
		return statisticsSynced{rows: true}
	default:
		return statisticsSynced{}
	}
}

// extra is a part of an object that a renderer can leave out, such as its
// triggers or its comment. strip removes the part from the object, and text is
// what the tools write for it.
type extra[T proto.Message] struct {
	strip func(T)
	text  string
}

// complete returns the rendering of an object with the extras the renderer
// leaves out, which are the ones it renders the object the same without. The
// renderers write a schema dump, so they leave out what a dump does not need,
// such as a trigger an extension owns. An engine without a renderer gets the
// plain listing and every extra. The test is a second rendering because the
// text of an extra can stand in the definition for another reason, as a
// comment that repeats a name does.
func complete[T proto.Message](object T, render func(T) (string, error), plain func() string, extras ...extra[T]) string {
	definition, err := render(object)
	rendered := err == nil && strings.TrimSpace(definition) != ""
	if !rendered {
		definition = plain()
	}
	parts := []string{definition}
	for _, extra := range extras {
		if extra.text == "" {
			continue
		}
		if rendered {
			bare := proto.CloneOf(object)
			extra.strip(bare)
			if without, err := render(bare); err != nil || without != definition {
				continue
			}
		}
		parts = append(parts, extra.text)
	}
	return join(parts...)
}

func (c *catalogTools) tableDefinition(schemaName string, table *metadatapb.TableMetadata, owned []*metadatapb.SequenceMetadata) string {
	type tablepb = metadatapb.TableMetadata
	// The renderer writes the ownership of every sequence it is given, so it
	// gets the ones this table owns.
	render := func(table *tablepb) (string, error) {
		return schema.GetTableDefinition(c.engine, schemaName, table, owned)
	}
	plain := func() string { return plainTable(c.engine, schemaName, table) }
	parts := []extra[*tablepb]{
		{strip: func(table *tablepb) { table.Indexes = nil }, text: plainIndexes(table.GetIndexes())},
		{strip: func(table *tablepb) { table.ForeignKeys = nil }, text: plainForeignKeys(table.GetForeignKeys())},
		{strip: func(table *tablepb) { table.CheckConstraints = nil }, text: plainCheckConstraints(table.GetCheckConstraints())},
		{strip: func(table *tablepb) { table.ExcludeConstraints = nil }, text: plainExcludeConstraints(table.GetExcludeConstraints())},
		{strip: func(table *tablepb) { table.Partitions = nil }, text: plainPartitions(table.GetPartitions())},
		{strip: func(table *tablepb) { table.Rules = nil }, text: plainRules(table.GetRules())},
		{strip: func(table *tablepb) { enableRules(table.GetRules()) }, text: disabledRules(table.GetRules())},
		{strip: func(table *tablepb) { table.Comment = "" }, text: plainComment("", table.GetComment())},
		{strip: func(table *tablepb) { table.Owner = "" }, text: plainFact("OWNER", table.GetOwner())},
		{strip: func(table *tablepb) { showIndexes(table.GetIndexes()) }, text: invisibleIndexes(c.engine, table.GetIndexes())},
		{strip: func(table *tablepb) { stripDefaultConstraintNames(table.GetColumns()) }, text: defaultConstraintNames(table.GetColumns())},
		{strip: func(table *tablepb) { table.SortingKeys = nil }, text: plainFact("SORTING KEY", strings.Join(table.GetSortingKeys(), ", "))},
		{strip: func(table *tablepb) { table.ShardingInfo = "" }, text: plainFact("SHARDING", table.GetShardingInfo())},
		{strip: func(table *tablepb) { table.PrimaryKeyType = "" }, text: plainFact("PRIMARY KEY TYPE", table.GetPrimaryKeyType())},
		{strip: func(table *tablepb) { stripColumnComments(table.GetColumns()) }, text: columnComments(table.GetColumns())},
		{strip: func(table *tablepb) { stripIndexComments(table.GetIndexes()) }, text: indexComments(table.GetIndexes())},
	}
	parts = append(parts, partitionParts(table.GetPartitions())...)
	parts = append(parts, triggerParts(qualify(schemaName, table.GetName()), table,
		func(table *tablepb, triggers []*metadatapb.TriggerMetadata) { table.Triggers = triggers })...)
	return complete(table, render, plain, parts...)
}

func (c *catalogTools) viewDefinition(schemaName string, view *metadatapb.ViewMetadata) string {
	type viewpb = metadatapb.ViewMetadata
	render := func(view *viewpb) (string, error) {
		return schema.GetViewDefinition(c.engine, schemaName, view)
	}
	on := qualify(schemaName, view.GetName())
	plain := func() string { return fmt.Sprintf("VIEW %s AS\n%s", on, view.GetDefinition()) }
	parts := []extra[*viewpb]{
		{strip: func(view *viewpb) { view.Rules = nil }, text: plainRules(view.GetRules())},
		{strip: func(view *viewpb) { enableRules(view.GetRules()) }, text: disabledRules(view.GetRules())},
		{strip: func(view *viewpb) { view.Comment = "" }, text: plainComment("", view.GetComment())},
		{strip: func(view *viewpb) { stripColumnComments(view.GetColumns()) }, text: columnComments(view.GetColumns())},
	}
	parts = append(parts, triggerParts(on, view,
		func(view *viewpb, triggers []*metadatapb.TriggerMetadata) { view.Triggers = triggers })...)
	return complete(view, render, plain, parts...)
}

func (c *catalogTools) materializedViewDefinition(schemaName string, view *metadatapb.MaterializedViewMetadata) string {
	type viewpb = metadatapb.MaterializedViewMetadata
	render := func(view *viewpb) (string, error) {
		return schema.GetMaterializedViewDefinition(c.engine, schemaName, view)
	}
	on := qualify(schemaName, view.GetName())
	plain := func() string { return fmt.Sprintf("MATERIALIZED VIEW %s AS\n%s", on, view.GetDefinition()) }
	parts := []extra[*viewpb]{
		{strip: func(view *viewpb) { view.Indexes = nil }, text: plainIndexes(view.GetIndexes())},
		{strip: func(view *viewpb) { view.Comment = "" }, text: plainComment("", view.GetComment())},
		{strip: func(view *viewpb) { stripIndexComments(view.GetIndexes()) }, text: indexComments(view.GetIndexes())},
	}
	parts = append(parts, triggerParts(on, view,
		func(view *viewpb, triggers []*metadatapb.TriggerMetadata) { view.Triggers = triggers })...)
	return complete(view, render, plain, parts...)
}

// partitionParts returns what hangs on the partitions of a table as extras:
// their indexes, the comments of those, and their constraints.
func partitionParts(partitions []*metadatapb.TablePartitionMetadata) []extra[*metadatapb.TableMetadata] {
	type partitionpb = metadatapb.TablePartitionMetadata
	fields := []struct {
		text  func(*partitionpb) string
		strip func(*partitionpb)
	}{
		{
			text:  func(partition *partitionpb) string { return plainIndexes(partition.GetIndexes()) },
			strip: func(partition *partitionpb) { partition.Indexes = nil },
		},
		{
			text:  func(partition *partitionpb) string { return indexComments(partition.GetIndexes()) },
			strip: func(partition *partitionpb) { stripIndexComments(partition.GetIndexes()) },
		},
		{
			text:  func(partition *partitionpb) string { return plainCheckConstraints(partition.GetCheckConstraints()) },
			strip: func(partition *partitionpb) { partition.CheckConstraints = nil },
		},
		{
			text:  func(partition *partitionpb) string { return plainExcludeConstraints(partition.GetExcludeConstraints()) },
			strip: func(partition *partitionpb) { partition.ExcludeConstraints = nil },
		},
	}
	parts := make([]extra[*metadatapb.TableMetadata], 0, len(fields))
	for _, field := range fields {
		var all []string
		eachPartition(partitions, func(partition *partitionpb) {
			if text := field.text(partition); text != "" {
				all = append(all, fmt.Sprintf("ON PARTITION %s:\n  %s", partition.GetName(), strings.ReplaceAll(text, "\n", "\n  ")))
			}
		})
		parts = append(parts, extra[*metadatapb.TableMetadata]{
			strip: func(table *metadatapb.TableMetadata) { eachPartition(table.GetPartitions(), field.strip) },
			text:  lines(all),
		})
	}
	return parts
}

// eachPartition visits the partitions and the partitions inside them.
func eachPartition(partitions []*metadatapb.TablePartitionMetadata, visit func(*metadatapb.TablePartitionMetadata)) {
	for _, partition := range partitions {
		visit(partition)
		eachPartition(partition.GetSubpartitions(), visit)
	}
}

// triggered is an object that triggers can be set on.
type triggered interface {
	proto.Message
	GetTriggers() []*metadatapb.TriggerMetadata
}

// triggerParts returns the triggers of an object and their comments as
// extras. The triggers a dump skips are extras of their own, since a renderer
// that writes the others leaves those out.
func triggerParts[T triggered](on string, object T, set func(T, []*metadatapb.TriggerMetadata)) []extra[T] {
	var parts []extra[T]
	for _, skipped := range []bool{false, true} {
		var kept []*metadatapb.TriggerMetadata
		var comments []string
		for _, trigger := range object.GetTriggers() {
			if trigger.GetSkipDump() == skipped {
				kept = append(kept, trigger)
				comments = append(comments, plainComment("TRIGGER "+trigger.GetName(), trigger.GetComment()))
			}
		}
		parts = append(parts,
			extra[T]{
				strip: func(object T) {
					var others []*metadatapb.TriggerMetadata
					for _, trigger := range object.GetTriggers() {
						if trigger.GetSkipDump() != skipped {
							others = append(others, trigger)
						}
					}
					set(object, others)
				},
				text: plainTriggers(on, kept),
			},
			extra[T]{
				strip: func(object T) {
					for _, trigger := range object.GetTriggers() {
						if trigger.GetSkipDump() == skipped {
							trigger.Comment = ""
						}
					}
				},
				text: lines(comments),
			},
		)
	}
	return parts
}

// plainComment writes the comment of an object, or of the part of it that on
// names.
func plainComment(on string, comment string) string {
	switch {
	case comment == "":
		return ""
	case on == "":
		return "COMMENT: " + comment
	default:
		return fmt.Sprintf("COMMENT ON %s: %s", on, comment)
	}
}

func columnComments(columns []*metadatapb.ColumnMetadata) string {
	comments := make([]string, 0, len(columns))
	for _, column := range columns {
		comments = append(comments, plainComment("COLUMN "+column.GetName(), column.GetComment()))
	}
	return lines(comments)
}

func stripColumnComments(columns []*metadatapb.ColumnMetadata) {
	for _, column := range columns {
		column.Comment = ""
	}
}

func indexComments(indexes []*metadatapb.IndexMetadata) string {
	comments := make([]string, 0, len(indexes))
	for _, index := range indexes {
		comments = append(comments, plainComment("INDEX "+index.GetName(), index.GetComment()))
	}
	return lines(comments)
}

func stripIndexComments(indexes []*metadatapb.IndexMetadata) {
	for _, index := range indexes {
		index.Comment = ""
	}
}

// lines puts the lines that are not empty one below the other.
func lines(all []string) string {
	kept := make([]string, 0, len(all))
	for _, line := range all {
		if line != "" {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// join puts the parts of a definition one blank line apart.
func join(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "\n\n")
}

func plainTriggers(on string, triggers []*metadatapb.TriggerMetadata) string {
	parts := make([]string, 0, len(triggers))
	for _, trigger := range triggers {
		body := strings.TrimSpace(trigger.GetBody())
		context := creationContext{sqlMode: trigger.GetSqlMode(), characterSetClient: trigger.GetCharacterSetClient(), collationConnection: trigger.GetCollationConnection()}.text()
		switch {
		case body == "":
		case len(body) >= 6 && strings.EqualFold(body[:6], "CREATE"):
			// PostgreSQL syncs the whole statement as the body.
			parts = append(parts, join(body, context, extensionMember(trigger.GetSkipDump())))
		default:
			parts = append(parts, join(fmt.Sprintf("TRIGGER %s %s %s ON %s\n%s", trigger.GetName(), trigger.GetTiming(), trigger.GetEvent(), on, body), context, extensionMember(trigger.GetSkipDump())))
		}
	}
	return strings.Join(parts, "\n\n")
}

// creationContext is what a MySQL-family trigger or event was created under:
// the session's SQL mode, client character set, connection collation, and
// time zone. They change how its body parses, compares, and schedules.
type creationContext struct {
	sqlMode             string
	characterSetClient  string
	collationConnection string
	timeZone            string
}

func (c creationContext) text() string {
	var settings []string
	for _, setting := range []struct{ name, value string }{
		{name: "sql_mode", value: c.sqlMode},
		{name: "character_set_client", value: c.characterSetClient},
		{name: "collation_connection", value: c.collationConnection},
		{name: "time_zone", value: c.timeZone},
	} {
		if setting.value != "" {
			settings = append(settings, fmt.Sprintf("%s=%s", setting.name, setting.value))
		}
	}
	if len(settings) == 0 {
		return ""
	}
	return "-- created with " + strings.Join(settings, ", ")
}

// contextLine names one setting a MySQL-family routine was created under.
func contextLine(name string, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("-- created with %s=%s", name, value)
}

// extensionMember marks an object PostgreSQL says an extension owns.
func extensionMember(skipDump bool) string {
	if !skipDump {
		return ""
	}
	return "MEMBER OF AN EXTENSION: the extension creates and upgrades it"
}

// ident writes a name as it is when it is a plain identifier, and quoted
// otherwise, so a name holding a comma or a space stays one name in a list.
func ident(name string) string {
	plain := name != ""
	for i, r := range name {
		if r != '_' && !unicode.IsLetter(r) && (i == 0 || (!unicode.IsDigit(r) && r != '$')) {
			plain = false
			break
		}
	}
	if plain {
		return name
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func idents(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, ident(name))
	}
	return strings.Join(quoted, ", ")
}

// sqlStrings writes values as SQL string literals, so a value holding a
// comma or a quote stays one value.
func sqlStrings(values []string) string {
	literals := make([]string, 0, len(values))
	for _, value := range values {
		literals = append(literals, "'"+strings.ReplaceAll(value, "'", "''")+"'")
	}
	return strings.Join(literals, ", ")
}

// plainFact writes a fact the definition does not carry, such as an owner.
func plainFact(label string, value string) string {
	if value == "" {
		return ""
	}
	return label + ": " + value
}

func plainEventTrigger(trigger *metadatapb.EventTriggerMetadata) string {
	definition := trigger.GetDefinition()
	if definition == "" {
		definition = fmt.Sprintf("EVENT TRIGGER %s ON %s", trigger.GetName(), trigger.GetEvent())
		if tags := trigger.GetTags(); len(tags) > 0 {
			definition += fmt.Sprintf(" WHEN TAG IN (%s)", sqlStrings(tags))
		}
		definition += fmt.Sprintf(" EXECUTE FUNCTION %s()", qualify(trigger.GetFunctionSchema(), trigger.GetFunctionName()))
	}
	if !trigger.GetEnabled() {
		definition += "\nDISABLED"
	}
	return definition
}

func plainTable(engine storepb.Engine, schemaName string, table *metadatapb.TableMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "TABLE %s (\n", qualify(schemaName, table.GetName()))
	for _, column := range table.GetColumns() {
		_, _ = fmt.Fprintf(&b, "  %s\n", plainColumn(engine, column))
	}
	_, _ = b.WriteString(")")
	storage := table.GetEngine()
	// Hive's sync writes HDFS for every table without reading its storage.
	if engine == storepb.Engine_HIVE {
		storage = ""
	}
	for _, option := range []struct{ keyword, value string }{
		{keyword: "ENGINE=", value: storage},
		{keyword: "DEFAULT CHARSET=", value: table.GetCharset()},
		{keyword: "COLLATE=", value: table.GetCollation()},
		{keyword: "", value: table.GetCreateOptions()},
	} {
		if option.value != "" {
			_, _ = fmt.Fprintf(&b, " %s%s", option.keyword, option.value)
		}
	}
	return b.String()
}

func plainIndexes(indexes []*metadatapb.IndexMetadata) string {
	all := make([]string, 0, len(indexes))
	for _, index := range indexes {
		// DynamoDB syncs an empty range index for a table with no range key.
		if index.GetDefinition() == "" && len(index.GetExpressions()) == 0 {
			continue
		}
		all = append(all, plainIndex(index))
	}
	return lines(all)
}

func plainForeignKeys(keys []*metadatapb.ForeignKeyMetadata) string {
	all := make([]string, 0, len(keys))
	for _, key := range keys {
		definition := fmt.Sprintf("FOREIGN KEY %s (%s) REFERENCES %s (%s)", key.GetName(), idents(key.GetColumns()), qualify(key.GetReferencedSchema(), key.GetReferencedTable()), idents(key.GetReferencedColumns()))
		matchType := key.GetMatchType()
		// MySQL and MariaDB report NONE, for a key declared with no MATCH.
		if strings.EqualFold(matchType, "NONE") {
			matchType = ""
		}
		for _, clause := range []struct{ keyword, value string }{
			{keyword: "MATCH", value: matchType},
			{keyword: "ON DELETE", value: key.GetOnDelete()},
			{keyword: "ON UPDATE", value: key.GetOnUpdate()},
		} {
			if clause.value != "" {
				definition += fmt.Sprintf(" %s %s", clause.keyword, clause.value)
			}
		}
		all = append(all, definition)
	}
	return lines(all)
}

func plainCheckConstraints(checks []*metadatapb.CheckConstraintMetadata) string {
	all := make([]string, 0, len(checks))
	for _, check := range checks {
		all = append(all, fmt.Sprintf("CHECK %s %s", check.GetName(), check.GetExpression()))
	}
	return lines(all)
}

func plainExcludeConstraints(excludes []*metadatapb.ExcludeConstraintMetadata) string {
	all := make([]string, 0, len(excludes))
	for _, exclude := range excludes {
		all = append(all, fmt.Sprintf("EXCLUDE %s %s", exclude.GetName(), exclude.GetExpression()))
	}
	return lines(all)
}

// plainPartitions writes only the parts of a partition the sync holds. Hive
// syncs the names alone.
func plainPartitions(partitions []*metadatapb.TablePartitionMetadata) string {
	var all []string
	for _, partition := range partitions {
		definition := "PARTITION " + partition.GetName()
		if kind := partition.GetType(); kind != metadatapb.TablePartitionMetadata_TYPE_UNSPECIFIED {
			definition += " BY " + strings.ReplaceAll(kind.String(), "_", " ")
		}
		if expression := partition.GetExpression(); expression != "" {
			definition += fmt.Sprintf(" (%s)", expression)
		}
		if value := partition.GetValue(); value != "" {
			definition += " VALUES " + value
		}
		// MySQL keeps the n of a PARTITIONS n clause here.
		if count := partition.GetUseDefault(); count != "" && count != "0" {
			definition += " PARTITIONS " + count
		}
		all = append(all, definition)
		if subpartitions := plainPartitions(partition.GetSubpartitions()); subpartitions != "" {
			all = append(all, "  "+strings.ReplaceAll(subpartitions, "\n", "\n  "))
		}
	}
	return lines(all)
}

func plainRules(rules []*metadatapb.RuleMetadata) string {
	all := make([]string, 0, len(rules))
	for _, rule := range rules {
		definition := rule.GetDefinition()
		if definition == "" {
			definition = fmt.Sprintf("RULE %s ON %s WHERE %s DO %s", rule.GetName(), rule.GetEvent(), rule.GetCondition(), rule.GetAction())
		}
		all = append(all, definition)
	}
	return lines(all)
}

// invisibleIndexes names the indexes the optimizer ignores. Only the syncs
// below read visibility; another engine's index reads as invisible by default.
func invisibleIndexes(engine storepb.Engine, indexes []*metadatapb.IndexMetadata) string {
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE, storepb.Engine_ORACLE, storepb.Engine_TIDB:
	default:
		return ""
	}
	var all []string
	for _, index := range indexes {
		if !index.GetVisible() {
			all = append(all, fmt.Sprintf("INDEX %s INVISIBLE", index.GetName()))
		}
	}
	return lines(all)
}

func showIndexes(indexes []*metadatapb.IndexMetadata) {
	for _, index := range indexes {
		index.Visible = true
	}
}

// defaultConstraintNames names the constraints that hold column defaults,
// which SQL Server syncs and a change to a default has to drop by name.
func defaultConstraintNames(columns []*metadatapb.ColumnMetadata) string {
	var all []string
	for _, column := range columns {
		if name := column.GetDefaultConstraintName(); name != "" {
			all = append(all, fmt.Sprintf("DEFAULT CONSTRAINT %s ON COLUMN %s", name, column.GetName()))
		}
	}
	return lines(all)
}

func stripDefaultConstraintNames(columns []*metadatapb.ColumnMetadata) {
	for _, column := range columns {
		column.DefaultConstraintName = ""
	}
}

// disabledRules names the rules that are switched off. Only PostgreSQL syncs
// rules, and it reads whether each one is enabled.
func disabledRules(rules []*metadatapb.RuleMetadata) string {
	var all []string
	for _, rule := range rules {
		if !rule.GetIsEnabled() {
			all = append(all, fmt.Sprintf("RULE %s DISABLED", rule.GetName()))
		}
	}
	return lines(all)
}

func enableRules(rules []*metadatapb.RuleMetadata) {
	for _, rule := range rules {
		rule.IsEnabled = true
	}
}

func plainExternalTable(engine storepb.Engine, schemaName string, table *metadatapb.ExternalTableMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "EXTERNAL TABLE %s", qualify(schemaName, table.GetName()))
	if server := table.GetExternalServerName(); server != "" {
		_, _ = fmt.Fprintf(&b, " ON SERVER %s", server)
	}
	if database := table.GetExternalDatabaseName(); database != "" {
		_, _ = fmt.Fprintf(&b, " DATABASE %s", database)
	}
	_, _ = b.WriteString(" (\n")
	for _, column := range table.GetColumns() {
		_, _ = fmt.Fprintf(&b, "  %s\n", plainColumn(engine, column))
	}
	_, _ = b.WriteString(")")
	return b.String()
}

// syncsNullability reports whether the engine's sync reads column
// nullability. Where it does not, every column would read as NOT NULL, so the
// fallback writes no nullability at all.
func syncsNullability(engine storepb.Engine) bool {
	switch engine {
	case storepb.Engine_HIVE, storepb.Engine_DYNAMODB:
		return false
	default:
		return true
	}
}

func plainColumn(engine storepb.Engine, column *metadatapb.ColumnMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s %s", column.GetName(), column.GetType())
	if value := column.GetCharacterSet(); value != "" {
		_, _ = fmt.Fprintf(&b, " CHARACTER SET %s", value)
	}
	if value := column.GetCollation(); value != "" {
		_, _ = fmt.Fprintf(&b, " COLLATE %s", value)
	}
	if generation := column.GetGeneration(); generation.GetExpression() != "" {
		_, _ = fmt.Fprintf(&b, " GENERATED ALWAYS AS (%s)", generation.GetExpression())
		switch generation.GetType() {
		case metadatapb.GenerationMetadata_TYPE_STORED:
			_, _ = b.WriteString(" STORED")
		case metadatapb.GenerationMetadata_TYPE_VIRTUAL:
			_, _ = b.WriteString(" VIRTUAL")
		default:
		}
	}
	if column.GetIsIdentity() {
		switch column.GetIdentityGeneration() {
		case metadatapb.ColumnMetadata_ALWAYS:
			_, _ = b.WriteString(" GENERATED ALWAYS AS IDENTITY")
		case metadatapb.ColumnMetadata_BY_DEFAULT:
			_, _ = b.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
		default:
			_, _ = b.WriteString(" IDENTITY")
		}
	}
	if !column.GetNullable() && syncsNullability(engine) {
		_, _ = b.WriteString(" NOT NULL")
	}
	// The MySQL-family syncs keep AUTO_INCREMENT in the default. NULL is not
	// written: BigQuery stores it for a column with no default, and the
	// MySQL-family syncs cannot tell an explicit DEFAULT NULL from none.
	switch value := column.GetDefault(); {
	case value == "" || strings.EqualFold(value, "NULL"):
	case strings.EqualFold(value, "AUTO_INCREMENT"):
		_, _ = b.WriteString(" AUTO_INCREMENT")
	default:
		_, _ = fmt.Fprintf(&b, " DEFAULT %s", value)
	}
	if value := column.GetOnUpdate(); value != "" {
		_, _ = fmt.Fprintf(&b, " ON UPDATE %s", value)
	}
	if column.GetIsInvisible() {
		_, _ = b.WriteString(" INVISIBLE")
	}
	return b.String()
}

// plainIndex leaves visibility out: an engine whose sync never reads it would
// show every index as invisible.
func plainIndex(index *metadatapb.IndexMetadata) string {
	kind := "INDEX"
	method := index.GetType()
	// The MySQL-family syncs keep FULLTEXT and SPATIAL as the type. They are
	// kinds of index, and their parts take no prefix length, though the sync
	// records one for a spatial part.
	keyLength := index.GetKeyLength()
	switch {
	case index.GetPrimary():
		kind = "PRIMARY KEY"
	case strings.EqualFold(method, "FULLTEXT"), strings.EqualFold(method, "SPATIAL"):
		kind = strings.ToUpper(method) + " INDEX"
		method = ""
		keyLength = nil
	case index.GetUnique():
		kind = "UNIQUE INDEX"
	default:
	}
	if definition := index.GetDefinition(); definition != "" {
		// Cassandra keeps the options of a secondary index here, as JSON.
		if strings.HasPrefix(strings.TrimSpace(definition), "{") {
			named := fmt.Sprintf("%s %s", kind, index.GetName())
			if method != "" {
				named += " USING " + method
			}
			return named + " OPTIONS " + definition
		}
		return definition
	}
	expressions := make([]string, 0, len(index.GetExpressions()))
	for i, expression := range index.GetExpressions() {
		if i < len(keyLength) && keyLength[i] > 0 {
			expression += fmt.Sprintf("(%d)", keyLength[i])
		}
		if i < len(index.GetDescending()) && index.GetDescending()[i] {
			expression += " DESC"
		}
		expressions = append(expressions, expression)
	}
	definition := fmt.Sprintf("%s %s", kind, index.GetName())
	if method != "" {
		definition += fmt.Sprintf(" USING %s", method)
	}
	definition = fmt.Sprintf("%s (%s)", definition, strings.Join(expressions, ", "))
	// Only ClickHouse syncs the granularity of a data-skipping index.
	if granularity := index.GetGranularity(); granularity > 0 {
		definition += fmt.Sprintf(" GRANULARITY %d", granularity)
	}
	return definition
}

func plainSequence(schemaName string, sequence *metadatapb.SequenceMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "SEQUENCE %s", qualify(schemaName, sequence.GetName()))
	var missing []string
	for _, attribute := range []struct{ keyword, value, required string }{
		{keyword: "AS", value: sequence.GetDataType()},
		{keyword: "START", value: sequence.GetStart(), required: "start"},
		{keyword: "INCREMENT", value: sequence.GetIncrement(), required: "increment"},
		{keyword: "MINVALUE", value: sequence.GetMinValue()},
		{keyword: "MAXVALUE", value: sequence.GetMaxValue()},
		{keyword: "CACHE", value: sequence.GetCacheSize()},
	} {
		switch {
		case attribute.value != "":
			_, _ = fmt.Fprintf(&b, " %s %s", attribute.keyword, attribute.value)
		case attribute.required != "":
			missing = append(missing, attribute.required)
		default:
		}
	}
	if sequence.GetCycle() {
		_, _ = b.WriteString(" CYCLE")
	}
	if table := sequence.GetOwnerTable(); table != "" {
		_, _ = fmt.Fprintf(&b, " OWNED BY %s.%s", ident(table), ident(sequence.GetOwnerColumn()))
	}
	if len(missing) > 0 {
		_, _ = fmt.Fprintf(&b, "\n-- not synced: %s", strings.Join(missing, ", "))
	}
	return b.String()
}

func qualify(schemaName string, name string) string {
	if schemaName == "" {
		return name
	}
	return schemaName + "." + name
}
