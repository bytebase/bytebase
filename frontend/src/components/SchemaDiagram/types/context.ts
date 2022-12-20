import { Ref } from "vue";
import type Emittery from "emittery";
import { Position, Rect } from "./geometry";
import { Table } from "@/types/schemaEditor/atomType";

// This is an abstract Schema Diagram context including states, methods and
// events. This should be implemented in the root component of SchemaDiagram
// and used in its descendants.
export type SchemaDiagramContext = {
  // Props
  tableList: Ref<Table[]>;

  // States
  zoom: Ref<number>;
  position: Ref<Position>;

  // Methods
  idOfTable: (table: Table) => string;
  rectOfTable: (table: Table) => Rect;
  // tell the components to re-calculate positions if needed
  render: () => void;

  // Events
  events: Emittery<{
    render: undefined;
  }>;
};
