import type { ComposedDatabase, DatabaseResource } from "@/types";
import type {
  TableCatalog,
  ColumnCatalog,
  ObjectSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { MaskingExceptionPolicy_MaskingException_Action } from "@/types/proto/v1/org_policy_service";
import { type User } from "@/types/proto/v1/user_service";

export interface MaskData {
  schema: string;
  table: string;
  column: string;
  disableSemanticType?: boolean;
  semanticTypeId: string;
  disableClassification?: boolean;
  classificationId: string;
  target: TableCatalog | ColumnCatalog | ObjectSchema;
}

export interface SensitiveColumn {
  database: ComposedDatabase;
  maskData: MaskData;
}

export interface AccessUser {
  type: "user" | "group";
  key: string;
  group?: Group;
  user?: User;
  supportActions: Set<MaskingExceptionPolicy_MaskingException_Action>;
  expirationTimestamp?: number;
  rawExpression: string;
  description: string;
  databaseResource?: DatabaseResource;
}
