package aireview

import (
	"fmt"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"google.golang.org/protobuf/proto"

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
			return join(plainEventTrigger(trigger), plainComment("", trigger.GetComment()))
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
	add := func(kind string, name string, signature string, statistics *tableStatistics, render func() string) {
		c.objects = append(c.objects, &catalogObject{kind: kind, schema: schemaName, name: name, signature: signature, statistics: statistics, render: render})
	}
	owned := make(map[string][]*metadatapb.SequenceMetadata)
	for _, sequence := range s.GetSequences() {
		if table := sequence.GetOwnerTable(); table != "" {
			owned[table] = append(owned[table], sequence)
		}
	}
	for _, table := range s.GetTables() {
		add(kindTable, table.GetName(), "", statisticsOf(table), func() string {
			return c.tableDefinition(schemaName, table, owned[table.GetName()])
		})
	}
	for _, table := range s.GetExternalTables() {
		add(kindExternalTable, table.GetName(), "", nil, func() string {
			return join(plainExternalTable(schemaName, table), columnComments(table.GetColumns()))
		})
	}
	for _, view := range s.GetViews() {
		add(kindView, view.GetName(), "", nil, func() string { return c.viewDefinition(schemaName, view) })
	}
	for _, view := range s.GetMaterializedViews() {
		add(kindMaterializedView, view.GetName(), "", nil, func() string { return c.materializedViewDefinition(schemaName, view) })
	}
	for _, function := range s.GetFunctions() {
		add(kindFunction, function.GetName(), function.GetSignature(), nil, func() string {
			render := func(function *metadatapb.FunctionMetadata) (string, error) {
				return schema.GetFunctionDefinition(c.engine, schemaName, function)
			}
			return complete(function, render, function.GetDefinition, extra[*metadatapb.FunctionMetadata]{
				strip: func(function *metadatapb.FunctionMetadata) { function.Comment = "" },
				text:  plainComment("", function.GetComment()),
			})
		})
	}
	for _, procedure := range s.GetProcedures() {
		add(kindProcedure, procedure.GetName(), procedure.GetSignature(), nil, func() string {
			render := func(procedure *metadatapb.ProcedureMetadata) (string, error) {
				return schema.GetProcedureDefinition(c.engine, schemaName, procedure)
			}
			return complete(procedure, render, procedure.GetDefinition, extra[*metadatapb.ProcedureMetadata]{
				strip: func(procedure *metadatapb.ProcedureMetadata) { procedure.Comment = "" },
				text:  plainComment("", procedure.GetComment()),
			})
		})
	}
	for _, sequence := range s.GetSequences() {
		add(kindSequence, sequence.GetName(), "", nil, func() string {
			render := func(sequence *metadatapb.SequenceMetadata) (string, error) {
				return schema.GetSequenceDefinition(c.engine, schemaName, sequence)
			}
			plain := func() string { return plainSequence(schemaName, sequence) }
			return complete(sequence, render, plain, extra[*metadatapb.SequenceMetadata]{
				strip: func(sequence *metadatapb.SequenceMetadata) { sequence.Comment = "" },
				text:  plainComment("", sequence.GetComment()),
			})
		})
	}
	for _, pkg := range s.GetPackages() {
		add(kindPackage, pkg.GetName(), "", nil, pkg.GetDefinition)
	}
	for _, event := range s.GetEvents() {
		add(kindEvent, event.GetName(), "", nil, func() string {
			return join(event.GetDefinition(), plainComment("", event.GetComment()))
		})
	}
	for _, stream := range s.GetStreams() {
		add(kindStream, stream.GetName(), "", nil, func() string {
			return join(stream.GetDefinition(), plainComment("", stream.GetComment()))
		})
	}
	for _, task := range s.GetTasks() {
		add(kindTask, task.GetName(), "", nil, func() string {
			return join(task.GetDefinition(), plainComment("", task.GetComment()))
		})
	}
	for _, enum := range s.GetEnumTypes() {
		add(kindEnumType, enum.GetName(), "", nil, func() string {
			definition := fmt.Sprintf("ENUM TYPE %s: %s", qualify(schemaName, enum.GetName()), strings.Join(enum.GetValues(), ", "))
			return join(definition, plainComment("", enum.GetComment()))
		})
	}
	for _, composite := range s.GetCompositeTypes() {
		add(kindCompositeType, composite.GetName(), "", nil, func() string {
			attributes := make([]string, 0, len(composite.GetAttributes()))
			comments := []string{plainComment("", composite.GetComment())}
			for _, attribute := range composite.GetAttributes() {
				attributes = append(attributes, attribute.GetName()+" "+attribute.GetType())
				comments = append(comments, plainComment("ATTRIBUTE "+attribute.GetName(), attribute.GetComment()))
			}
			definition := fmt.Sprintf("COMPOSITE TYPE %s (%s)", qualify(schemaName, composite.GetName()), strings.Join(attributes, ", "))
			return join(definition, lines(comments))
		})
	}
}

