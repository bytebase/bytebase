import { LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import { Canvas } from "./Canvas/Canvas";
import {
  SchemaDiagramProvider,
  useSchemaDiagramContext,
} from "./common/context";
import { ForeignKeyLine } from "./ER/ForeignKeyLine";
import {
  type ForeignKeyEdge,
  useAutoLayout,
} from "./ER/libs/autoLayout/useAutoLayout";
import { TableNode } from "./ER/TableNode";
import { Navigator } from "./Navigator/Navigator";
import type { EditStatus } from "./types";

interface SchemaDiagramProps {
  database: Database;
  databaseMetadata: DatabaseMetadata;
  editable?: boolean;
  schemaStatus?: (schema: SchemaMetadata) => EditStatus;
  tableStatus?: (table: TableMetadata) => EditStatus;
  columnStatus?: (column: ColumnMetadata) => EditStatus;
  onEditTable?: (schema: SchemaMetadata, table: TableMetadata) => void;
  onEditColumn?: (
    schema: SchemaMetadata,
    table: TableMetadata,
    column: ColumnMetadata,
    target: "name" | "type"
  ) => void;
}

/**
 * React port of `frontend/src/components/SchemaDiagram/SchemaDiagram.vue`.
 * Composes Navigator + Canvas inside the per-instance React context;
 * an inner `<Body>` component runs the ELK layout when metadata /
 * selection changes.
 */
export function SchemaDiagram({
  database,
  databaseMetadata,
  editable = false,
  schemaStatus,
  tableStatus,
  columnStatus,
  onEditTable,
  onEditColumn,
}: SchemaDiagramProps) {
  return (
    <SchemaDiagramProvider
      database={database}
      databaseMetadata={databaseMetadata}
      editable={editable}
      schemaStatus={schemaStatus}
      tableStatus={tableStatus}
      columnStatus={columnStatus}
    >
      <Body
        databaseName={database.name}
        onEditTable={onEditTable}
        onEditColumn={onEditColumn}
      />
    </SchemaDiagramProvider>
  );
}

interface BodyProps {
  databaseName: string;
  onEditTable?: SchemaDiagramProps["onEditTable"];
  onEditColumn?: SchemaDiagramProps["onEditColumn"];
}

function Body({ databaseName, onEditTable, onEditColumn }: BodyProps) {
  const ctx = useSchemaDiagramContext();
  const {
    busy,
    selectedSchemas,
    selectedSchemaNames,
    setSelectedSchemaNames,
    setFocusedTables,
    foreignKeys,
    idOfTable,
    mergeTableRects,
    events,
    databaseMetadata,
  } = ctx;

  // Vue mirrors: pick the first schema as the initial selection, NOT all
  // of them. (Most diagrams target a single schema; users can opt into
  // others via the SchemaSelector.)
  const lastInitializedMetadataRef = useRef<DatabaseMetadata | null>(null);
  useEffect(() => {
    if (lastInitializedMetadataRef.current === databaseMetadata) return;
    lastInitializedMetadataRef.current = databaseMetadata;
    const first = databaseMetadata.schemas[0]?.name ?? "";
    setSelectedSchemaNames([first]);
  }, [databaseMetadata, setSelectedSchemaNames]);

  const [initialized, setInitialized] = useState(false);

  // Edge list for ELK derived from foreignKeys.
  const edges = useMemo<ForeignKeyEdge[]>(
    () =>
      foreignKeys.map((fk) => ({
        fromSchema: fk.from.schema.name,
        fromTable: fk.from.table,
        fromColumn: fk.from.column,
        toSchema: fk.to.schema.name,
        toTable: fk.to.table,
        toColumn: fk.to.column,
      })),
    [foreignKeys]
  );

  const sizeOfTable = useCallback((id: string) => {
    const elem = document.querySelector(`[data-bb-node-id="${id}"]`);
    if (!elem) return null;
    const rect = elem.getBoundingClientRect();
    return { width: rect.width, height: rect.height };
  }, []);

  const runLayout = useAutoLayout({
    selectedSchemas,
    edges,
    idOfTable,
    sizeOfTable,
  });

  // Drive the layout when (a) selectedSchemas changes, (b) the diagram
  // emits "layout" via the context (e.g. external trigger).
  const driveLayout = useCallback(async () => {
    setFocusedTables(new Set());
    // Defer one frame so newly-mounted TableNodes have laid out their
    // DOM before we measure them.
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => resolve())
    );
    const rects = await runLayout();
    if (!rects) return;
    mergeTableRects(rects);
    setInitialized(true);
    requestAnimationFrame(() => {
      void events.emit("fit-view");
      void events.emit("render");
    });
  }, [runLayout, mergeTableRects, setFocusedTables, events]);

  // Re-run layout on selection change.
  // (selectedSchemas is a reference-stable derived from selectedSchemaNames
  // + databaseMetadata, but listing its identity is enough.)
  useEffect(() => {
    if (selectedSchemaNames.length === 0) return;
    void driveLayout();
    // eslint-disable-next-line @typescript-eslint/no-unused-expressions
    selectedSchemas;
  }, [selectedSchemas, selectedSchemaNames, driveLayout]);

  // External `layout` trigger.
  useEffect(() => {
    const off = events.on("layout", () => {
      void driveLayout();
    });
    return () => {
      off();
    };
  }, [events, driveLayout]);

  // External `edit-table` / `edit-column` events bridge to caller props.
  useEffect(() => {
    if (!onEditTable) return;
    const off = events.on("edit-table", ({ schema, table }) => {
      onEditTable(schema, table);
    });
    return () => {
      off();
    };
  }, [events, onEditTable]);

  useEffect(() => {
    if (!onEditColumn) return;
    const off = events.on(
      "edit-column",
      ({ schema, table, column, target }) => {
        onEditColumn(schema, table, column, target);
      }
    );
    return () => {
      off();
    };
  }, [events, onEditColumn]);

  const screenshotFilename = useCallback(() => {
    const { databaseName: shortName } =
      extractDatabaseResourceName(databaseName);
    return `${shortName}.png`;
  }, [databaseName]);

  return (
    <div className="w-full h-full relative overflow-hidden flex">
      <Navigator />

      <div className="flex-1 relative">
        <Canvas screenshotFilename={screenshotFilename}>
          {selectedSchemas.map((schema) =>
            schema.tables.map((table) => (
              <TableNode key={idOfTable(table)} schema={schema} table={table} />
            ))
          )}
          {initialized &&
            foreignKeys.map((fk, i) => <ForeignKeyLine key={i} fk={fk} />)}
        </Canvas>

        {(busy || !initialized) && (
          <div
            className="absolute inset-0 bg-white/40 flex items-center justify-center pointer-events-none"
            data-screenshot-hide
          >
            <LoaderCircle className="size-6 text-accent animate-spin" />
          </div>
        )}
      </div>
    </div>
  );
}
