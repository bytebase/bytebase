import type { ComposedDatabase, DatabaseResource } from "@/types";
import type {
  TableCatalog,
  ColumnCatalog,
  ObjectSchema,
} from "@/types/proto/api/v1alpha/database_catalog_service";
import type { Group } from "@/types/proto/api/v1alpha/group_service";
import type { MaskingExceptionPolicy_MaskingException_Action } from "@/types/proto/api/v1alpha/org_policy_service";
import { type User } from "@/types/proto/api/v1alpha/user_service";

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
  databaseResource?: DatabaseResource;
}
