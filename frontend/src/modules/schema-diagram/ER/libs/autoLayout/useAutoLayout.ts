import { useCallback, useRef } from "react";
import type {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { Size } from "../../../types";
import { autoLayout } from "./index";
import type { GraphEdgeItem, GraphNodeItem } from "./types";

export type ForeignKeyEdge = {
  fromSchema: string;
  fromTable: TableMetadata;
  fromColumn: string;
  toSchema: string;
  toTable: TableMetadata;
  toColumn: string;
};

interface UseAutoLayoutInput {
  selectedSchemas: SchemaMetadata[];
  edges: ForeignKeyEdge[];
  /** Resolves a TableMetadata to its assigned id (from the SchemaDiagram context). */
  idOfTable: (table: TableMetadata) => string;
  /** Reads the rendered DOM size of a table card by id. */
  sizeOfTable: (id: string) => Size | null;
}

/**
 * Drive ELK auto-layout for the diagram's currently-selected schemas.
 *
 * Returns a `runLayout()` callback that the SchemaDiagram root invokes
 * (typically wired to the `events.on("layout", ...)` channel). The
 * callback gathers each table's measured DOM size, builds the node + edge
 * graph, awaits ELK, and resolves with `Map<tableId, Rect>` so the caller
 * can `mergeTableRects(...)` the result back into the context.
 *
 * The most-recent `latest()` token guards against stale ELK results when
 * the user rapidly toggles schemas: only the in-flight call from the
 * latest invocation resolves with data; earlier ones resolve `null`.
 */
export function useAutoLayout({
  selectedSchemas,
  edges,
  idOfTable,
  sizeOfTable,
}: UseAutoLayoutInput) {
  const tokenRef = useRef(0);

  return useCallback(async () => {
    const myToken = ++tokenRef.current;

    const nodeList: GraphNodeItem[] = [];
    for (const schema of selectedSchemas) {
      for (const table of schema.tables) {
        const id = idOfTable(table);
        const size = sizeOfTable(id);
        if (!size) continue;
        nodeList.push({
          group: `schema-${schema.name}`,
          id,
          size,
          children: [],
        });
      }
    }

    const edgeList: GraphEdgeItem[] = edges.map((fk) => {
      const fromTableId = idOfTable(fk.fromTable);
      const toTableId = idOfTable(fk.toTable);
      return {
        id: `${fk.fromSchema}.${fromTableId}.${fk.fromColumn}->${fk.toSchema}.${toTableId}.${fk.toColumn}`,
        from: fromTableId,
        to: toTableId,
      };
    });

    const { rects } = await autoLayout(nodeList, edgeList);
    if (myToken !== tokenRef.current) {
      // A newer call superseded this one — discard.
      return null;
    }
    return rects;
  }, [selectedSchemas, edges, idOfTable, sizeOfTable]);
}
