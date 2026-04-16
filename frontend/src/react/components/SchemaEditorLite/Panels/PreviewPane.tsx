import { cloneDeep } from "lodash-es";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { generateDiffDDL } from "@/components/SchemaEditorLite/common";
import { ReadonlyMonaco } from "@/react/components/monaco/ReadonlyMonaco";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { languageOfEngineV1 } from "@/types/sqlEditor/editor";
import { getDatabaseEngine } from "@/utils";
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

  // Build mocked metadata (filter dropped columns)
  const mockedTable = useMemo(() => {
    const cloned = cloneDeep(table);
    cloned.columns = cloned.columns.filter((column) => {
      const status = editStatus.getColumnStatus(db, {
        schema,
        table,
        column,
      });
      return status !== "dropped";
    });
    return cloned;
  }, [table, editStatus, db, schema]);

  // Fetch DDL preview
  const fetchDDL = useCallback(async () => {
    const id = ++fetchIdRef.current;
    setPending(true);
    setError("");
    try {
      const result = await generateDiffDDL({
        database: db,
        sourceMetadata: database,
        targetMetadata: database,
      });
      if (id !== fetchIdRef.current) return;
      if (result.errors.length > 0) {
        setError(result.errors.join("\n"));
      } else {
        setDdl(result.statement);
      }
    } catch (err) {
      if (id !== fetchIdRef.current) return;
      setError(String(err));
    } finally {
      if (id === fetchIdRef.current) {
        setPending(false);
      }
    }
  }, [db, database]);

  useEffect(() => {
    if (expanded && !hidePreview) {
      fetchDDL();
    }
  }, [expanded, hidePreview, fetchDDL, mockedTable]);

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
