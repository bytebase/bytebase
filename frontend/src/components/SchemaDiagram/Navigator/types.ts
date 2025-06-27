import type { TreeOption } from "naive-ui";
import type {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

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
