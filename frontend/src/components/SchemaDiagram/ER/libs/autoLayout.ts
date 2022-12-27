import { Path, Rect, Size } from "../../types";
import dagre from "dagre";

export type GraphNodeItem = {
  id: string;
  size: Size;
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

export const layoutDagre = async (
  nodeList: GraphNodeItem[],
  edgeList: GraphEdgeItem[]
): Promise<Layout> => {
  const g = new dagre.graphlib.Graph({ multigraph: true });
  g.setGraph({
    rankdir: "LR",
    nodesep: 160, // gap between nodes in a layer
    ranksep: 160, // gap between layers
  });
  g.setDefaultEdgeLabel(() => ({}));
  nodeList.forEach(({ id, size }) => {
    g.setNode(id, {
      ...size,
    });
  });
  edgeList.forEach(({ id, from, to }) => {
    g.setEdge({ name: id, v: from, w: to });
  });
  dagre.layout(g);
  const rects = new Map<string, Rect>();
  g.nodes().forEach((id) => {
    const node = g.node(id);
    rects.set(id, node);
  });
  const paths = new Map<string, Path>();
  g.edges().forEach((e) => {
    const edge = g.edge(e);
    const { points } = edge;
    paths.set(e.name!, points);
  });

  return { rects, paths };
};

export const autoLayout = async (
  nodeList: GraphNodeItem[],
  edgeList: GraphEdgeItem[]
): Promise<Layout> => {
  return layoutDagre(nodeList, edgeList);
};
