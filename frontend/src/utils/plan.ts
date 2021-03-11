export type FeatureType =
  // - Task can only be assigned to DBA and Owner
  "bytebase.admin";

export enum PlanType {
  FREE = 0,
  STANDARD = 1,
  PREMIUM = 2,
}

// A map from the a particular feature to the respective enablement of a particular plan
const FEATURE_MATRIX: Map<FeatureType, boolean[]> = new Map([
  ["bytebase.admin", [false, true, true]],
]);

// Feature gate. Returns true if the particular "type" feature is supported
export function feature(type: FeatureType): boolean {
  const currentPlanType = PlanType.STANDARD;
  // It must be a typo or we forget adding the entry to the FEATURE_MATRIX,
  // so let's just crash it.
  return FEATURE_MATRIX.get(type)![currentPlanType];
}
