import type { ComposedDatabase, DatabaseResource } from "@/types";
import { type User } from "@/types/proto/v1/auth_service";
import { MaskingLevel } from "@/types/proto/v1/common";
import type { Group } from "@/types/proto/v1/group";
import type {
  MaskData,
  MaskingExceptionPolicy_MaskingException_Action,
} from "@/types/proto/v1/org_policy_service";

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
  maskingLevel: MaskingLevel;
  expirationTimestamp?: number;
  rawExpression: string;
  databaseResource?: DatabaseResource;
}
