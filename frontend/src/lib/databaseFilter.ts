import type { Engine } from "@/types/proto-es/v1/common_pb";

/**
 * Structured filter consumed by the app store's `fetchDatabases`.
 */
export interface DatabaseFilter {
  project?: string;
  instance?: string;
  environment?: string;
  query?: string;
  showDeleted?: boolean;
  excludeUnassigned?: boolean;
  // label should be "{label key}:{label value}" format
  labels?: string[];
  engines?: Engine[];
  excludeEngines?: Engine[];
  table?: string;
}
