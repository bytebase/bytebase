import { useCallback, useMemo } from "react";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import type {
  Database,
  DatabaseMetadata,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { getDatabaseEngine } from "@/utils";
import { useSchemaEditorContext } from "../context";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func: FunctionMetadata;
}

export function FunctionEditor({
  db,
  database: _database,
  schema,
  func,
}: Props) {
  const { readonly, editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);
  const language = languageOfEngineV1(engine);

  const status = useMemo(
    () => editStatus.getFunctionStatus(db, { schema, function: func }),
    [editStatus, db, schema, func]
  );

  const schemaStatus = useMemo(
    () => editStatus.getSchemaStatus(db, { schema }),
    [editStatus, db, schema]
  );

  const disallowChange =
    readonly || schemaStatus === "dropped" || status === "dropped";

  const handleUpdateDefinition = useCallback(
    (code: string) => {
      func.definition = code;
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, function: func }, "updated");
      }
    },
    [func, status, editStatus, db, schema]
  );

  return (
    <div className="flex size-full flex-col">
      <MonacoEditor
        content={func.definition}
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
