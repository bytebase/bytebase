import type { Path, Point, Rect, Size } from "@/components/SchemaDiagram/types";

export type GraphChildNodeItem = { id: string; size: Size; pos: Point };

export type GraphNodeItem = {
  group: string;
  id: string;
  size: Size;
  children: GraphChildNodeItem[]; // not used yet
};

export type GraphEdgeItem = {
  id: string;
  from: string;
  to: string;
};

export type Layout = {
  rects: Map<string, Rect>;
  paths: Map<string, Path>;
};
