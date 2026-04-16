import { Plus } from "lucide-react";
import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Combobox } from "@/react/components/ui/combobox";
import type {
  Database,
  DatabaseMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getDatabaseEngine, hasSchemaProperty } from "@/utils";
import { useSchemaEditorContext } from "../context";
import { TableList } from "./TableList";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  selectedSchemaName: string | undefined;
  onSelectedSchemaNameChange: (name: string | undefined) => void;
  searchPattern?: string;
}

export function DatabaseEditor({
  db,
  database,
  selectedSchemaName,
  onSelectedSchemaNameChange,
  searchPattern,
}: Props) {
  const { t } = useTranslation();
  const { readonly, tabs, editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);

  const shouldShowSchemaSelector = hasSchemaProperty(engine);

  const schemaOptions = useMemo(() => {
    return database.schemas.map((s) => ({
      label: s.name,
      value: s.name,
    }));
  }, [database.schemas]);

  const selectedSchema = useMemo(() => {
    return database.schemas.find((s) => s.name === selectedSchemaName);
  }, [database.schemas, selectedSchemaName]);

  // Auto-select first schema if current selection is invalid
  useEffect(() => {
    if (!shouldShowSchemaSelector) return;
    if (
      selectedSchemaName &&
      database.schemas.some((s) => s.name === selectedSchemaName)
    ) {
      return;
    }
    if (database.schemas.length > 0) {
      onSelectedSchemaNameChange(database.schemas[0].name);
    }
  }, [
    shouldShowSchemaSelector,
    selectedSchemaName,
    database.schemas,
    onSelectedSchemaNameChange,
  ]);

  // For single-schema engines, use the first (and only) schema
  const effectiveSchema = useMemo(() => {
    if (shouldShowSchemaSelector) return selectedSchema;
    return database.schemas[0];
  }, [shouldShowSchemaSelector, selectedSchema, database.schemas]);

  const allowCreateTable = useMemo(() => {
    if (!effectiveSchema) return false;
    const status = editStatus.getSchemaStatus(db, {
      schema: effectiveSchema,
    });
    return status !== "dropped";
  }, [effectiveSchema, editStatus, db]);

  const handleEditTable = useCallback(
    (table: { name: string }) => {
      if (!effectiveSchema) return;
      const tableMetadata = effectiveSchema.tables.find(
        (t) => t.name === table.name
      );
      if (!tableMetadata) return;
      tabs.addTab({
        type: "table",
        database: db,
        metadata: {
          database,
          schema: effectiveSchema,
          table: tableMetadata,
        },
      });
    },
    [effectiveSchema, tabs, db, database]
  );

  if (!effectiveSchema) {
    return (
      <div className="flex size-full items-center justify-center text-sm text-control-light">
        No schema available
      </div>
    );
  }

  return (
    <div className="flex size-full flex-col overflow-y-hidden">
      <div className="flex items-center gap-x-2 border-b border-control-border px-4 py-2">
        {shouldShowSchemaSelector && (
          <Combobox
            value={selectedSchemaName ?? ""}
            onChange={(val) => onSelectedSchemaNameChange(val as string)}
            options={schemaOptions}
            className="w-48"
          />
        )}
        {!readonly && allowCreateTable && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              // Will be wired to TableNameDialog in T22
            }}
          >
            <Plus className="mr-1 size-4" />
            {t("schema-editor.actions.create-table")}
          </Button>
        )}
      </div>
      <div className="flex-1 overflow-y-auto">
        <TableList
          db={db}
          database={database}
          schema={effectiveSchema}
          tables={effectiveSchema.tables}
          searchPattern={searchPattern}
          onEditTable={handleEditTable}
        />
      </div>
    </div>
  );
}
