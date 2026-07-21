import type {
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

export type ForeignKey = {
  from: {
    schema: SchemaMetadata;
    table: TableMetadata;
    column: string;
  };
  to: {
    schema: SchemaMetadata;
    table: TableMetadata;
    column: string;
  };
  metadata: ForeignKeyMetadata;
};
