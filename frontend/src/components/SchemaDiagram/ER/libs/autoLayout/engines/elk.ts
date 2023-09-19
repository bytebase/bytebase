import ELK, { LayoutOptions, type ElkNode } from "elkjs/lib/elk.bundled";
import { groupBy } from "lodash-es";
import type { Path, Rect } from "@/components/SchemaDiagram/types";
import type { GraphEdgeItem, GraphNodeItem, Layout } from "../types";

export const layoutELK = async (
  nodeList: GraphNodeItem[],
  edgeList: GraphEdgeItem[]
): Promise<Layout> => {
  const elk = new ELK({});
  const nodeGroups = groupBy(nodeList, (node) => node.group);
  const layoutOptions: LayoutOptions = {
    "elk.algorithm": "layered",
    "elk.direction": "RIGHT",
    "spacing.nodeNode": "100",
    "spacing.nodeNodeBetweenLayers": "200",
    "nodePlacement.favorStraightEdges": "true",
    "elk.edgeRouting": "POLYLINE",
  };
  const graph: ElkNode = {
    id: "root",
    layoutOptions,
    children: Object.keys(nodeGroups).map((group) => {
      return {
        id: `group-${group}`,
        layoutOptions,
        children: nodeGroups[group].map(({ id, size }) => ({
          id,
          ...size,
        })),
      };
    }),
    edges: edgeList.map(({ id, from, to }) => ({
      id,
      sources: [from],
      targets: [to],
    })),
  };

  const result = await elk.layout(graph, {
    // logging: true,
  });
  const rects = new Map<string, Rect>();
  result.children?.forEach((cluster) => {
    const { x = 0, y = 0 } = cluster;
    cluster.children?.forEach((node) => {
      rects.set(node.id, {
        x: node.x! + x,
        y: node.y! + y,
        width: node.width!,
        height: node.height!,
      });
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
