import ELK, { type ElkNode } from "elkjs/lib/elk.bundled";

import type { Path, Rect } from "@/components/SchemaDiagram/types";
import type { GraphEdgeItem, GraphNodeItem, Layout } from "../types";

export const layoutELK = async (
  nodeList: GraphNodeItem[],
  edgeList: GraphEdgeItem[]
): Promise<Layout> => {
  const elk = new ELK({});
  const graph: ElkNode = {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "100",
      "elk.layered.spacing.nodeNodeBetweenLayers": "200",
      "elk.layered.nodePlacement.favorStraightEdges": "true",
      "elk.edgeRouting": "POLYLINE",
    },
    children: nodeList.map(({ id, size }) => ({
      id,
      ...size,
    })),
    edges: edgeList.map(({ id, from, to }) => ({
      id,
      sources: [from],
      targets: [to],
    })),
  };

  const result = await elk.layout(graph, {
    logging: true,
  });
  const rects = new Map<string, Rect>();
  result.children?.forEach((node) => {
    rects.set(node.id, {
      x: node.x!,
      y: node.y!,
      width: node.width!,
      height: node.height!,
    });
  });
  const paths = new Map<string, Path>();
  result.edges?.forEach((edge) => {
    const { sections = [] } = edge;
    const path: Path = sections.flatMap((section) => {
      return [
        section.startPoint,
        ...(section.bendPoints ?? []),
        section.endPoint,
      ];
    });
    paths.set(edge.id, path);
  });

  return { rects, paths };
};
