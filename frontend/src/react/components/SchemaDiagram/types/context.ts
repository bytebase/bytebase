import type Emittery from "emittery";
import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { EditStatus } from "./edit";
import type { Geometry, Point, Rect } from "./geometry";
import type { ForeignKey } from "./schema";

/**
 * React port of the Vue `SchemaDiagramContext`. Values are stored as plain
 * fields (held by React state inside the provider) instead of Vue `Ref<T>`,
 * matching the per-instance React-context pattern decided in the Stage 19
 * design doc.
 *
 * Mutators are exposed as methods rather than direct setters so the
 * provider can scope updates inside `useState`/`useRef` correctly. Methods
 * are stable (wrapped in `useCallback`) so descendants can list them in
 * effect deps without re-running on every render.
 */
export type SchemaDiagramContext = {
  // Props (set once by the SchemaDiagram root from its own props)
  database: Database;
  databaseMetadata: DatabaseMetadata;
  editable: boolean;

  // Reactive state (mutates over time, drives renders)
  foreignKeys: ForeignKey[];
  dummy: boolean;
  busy: boolean;
  zoom: number;
  position: Point;
  panning: boolean;
  /**
   * Mutable Set tracked across descendants via the `useGeometry` hook.
   * It's a live reference (not a copy) so consumers can read it inside
   * a single render without triggering an extra round-trip; writes go
   * through `addGeometry` / `removeGeometry`.
   */
  geometries: Set<Geometry>;
  focusedTables: Set<TableMetadata>;
  selectedSchemaNames: string[];
  selectedSchemas: SchemaMetadata[];

  // State mutators (stable references)
  setZoom: (zoom: number) => void;
  setPosition: (position: Point) => void;
  setPanning: (panning: boolean) => void;
  setBusy: (busy: boolean) => void;
  setSelectedSchemaNames: (names: string[]) => void;
  setFocusedTables: (tables: Set<TableMetadata>) => void;
  addGeometry: (g: Geometry) => void;
  removeGeometry: (g: Geometry) => void;
  /**
   * Replace one table's rect (e.g. after a per-table drag). The first
   * argument is the table id from `idOfTable(table)`.
   */
  updateTableRect: (id: string, rect: Rect) => void;
  /**
   * Bulk-merge rects after auto-layout resolves. Updates state in a
   * single pass so descendants re-render once.
   */
  mergeTableRects: (rects: Map<string, Rect>) => void;

  // Pure helpers (do not depend on context state)
  idOfTable: (table: TableMetadata) => string;
  rectOfTable: (table: TableMetadata) => Rect;
  schemaStatus: (schema: SchemaMetadata) => EditStatus;
  tableStatus: (table: TableMetadata) => EditStatus;
  columnStatus: (column: ColumnMetadata) => EditStatus;

  // Imperative actions (fire-and-forget)
  render: () => void;
  layout: () => void;

  // Event bus (per-instance Emittery — see design doc §4 / §8)
  events: Emittery<SchemaDiagramEvents>;
};

export type SchemaDiagramEvents = {
  render: undefined;
  layout: undefined;
  "fit-view": undefined;
  "edit-table": { schema: SchemaMetadata; table: TableMetadata };
  "edit-column": {
    schema: SchemaMetadata;
    table: TableMetadata;
    column: ColumnMetadata;
    target: "name" | "type";
  };
  "set-center": CenterTarget;
};

export type CenterTargetType = "table" | "rect" | "point";
export type CenterTarget<T extends CenterTargetType = CenterTargetType> = {
  type: T;
  target: {
    table: TableMetadata;
    rect: Rect;
    point: Point;
  }[T];
  padding?: number[]; // [T,R,B,L]
  zooms?: number[]; // [min, max]
};
