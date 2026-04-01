import type { DatabaseResource } from "@/types";
import type {
  ColumnCatalog,
  ObjectSchema,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

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
  database: Database;
  maskData: MaskData;
}

export interface ClassificationLevel {
  operator: string;
  value: number;
}

export interface AccessUser {
  type: "user" | "group";
  key: string;
  member: string;
  expirationTimestamp?: number;
  rawExpression: string;
  description: string;
  databaseResources?: DatabaseResource[];
  // The condition portion of the CEL expression, excluding request.time.
  // Used as fallback display when databaseResources can't be parsed.
  conditionExpression: string;
}

export interface ExemptionGrant {
  id: string;
  description: string;
  expirationTimestamp?: number;
  rawExpression: string;
  databaseResources?: DatabaseResource[];
  conditionExpression: string;
  classificationLevel?: ClassificationLevel;
  // TODO(BYT-9098): Add grantedBy (creator) and grantedAt (creation timestamp)
  // for auditability. Requires proto change: MaskingExemptionPolicy_Exemption
  // currently only stores members + condition. The DB policy table only has
  // updated_at, no created_at or creator. Needs schema migration + proto update.
}

export interface ExemptionMember {
  type: "user" | "group";
  member: string;
  grants: ExemptionGrant[];
  totalResources: number;
  databaseNames: string[];
  neverExpires: boolean;
  nearestExpiration?: number;
}
