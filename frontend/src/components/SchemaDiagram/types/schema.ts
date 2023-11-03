import {
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

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
