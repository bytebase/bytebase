import { create } from "@bufbuild/protobuf";
import { Plus } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { SegmentedControl } from "@/react/components/ui/segmented-control";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  ColumnMetadataSchema,
  IndexMetadataSchema,
  TablePartitionMetadata_Type,
  TablePartitionMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseEngine } from "@/utils";
import { useSchemaEditorContext } from "../context";
import {
  engineSupportsEditIndexes,
  engineSupportsEditTablePartitions,
} from "../core/spec";
import { markUUID } from "./common";
import { IndexesEditor } from "./IndexesEditor";
import { PartitionsEditor } from "./PartitionsEditor";
import { PreviewPane } from "./PreviewPane";
import { TableColumnEditor } from "./TableColumnEditor";

type EditorMode = "COLUMNS" | "INDEXES" | "PARTITIONS";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  searchPattern?: string;
}

export function TableEditor({
  db,
  database,
  schema,
  table,
  searchPattern,
}: Props) {
  const { t } = useTranslation();
  const { readonly, editStatus, scrollStatus, rebuildTree, options } =
    useSchemaEditorContext();

  const [mode, setMode] = useState<EditorMode>("COLUMNS");
  const engine = getDatabaseEngine(db);

  const disableChangeTable = useMemo(() => {
    const schemaStatus = editStatus.getSchemaStatus(db, { schema });
    const tableStatus = editStatus.getTableStatus(db, { schema, table });
    return schemaStatus === "dropped" || tableStatus === "dropped";
  }, [editStatus, db, schema, table]);

  const tableStatus = useMemo(
    () => editStatus.getTableStatus(db, { schema, table }),
    [editStatus, db, schema, table]
  );

  const allowChangePrimaryKeys = tableStatus === "created";

  const showIndexes =
    engineSupportsEditIndexes(engine) || (options?.forceShowIndexes ?? false);

  const showPartitions =
    engineSupportsEditTablePartitions(engine) ||
    (options?.forceShowPartitions ?? false);

  const handleAddColumn = useCallback(() => {
    const column = create(ColumnMetadataSchema, {
      name: "",
      type: "",
      nullable: true,
      hasDefault: false,
      default: "",
      comment: "",
    });
    markUUID(column);
    table.columns.push(column);
    editStatus.markEditStatus(db, { schema, table, column }, "created");
    rebuildTree(false);
    scrollStatus.queuePendingScrollToColumn({
      db,
      metadata: { database, schema, table, column },
    });
  }, [db, database, schema, table, editStatus, rebuildTree, scrollStatus]);

  // Add-index / add-partition handlers live at the TableEditor level so the
  // three "Add …" actions can share a single right-aligned slot in the
  // toolbar instead of jumping into the body panel when the user switches
  // tabs. The child editors no longer render their own add buttons.
  const handleAddIndex = useCallback(() => {
    const index = create(IndexMetadataSchema, {
      name: `idx_${table.name}_${Date.now()}`,
      expressions: [],
      primary: false,
      unique: false,
      comment: "",
    });
    table.indexes.push(index);
    editStatus.markEditStatus(db, { schema, table }, "updated");
  }, [table, editStatus, db, schema]);

  const handleAddPartition = useCallback(() => {
    const firstPartition = table.partitions[0];
    const partition = create(TablePartitionMetadataSchema, {
      name: `p${table.partitions.length}`,
      type: firstPartition?.type ?? TablePartitionMetadata_Type.RANGE,
      expression: firstPartition?.expression ?? "",
      value: "",
      subpartitions: [],
    });
    table.partitions.push(partition);
    editStatus.markEditStatus(db, { schema, table }, "updated");
  }, [table, editStatus, db, schema]);

  const addAction = useMemo(() => {
    if (readonly || disableChangeTable) return null;
    if (mode === "COLUMNS") {
      return {
        label: t("schema-editor.actions.add-column"),
        onClick: handleAddColumn,
      };
    }
    if (mode === "INDEXES") {
      return {
        label: t("schema-editor.actions.add-index"),
        onClick: handleAddIndex,
      };
    }
    return {
      label: t("schema-editor.actions.add-partition"),
      onClick: handleAddPartition,
    };
  }, [
    readonly,
    disableChangeTable,
    mode,
    t,
    handleAddColumn,
    handleAddIndex,
    handleAddPartition,
  ]);

  const markTableStatus = useCallback(
    (status: "updated") => {
      if (tableStatus === "created" || tableStatus === "dropped") return;
      editStatus.markEditStatus(db, { schema, table }, status);
    },
    [tableStatus, editStatus, db, schema, table]
  );

  const modeOptions = useMemo(() => {
    const items: { value: EditorMode; label: string }[] = [
      { value: "COLUMNS", label: t("schema-editor.columns") },
    ];
    if (showIndexes) {
      items.push({ value: "INDEXES", label: t("schema-editor.indexes") });
    }
    if (showPartitions) {
      items.push({
        value: "PARTITIONS",
        label: t("schema-editor.partitions"),
      });
    }
    return items;
  }, [showIndexes, showPartitions, t]);

  return (
    <div className="flex size-full flex-col gap-y-2 overflow-y-hidden pt-2">
      {/* Toolbar */}
      <div className="flex items-center gap-x-2 px-4">
        <SegmentedControl
          value={mode}
          options={modeOptions}
          onValueChange={setMode}
          ariaLabel={t("schema-editor.self")}
          size="sm"
        />
        {addAction && (
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={addAction.onClick}
          >
            <Plus className="mr-1 size-4" />
            {addAction.label}
          </Button>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-hidden px-4">
        {mode === "COLUMNS" && (
          <TableColumnEditor
            db={db}
            database={database}
            schema={schema}
            table={table}
            engine={engine}
            readonly={readonly}
            disableChangeTable={disableChangeTable}
            allowChangePrimaryKeys={allowChangePrimaryKeys}
            searchPattern={searchPattern}
            onMarkTableStatus={markTableStatus}
          />
        )}
        {mode === "INDEXES" && (
          <IndexesEditor
            db={db}
            database={database}
            schema={schema}
            table={table}
            readonly={readonly}
          />
        )}
        {mode === "PARTITIONS" && (
          <PartitionsEditor
            db={db}
            database={database}
            schema={schema}
            table={table}
            readonly={readonly}
          />
        )}
      </div>

      {/* Preview */}
      <PreviewPane db={db} database={database} schema={schema} table={table} />
    </div>
  );
}
