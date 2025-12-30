import type { ComposedDatabase, DatabaseResource } from "@/types";
import type {
  ColumnCatalog,
  ObjectSchema,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";

export type MaskDataTarget = TableCatalog | ColumnCatalog | ObjectSchema;

export interface MaskData {
  schema: string;
  table: string;
  column: string;
  disableSemanticType?: boolean;
  semanticTypeId: string;
  disableClassification?: boolean;
  classificationId: string;
  target: MaskDataTarget;
}

export interface SensitiveColumn {
  database: ComposedDatabase;
  maskData: MaskData;
}

export interface AccessUser {
  type: "user" | "group";
  key: string;
  member: string;
  expirationTimestamp?: number;
  rawExpression: string;
  description: string;
  databaseResource?: DatabaseResource;
}
