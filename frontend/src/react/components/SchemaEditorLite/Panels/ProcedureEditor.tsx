import { useCallback, useMemo } from "react";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import type {
  Database,
  DatabaseMetadata,
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { getDatabaseEngine } from "@/utils";
import { useSchemaEditorContext } from "../context";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
}

export function ProcedureEditor({
  db,
  database: _database,
  schema,
  procedure,
}: Props) {
  const { readonly, editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);
  const language = languageOfEngineV1(engine);

  const status = useMemo(
    () => editStatus.getProcedureStatus(db, { schema, procedure }),
    [editStatus, db, schema, procedure]
  );

  const schemaStatus = useMemo(
    () => editStatus.getSchemaStatus(db, { schema }),
    [editStatus, db, schema]
  );

  const disallowChange =
    readonly || schemaStatus === "dropped" || status === "dropped";

  const handleUpdateDefinition = useCallback(
    (code: string) => {
      procedure.definition = code;
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, procedure }, "updated");
      }
    },
    [procedure, status, editStatus, db, schema]
  );

  return (
    <div className="flex size-full flex-col">
      <MonacoEditor
        content={procedure.definition}
        language={language}
        readOnly={disallowChange}
        className="flex-1"
        options={{
          minimap: { enabled: false },
          lineNumbers: "on",
          scrollBeyondLastLine: false,
        }}
        onChange={handleUpdateDefinition}
      />
    </div>
  );
}
