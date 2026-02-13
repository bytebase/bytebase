import dayjs from "dayjs";
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
  if (grant.expiration.case === "ttl") {
    return Date.now() + Number(grant.expiration.value.seconds) * 1000;
  }
  return undefined;
};

export const getAccessGrantExpirationText = (
  grant: AccessGrant
):
  | { type: "datetime"; value: string }
  | { type: "duration"; value: string }
  | { type: "never" } => {
  if (grant.expiration.case === "expireTime") {
    const ms = getTimeForPbTimestampProtoEs(grant.expiration.value);
    return { type: "datetime", value: dayjs(ms).format("LLL") };
  }
  if (grant.expiration.case === "ttl") {
    const totalSeconds = Number(grant.expiration.value.seconds);
    const dur = dayjs.duration(totalSeconds, "seconds");
    const days = Math.floor(dur.asDays());
    const hours = dur.hours();
    const minutes = dur.minutes();
    const parts: string[] = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0) parts.push(`${minutes}m`);
    return { type: "duration", value: parts.join("") || "<1m" };
  }
  return { type: "never" };
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
