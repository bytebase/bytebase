import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";

export type AccessGrantDisplayStatus =
  | "ACTIVE"
  | "PENDING"
  | "EXPIRED"
  | "REVOKED"
  | "UNKNOWN";

export const getAccessGrantExpireTimeMs = (
  grant: AccessGrant
): number | undefined => {
  if (grant.expiration.case === "expireTime") {
    return getTimeForPbTimestampProtoEs(grant.expiration.value);
  }
  return undefined;
};

export const getAccessGrantDisplayStatus = (
  grant: AccessGrant
): AccessGrantDisplayStatus => {
  const expireMs = getAccessGrantExpireTimeMs(grant);
  if (
    grant.status === AccessGrant_Status.ACTIVE &&
    expireMs !== undefined &&
    expireMs < Date.now()
  ) {
    return "EXPIRED";
  }
  switch (grant.status) {
    case AccessGrant_Status.PENDING:
      return "PENDING";
    case AccessGrant_Status.ACTIVE:
      return "ACTIVE";
    case AccessGrant_Status.REVOKED:
      return "REVOKED";
    default:
      return "UNKNOWN";
  }
};

export const getAccessGrantStatusTagType = (
  status: AccessGrantDisplayStatus
): "success" | "warning" | "error" | "default" => {
  switch (status) {
    case "ACTIVE":
      return "success";
    case "PENDING":
      return "warning";
    case "REVOKED":
      return "error";
    default:
      return "default";
  }
};
