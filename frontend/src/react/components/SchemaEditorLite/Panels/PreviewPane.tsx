import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { databaseServiceClientConnect } from "@/connect";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  DatabaseMetadataSchema,
  GetSchemaStringRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { getDatabaseEngine } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/connect";
import { useSchemaEditorContext } from "../context";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}

const STORAGE_KEY = "bb.schema-editor.preview-expanded";

export function PreviewPane({ db, database, schema, table }: Props) {
  const { t } = useTranslation();
  const { hidePreview, editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);
  const language = languageOfEngineV1(engine);

  const [expanded, setExpanded] = useState(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) !== "false";
    } catch {
      return true;
    }
  });

  const [pending, setPending] = useState(false);
  const [ddl, setDdl] = useState("");
  const [error, setError] = useState("");
  const fetchIdRef = useRef(0);

  const toggleExpanded = useCallback(() => {
    setExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(STORAGE_KEY, String(next));
      } catch {
        // ignore
      }
      return next;
    });
  }, []);

  // Build mocked metadata (filter dropped columns, wrap in single-table DatabaseMetadata)
  const mockedMetadata = useMemo(() => {
    const cloned = cloneDeep(table);
    cloned.columns = cloned.columns.filter((column) => {
      if (!column.name) return false;
      const status = editStatus.getColumnStatus(db, {
        schema,
        table,
        column,
      });
      return status !== "dropped";
    });
    return create(DatabaseMetadataSchema, {
      name: database.name,
      characterSet: database.characterSet,
      collation: database.collation,
      schemas: [{ name: schema.name, tables: [cloned] }],
    }) as DatabaseMetadata;
  }, [table, editStatus, db, schema, database]);

  // Fetch schema string for the mocked metadata
  const fetchSchemaString = useCallback(async () => {
    const id = ++fetchIdRef.current;
    setPending(true);
    setError("");
    try {
      const request = create(GetSchemaStringRequestSchema, {
        name: `${db.name}/schemaString`,
        metadata: mockedMetadata,
      });
      const response =
        await databaseServiceClientConnect.getSchemaString(request);
      if (id !== fetchIdRef.current) return;
      setDdl(response.schemaString);
    } catch (err) {
      if (id !== fetchIdRef.current) return;
      setError(extractGrpcErrorMessage(err));
    } finally {
      if (id === fetchIdRef.current) {
        setPending(false);
      }
    }
  }, [db, mockedMetadata]);

  useEffect(() => {
    if (expanded && !hidePreview) {
      fetchSchemaString();
    }
  }, [expanded, hidePreview, fetchSchemaString]);

  if (hidePreview) return null;

  return (
    <div className="border-t border-control-border">
      <button
        type="button"
        className="flex w-full items-center gap-x-1 px-4 py-1.5 text-xs font-medium text-control-light hover:text-control"
        onClick={toggleExpanded}
      >
        {expanded ? (
          <ChevronDown className="size-3.5" />
        ) : (
          <ChevronRight className="size-3.5" />
        )}
        {t("schema-editor.preview")}
      </button>
      {expanded && (
        <div className="relative h-32 overflow-hidden">
          {pending && (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-white/60">
              <Loader2 className="size-5 animate-spin text-accent" />
            </div>
          )}
          {error ? (
            <pre className="p-2 text-xs text-error">{error}</pre>
          ) : (
            <ReadonlyMonaco
              content={ddl}
              language={language}
              className="size-full"
            />
          )}
        </div>
      )}
    </div>
  );
}
