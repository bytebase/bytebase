import {
  ForeignKeyMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";

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
