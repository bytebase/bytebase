import { Ref } from "vue";
import type Emittery from "emittery";
import { TableMetadata } from "@/types/proto/store/database";

import { Position, Rect } from "./geometry";

// This is an abstract Schema Diagram context including states, methods and
// events. This should be implemented in the root component of SchemaDiagram
// and used in its descendants.
export type SchemaDiagramContext = {
  // Props
  tableList: Ref<TableMetadata[]>;

  // States
  zoom: Ref<number>;
  position: Ref<Position>;

  // Methods
  idOfTable: (table: TableMetadata) => string;
  rectOfTable: (table: TableMetadata) => Rect;
  // tell the components to re-calculate positions if needed
  render: () => void;
  // auto-layout all components
  layout: () => void;

  // Events
  events: Emittery<{
    render: undefined;
    layout: undefined;
    "fit-view": undefined;
  }>;
};
