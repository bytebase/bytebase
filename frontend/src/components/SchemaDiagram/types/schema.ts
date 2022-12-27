import {
  ForeignKeyMetadata,
  TableMetadata,
} from "@/types/proto/store/database";

export type ForeignKey = {
  from: {
    table: TableMetadata;
    column: string;
  };
  to: {
    table: TableMetadata;
    column: string;
  };
  metadata: ForeignKeyMetadata;
};
