import dagre from "dagre";
import { Rect, Size } from "../../types";

export type GraphNodeItem = {
  id: string;
  size: Size;
};

export type GraphEdgeItem = {
  from: string;
  to: string;
};

export const autoLayout = (
  nodeList: GraphNodeItem[],
  edgeList: GraphEdgeItem[]
) => {
  const g = new dagre.graphlib.Graph();
  g.setGraph({
    rankdir: "TD",
    nodesep: 100, // x gap
    ranksep: 100, // y gap
  });
  g.setDefaultEdgeLabel(() => ({}));

  nodeList.forEach(({ id, size }) => {
    g.setNode(id, {
      ...size,
    });
  });
  edgeList.forEach(({ from, to }) => {
    g.setEdge({ v: from, w: to });
  });
  dagre.layout(g);

  const rectsByTableId = new Map<string, Rect>();
  g.nodes().forEach((id) => {
    const node = g.node(id);
    rectsByTableId.set(id, node);
  });
  return rectsByTableId;
};
