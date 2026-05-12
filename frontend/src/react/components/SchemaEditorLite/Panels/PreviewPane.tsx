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
import { resizeHandleClass, usePersistedDragSize } from "../resize";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}

const EXPANDED_STORAGE_KEY = "bb.schema-editor.preview-expanded";
const HEIGHT_STORAGE_KEY = "bb.schema-editor.preview-height";
const PREVIEW_MIN_HEIGHT = 80;
const PREVIEW_MAX_HEIGHT = 600;
const PREVIEW_DEFAULT_HEIGHT = 160;

export function PreviewPane({ db, database, schema, table }: Props) {
  const { t } = useTranslation();
  const { hidePreview, editStatus } = useSchemaEditorContext();
  const engine = getDatabaseEngine(db);
  const language = languageOfEngineV1(engine);

  const [expanded, setExpanded] = useState(() => {
    try {
      return localStorage.getItem(EXPANDED_STORAGE_KEY) !== "false";
    } catch {
      return true;
    }
  });

  // Drag-to-resize for the preview height. Handle sits on the panel's top
  // edge, so dragging up (smaller Y) grows the panel.
  const { size: height, handleResizeStart } = usePersistedDragSize({
    storageKey: HEIGHT_STORAGE_KEY,
    axis: "y",
    growsToward: "before",
    defaultSize: PREVIEW_DEFAULT_HEIGHT,
    minSize: PREVIEW_MIN_HEIGHT,
    maxSize: PREVIEW_MAX_HEIGHT,
  });

  const [pending, setPending] = useState(false);
  const [ddl, setDdl] = useState("");
  const [error, setError] = useState("");
  const fetchIdRef = useRef(0);

  const toggleExpanded = useCallback(() => {
    setExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem(EXPANDED_STORAGE_KEY, String(next));
      } catch {
        // ignore
      }
      return next;
    });
  }, []);

  // Build mocked metadata (filter dropped columns, wrap in single-table
  // DatabaseMetadata). Depending on editStatus.version (a stable primitive
  // that bumps on any edit) instead of the editStatus object reference
  // prevents this memo from rebuilding on every parent re-render — which
  // would otherwise restart an in-flight getSchemaString fetch and leave
  // the preview perpetually empty.
  const editStatusVersion = editStatus.version;
  const getColumnStatus = editStatus.getColumnStatus;
  const mockedMetadata = useMemo(() => {
    const cloned = cloneDeep(table);
    cloned.columns = cloned.columns.filter((column) => {
      if (!column.name) return false;
      const status = getColumnStatus(db, { schema, table, column });
      return status !== "dropped";
    });
    return create(DatabaseMetadataSchema, {
      name: database.name,
      characterSet: database.characterSet,
      collation: database.collation,
      schemas: [{ name: schema.name, tables: [cloned] }],
    }) as DatabaseMetadata;
  }, [table, db, schema, database, getColumnStatus, editStatusVersion]);

  useEffect(() => {
    if (!expanded || hidePreview) return;
    const id = ++fetchIdRef.current;
    setPending(true);
    setError("");
    (async () => {
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
    })();
  }, [expanded, hidePreview, db, mockedMetadata]);

  if (hidePreview) return null;

  return (
    <div className="flex flex-col border-t border-control-border">
      {expanded && (
        <div
          role="separator"
          aria-orientation="horizontal"
          className={resizeHandleClass("horizontal")}
          onMouseDown={handleResizeStart}
        />
      )}
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
        <div
          className="relative overflow-y-auto"
          style={{ height: `${height}px` }}
        >
          {pending && (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/60">
              <Loader2 className="size-5 animate-spin text-accent" />
            </div>
          )}
          {error ? (
            <pre className="m-2 overflow-auto rounded-xs bg-error/5 p-3 text-xs whitespace-pre-wrap text-error">
              {error}
            </pre>
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
