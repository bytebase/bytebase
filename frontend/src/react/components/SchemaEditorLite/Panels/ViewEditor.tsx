import { useCallback, useMemo } from "react";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { getDatabaseEngine } from "@/utils";
import { useSchemaEditorContext } from "../context";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}

export function ViewEditor({ db, schema, view }: Props) {
  const { readonly, editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);
  const language = languageOfEngineV1(engine);

  const status = useMemo(
    () => editStatus.getViewStatus(db, { schema, view }),
    [editStatus, db, schema, view]
  );

  const schemaStatus = useMemo(
    () => editStatus.getSchemaStatus(db, { schema }),
    [editStatus, db, schema]
  );

  const disallowChange =
    readonly || schemaStatus === "dropped" || status === "dropped";

  const handleUpdateDefinition = useCallback(
    (code: string) => {
      view.definition = code;
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, view }, "updated");
      }
    },
    [view, status, editStatus, db, schema]
  );

  return (
    <div className="flex size-full flex-col">
      <MonacoEditor
        content={view.definition}
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
