import { ComposedDatabase } from "@/types";
import { MaskData } from "@/types/proto/v1/org_policy_service";

export interface SensitiveColumn {
  database: ComposedDatabase;
  maskData: MaskData;
}
