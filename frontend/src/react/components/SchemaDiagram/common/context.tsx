import Emittery from "emittery";
import { uniqueId } from "lodash-es";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type {
  EditStatus,
  ForeignKey,
  Geometry,
  Point,
  Rect,
  SchemaDiagramContext,
  SchemaDiagramEvents,
} from "../types";

const Context = createContext<SchemaDiagramContext | null>(null);

interface SchemaDiagramProviderProps {
  database: Database;
  databaseMetadata: DatabaseMetadata;
  editable?: boolean;
  /**
   * When true, the provider is mounted for the screenshot off-screen
   * pass — descendants should skip side effects (e.g. registering
   * geometries) that would otherwise pollute the live diagram's state.
   */
  dummy?: boolean;
  schemaStatus?: (schema: SchemaMetadata) => EditStatus;
  tableStatus?: (table: TableMetadata) => EditStatus;
  columnStatus?: (column: ColumnMetadata) => EditStatus;
  children: ReactNode;
}

const DEFAULT_RECT: Rect = { x: 0, y: 0, width: 0, height: 0 };

/**
 * Per-instance React context for SchemaDiagram. Owns the Emittery event
 * bus + reactive state used by Navigator / Canvas / ER children. Mounted
 * once per `<SchemaDiagram>` instance — see Stage 19 design doc §4.
 */
