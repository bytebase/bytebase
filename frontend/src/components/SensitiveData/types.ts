import type { ComposedDatabase } from "@/types";
import type { MaskData } from "@/types/proto/v1/org_policy_service";

export interface SensitiveColumn {
  database: ComposedDatabase;
  maskData: MaskData;
}
