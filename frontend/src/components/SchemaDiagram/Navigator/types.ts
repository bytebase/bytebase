import { SchemaMetadata, TableMetadata } from "@/types/proto/store/database";
import type { TreeOption } from "naive-ui";

export type NodeType = "schema" | "table";

type NodeTypeMapping = {
  schema: SchemaMetadata;
  table: TableMetadata;
};

export type TreeNode<T extends NodeType = NodeType> = TreeOption & {
  key: string;
  type: T;
  data: NodeTypeMapping[T];
  isLeaf: boolean;
  children: TreeNode<NodeType>[];
};
