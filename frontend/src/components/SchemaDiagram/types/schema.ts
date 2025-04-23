import type {
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/api/v1alpha/database_service";

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
