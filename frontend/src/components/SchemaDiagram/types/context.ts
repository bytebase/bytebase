import type { Ref } from "vue";
import type Emittery from "emittery";

import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
  ColumnMetadata,
} from "@/types/proto/store/database";
import type { Database } from "@/types";
import type { Geometry, Point, Rect } from "./geometry";
import { EditStatus } from "./edit";

// This is an abstract Schema Diagram context including states, methods and
// events. This should be implemented in the root component of SchemaDiagram
// and used in its descendants.
export type SchemaDiagramContext = {
  // Props
  database: Ref<Database>;
  databaseMetadata: Ref<DatabaseMetadata>;
  editable: Ref<boolean>;

  // States
  dummy: Ref<boolean>;
  busy: Ref<boolean>;
  zoom: Ref<number>;
  position: Ref<Point>;
  panning: Ref<boolean>;
  geometries: Ref<Set<Geometry>>;

  // Methods
  idOfTable: (table: TableMetadata) => string;
  rectOfTable: (table: TableMetadata) => Rect;
  // tell the components to re-calculate positions if needed
  render: () => void;
  // auto-layout all components
  layout: () => void;
  schemaStatus: (schema: SchemaMetadata) => EditStatus;
  tableStatus: (table: TableMetadata) => EditStatus;
  columnStatus: (column: ColumnMetadata) => EditStatus;

  // Events
  events: Emittery<{
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
  }>;
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
};
