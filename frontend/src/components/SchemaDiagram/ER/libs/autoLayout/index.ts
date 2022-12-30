import { GraphEdgeItem, GraphNodeItem, Layout } from "./types";

export * from "./types";

export const autoLayout = async (
  nodeList: GraphNodeItem[],
  edgeList: GraphEdgeItem[]
): Promise<Layout> => {
  const { layoutELK } = await import("./engines/elk");
  return layoutELK(nodeList, edgeList);
};
