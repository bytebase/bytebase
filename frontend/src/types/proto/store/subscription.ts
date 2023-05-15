/* eslint-disable */

export const protobufPackage = "bytebase.store";

export enum PlanType {
  PLAN_TYPE_UNSPECIFIED = 0,
  FREE = 1,
  TEAM = 2,
  ENTERPRISE = 3,
  UNRECOGNIZED = -1,
}

export function planTypeFromJSON(object: any): PlanType {
  switch (object) {
    case 0:
    case "PLAN_TYPE_UNSPECIFIED":
      return PlanType.PLAN_TYPE_UNSPECIFIED;
    case 1:
    case "FREE":
      return PlanType.FREE;
    case 2:
    case "TEAM":
      return PlanType.TEAM;
    case 3:
    case "ENTERPRISE":
      return PlanType.ENTERPRISE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanType.UNRECOGNIZED;
  }
}

export function planTypeToJSON(object: PlanType): string {
  switch (object) {
    case PlanType.PLAN_TYPE_UNSPECIFIED:
      return "PLAN_TYPE_UNSPECIFIED";
    case PlanType.FREE:
      return "FREE";
    case PlanType.TEAM:
      return "TEAM";
    case PlanType.ENTERPRISE:
      return "ENTERPRISE";
    case PlanType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