export function SchemaDiagramProvider({
  database,
  databaseMetadata,
  editable = false,
  dummy = false,
  schemaStatus: schemaStatusProp,
  tableStatus: tableStatusProp,
  columnStatus: columnStatusProp,
  children,
}: SchemaDiagramProviderProps) {
  // Stable singletons for this instance.
  const events = useMemo(() => new Emittery<SchemaDiagramEvents>(), []);
  const geometriesRef = useRef<Set<Geometry>>(new Set());
  const tableIdsRef = useRef<Map<TableMetadata, string>>(new Map());

  // Reactive state.
  const [zoom, setZoom] = useState(1);
  const [position, setPosition] = useState<Point>({ x: 0, y: 0 });
  const [panning, setPanning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [selectedSchemaNames, setSelectedSchemaNames] = useState<string[]>([]);
  const [focusedTables, setFocusedTables] = useState<Set<TableMetadata>>(
    () => new Set()
  );
  const [tableRects, setTableRects] = useState<Map<string, Rect>>(
    () => new Map()
  );
  // Bumped by `addGeometry` / `removeGeometry` so consumers that read
  // the live Set get a fresh render after each mutation.
  const [, setGeometryVersion] = useState(0);

  // Initialize selected schemas to the full list whenever metadata changes.
  useEffect(() => {
    setSelectedSchemaNames(databaseMetadata.schemas.map((s) => s.name));
  }, [databaseMetadata]);

  const selectedSchemas = useMemo(() => {
    const set = new Set(selectedSchemaNames);
    return databaseMetadata.schemas.filter((s) => set.has(s.name));
  }, [databaseMetadata, selectedSchemaNames]);

  const foreignKeys = useMemo<ForeignKey[]>(() => {
    // Iterate ONLY `selectedSchemas` for both endpoints. If a user
    // deselects a schema, FK edges that reference it are dropped — same
    // semantics as Vue's `find()` helper. Otherwise ELK gets edges whose
    // endpoints aren't in the node list, which destabilizes layout.
    const out: ForeignKey[] = [];
    for (const schema of selectedSchemas) {
      for (const table of schema.tables) {
        for (const fk of table.foreignKeys) {
          if (fk.columns.length === 0) continue;
          const referencedSchema = selectedSchemas.find(
            (s) => s.name === fk.referencedSchema
          );
          if (!referencedSchema) continue;
          const referencedTable = referencedSchema.tables.find(
            (t) => t.name === fk.referencedTable
          );
          if (!referencedTable) continue;
          for (let i = 0; i < fk.columns.length; i++) {
            const column = fk.columns[i];
            const referencedColumn = fk.referencedColumns[i] ?? "";
            out.push({
              from: { schema, table, column },
              to: {
                schema: referencedSchema,
                table: referencedTable,
                column: referencedColumn,
              },
              metadata: fk,
            });
          }
        }
      }
    }
    return out;
  }, [selectedSchemas]);

  const idOfTable = useCallback((table: TableMetadata) => {
    let id = tableIdsRef.current.get(table);
    if (!id) {
      id = uniqueId("bb-table-");
      tableIdsRef.current.set(table, id);
    }
    return id;
  }, []);

  const rectOfTable = useCallback(
    (table: TableMetadata) => {
      const id = idOfTable(table);
      return tableRects.get(id) ?? DEFAULT_RECT;
    },
    [idOfTable, tableRects]
  );

  const updateTableRect = useCallback((id: string, rect: Rect) => {
    setTableRects((prev) => {
      const next = new Map(prev);
      next.set(id, rect);
      return next;
    });
  }, []);

  const mergeTableRects = useCallback((rects: Map<string, Rect>) => {
    setTableRects((prev) => {
      const next = new Map(prev);
      for (const [id, rect] of rects) {
        next.set(id, rect);
      }
      return next;
    });
  }, []);

  const addGeometry = useCallback((g: Geometry) => {
    geometriesRef.current.add(g);
    setGeometryVersion((v) => v + 1);
  }, []);
  const removeGeometry = useCallback((g: Geometry) => {
    geometriesRef.current.delete(g);
    setGeometryVersion((v) => v + 1);
  }, []);

  const render = useCallback(() => {
    void events.emit("render");
  }, [events]);
  const layout = useCallback(() => {
    void events.emit("layout");
  }, [events]);

  const schemaStatus = useCallback<(schema: SchemaMetadata) => EditStatus>(
    (schema) => schemaStatusProp?.(schema) ?? "normal",
    [schemaStatusProp]
  );
  const tableStatus = useCallback<(table: TableMetadata) => EditStatus>(
    (table) => tableStatusProp?.(table) ?? "normal",
    [tableStatusProp]
  );
  const columnStatus = useCallback<(column: ColumnMetadata) => EditStatus>(
    (column) => columnStatusProp?.(column) ?? "normal",
    [columnStatusProp]
  );

  const value = useMemo<SchemaDiagramContext>(
    () => ({
      database,
      databaseMetadata,
      editable,
      foreignKeys,
      dummy,
      busy,
      zoom,
      position,
      panning,
      geometries: geometriesRef.current,
      focusedTables,
      selectedSchemaNames,
      selectedSchemas,
      setZoom,
      setPosition,
      setPanning,
      setBusy,
      setSelectedSchemaNames,
      setFocusedTables,
      addGeometry,
      removeGeometry,
      updateTableRect,
      mergeTableRects,
      idOfTable,
      rectOfTable,
      schemaStatus,
      tableStatus,
      columnStatus,
      render,
      layout,
      events,
    }),
    [
      database,
      databaseMetadata,
      editable,
      foreignKeys,
      dummy,
      busy,
      zoom,
      position,
      panning,
      focusedTables,
      selectedSchemaNames,
      selectedSchemas,
      addGeometry,
      removeGeometry,
      updateTableRect,
      mergeTableRects,
      idOfTable,
      rectOfTable,
      schemaStatus,
      tableStatus,
      columnStatus,
      render,
      layout,
      events,
    ]
  );

  return <Context value={value}>{children}</Context>;
}

export function useSchemaDiagramContext(): SchemaDiagramContext {
  const ctx = useContext(Context);
  if (!ctx) {
    throw new Error(
      "useSchemaDiagramContext must be called inside <SchemaDiagramProvider>"
    );
  }
  return ctx;
}

/**
 * Register a geometry with the parent SchemaDiagram so the canvas can
 * include it when computing the bbox for `fit-view`. Mirrors Vue's
 * `useGeometry(geometry)` composable. Skipped when the diagram is in
 * dummy (off-screen screenshot) mode.
 */
export function useGeometry(geometry: Geometry) {
  const ctx = useSchemaDiagramContext();
  const dummy = ctx.dummy;
  const addGeometry = ctx.addGeometry;
  const removeGeometry = ctx.removeGeometry;
  useEffect(() => {
    if (dummy) return;
    addGeometry(geometry);
    return () => {
      removeGeometry(geometry);
    };
  }, [geometry, dummy, addGeometry, removeGeometry]);
}