// statisticsOf returns the statistics the sync holds of a table. Not every
// engine syncs them, and then there are none to show: zeros would say the
// table is empty.
func statisticsOf(table *metadatapb.TableMetadata) *tableStatistics {
	if table.GetRowCount() == 0 && table.GetDataSize() == 0 && table.GetIndexSize() == 0 {
		return nil
	}
	return &tableStatistics{Rows: table.GetRowCount(), DataBytes: table.GetDataSize(), IndexBytes: table.GetIndexSize()}
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
	plain := func() string { return plainTable(schemaName, table) }
	parts := []extra[*tablepb]{
		{strip: func(table *tablepb) { table.Indexes = nil }, text: plainIndexes(table.GetIndexes())},
		{strip: func(table *tablepb) { table.ForeignKeys = nil }, text: plainForeignKeys(table.GetForeignKeys())},
		{strip: func(table *tablepb) { table.CheckConstraints = nil }, text: plainCheckConstraints(table.GetCheckConstraints())},
		{strip: func(table *tablepb) { table.ExcludeConstraints = nil }, text: plainExcludeConstraints(table.GetExcludeConstraints())},
		{strip: func(table *tablepb) { table.Partitions = nil }, text: plainPartitions(table.GetPartitions())},
		{strip: func(table *tablepb) { table.Rules = nil }, text: plainRules(table.GetRules())},
		{strip: func(table *tablepb) { table.Comment = "" }, text: plainComment("", table.GetComment())},
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
		switch {
		case body == "":
		case len(body) >= 6 && strings.EqualFold(body[:6], "CREATE"):
			// PostgreSQL syncs the whole statement as the body.
			parts = append(parts, body)
		default:
			parts = append(parts, fmt.Sprintf("TRIGGER %s %s %s ON %s\n%s", trigger.GetName(), trigger.GetTiming(), trigger.GetEvent(), on, body))
		}
	}
	return strings.Join(parts, "\n\n")
}

func plainEventTrigger(trigger *metadatapb.EventTriggerMetadata) string {
	definition := trigger.GetDefinition()
	if definition == "" {
		definition = fmt.Sprintf("EVENT TRIGGER %s ON %s", trigger.GetName(), trigger.GetEvent())
		if tags := trigger.GetTags(); len(tags) > 0 {
			definition += fmt.Sprintf(" WHEN TAG IN (%s)", strings.Join(tags, ", "))
		}
		definition += fmt.Sprintf(" EXECUTE FUNCTION %s()", qualify(trigger.GetFunctionSchema(), trigger.GetFunctionName()))
	}
	if !trigger.GetEnabled() {
		definition += "\nDISABLED"
	}
	return definition
}

func plainTable(schemaName string, table *metadatapb.TableMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "TABLE %s (\n", qualify(schemaName, table.GetName()))
	for _, column := range table.GetColumns() {
		_, _ = fmt.Fprintf(&b, "  %s\n", plainColumn(column))
	}
	_, _ = b.WriteString(")")
	return b.String()
}

func plainIndexes(indexes []*metadatapb.IndexMetadata) string {
	all := make([]string, 0, len(indexes))
	for _, index := range indexes {
		all = append(all, plainIndex(index))
	}
	return lines(all)
}

func plainForeignKeys(keys []*metadatapb.ForeignKeyMetadata) string {
	all := make([]string, 0, len(keys))
	for _, key := range keys {
		all = append(all, fmt.Sprintf("FOREIGN KEY %s (%s) REFERENCES %s (%s)", key.GetName(), strings.Join(key.GetColumns(), ", "), qualify(key.GetReferencedSchema(), key.GetReferencedTable()), strings.Join(key.GetReferencedColumns(), ", ")))
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

func plainPartitions(partitions []*metadatapb.TablePartitionMetadata) string {
	var all []string
	for _, partition := range partitions {
		all = append(all, fmt.Sprintf("PARTITION %s BY %s (%s) VALUES %s", partition.GetName(), partition.GetType(), partition.GetExpression(), partition.GetValue()))
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

func plainExternalTable(schemaName string, table *metadatapb.ExternalTableMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "EXTERNAL TABLE %s ON SERVER %s (\n", qualify(schemaName, table.GetName()), table.GetExternalServerName())
	for _, column := range table.GetColumns() {
		_, _ = fmt.Fprintf(&b, "  %s\n", plainColumn(column))
	}
	_, _ = b.WriteString(")")
	return b.String()
}

func plainColumn(column *metadatapb.ColumnMetadata) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s %s", column.GetName(), column.GetType())
	if !column.GetNullable() {
		_, _ = b.WriteString(" NOT NULL")
	}
	if value := column.GetDefault(); value != "" {
		_, _ = fmt.Fprintf(&b, " DEFAULT %s", value)
	}
	return b.String()
}

func plainIndex(index *metadatapb.IndexMetadata) string {
	if definition := index.GetDefinition(); definition != "" {
		return definition
	}
	kind := "INDEX"
	switch {
	case index.GetPrimary():
		kind = "PRIMARY KEY"
	case index.GetUnique():
		kind = "UNIQUE INDEX"
	default:
	}
	return fmt.Sprintf("%s %s (%s)", kind, index.GetName(), strings.Join(index.GetExpressions(), ", "))
}

func plainSequence(schemaName string, sequence *metadatapb.SequenceMetadata) string {
	definition := fmt.Sprintf("SEQUENCE %s AS %s START %s INCREMENT %s", qualify(schemaName, sequence.GetName()), sequence.GetDataType(), sequence.GetStart(), sequence.GetIncrement())
	if table := sequence.GetOwnerTable(); table != "" {
		definition += fmt.Sprintf(" OWNED BY %s.%s", table, sequence.GetOwnerColumn())
	}
	return definition
}

func qualify(schemaName string, name string) string {
	if schemaName == "" {
		return name
	}
	return schemaName + "." + name
}
