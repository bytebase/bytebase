import type {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

export type NodeType = "schema" | "table";

type NodeTypeMapping = {
  schema: SchemaMetadata;
  table: TableMetadata;
};

export type NavigatorTreeNode<T extends NodeType = NodeType> = {
  id: string;
  type: T;
  data: NodeTypeMapping[T];
  children?: NavigatorTreeNode<NodeType>[];
};
