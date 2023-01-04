import { Ref } from "vue";
import type Emittery from "emittery";

import { ColumnMetadata, TableMetadata } from "@/types/proto/store/database";
import { Position, Rect } from "./geometry";
import { EditStatus } from "./edit";

// This is an abstract Schema Diagram context including states, methods and
// events. This should be implemented in the root component of SchemaDiagram
// and used in its descendants.
export type SchemaDiagramContext = {
  // Props
  tableList: Ref<TableMetadata[]>;
  editable: Ref<boolean>;

  // States
  zoom: Ref<number>;
  position: Ref<Position>;
  panning: Ref<boolean>;

  // Methods
  idOfTable: (table: TableMetadata) => string;
  rectOfTable: (table: TableMetadata) => Rect;
  // tell the components to re-calculate positions if needed
  render: () => void;
  // auto-layout all components
  layout: () => void;
  tableStatus: (table: TableMetadata) => EditStatus;
  columnStatus: (column: ColumnMetadata) => EditStatus;

  // Events
  events: Emittery<{
    render: undefined;
    layout: undefined;
    "fit-view": undefined;
    "edit-table": TableMetadata;
    "edit-column": {
      table: TableMetadata;
      column: ColumnMetadata;
      target: "name" | "type";
    };
  }>;
};
