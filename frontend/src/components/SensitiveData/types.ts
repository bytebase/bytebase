import type { ComposedDatabase, DatabaseResource } from "@/types";
import { type User } from "@/types/proto/v1/user_service";
import type { Group } from "@/types/proto/v1/group_service";
import type { MaskingExceptionPolicy_MaskingException_Action } from "@/types/proto/v1/org_policy_service";

export interface MaskData {
  schema: string;
  table: string;
  column: string;
  semanticTypeId: string;
  classificationId: string;
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
