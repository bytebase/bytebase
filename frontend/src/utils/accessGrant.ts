import dayjs from "dayjs";
import { t } from "@/plugins/i18n";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";

export type AccessGrantFilterStatus =
  | "ACTIVE"
  | "PENDING"
  | "REVOKED"
  | "EXPIRED";

export type AccessGrantDisplayStatus =
  | AccessGrantFilterStatus
  | "REJECTED"
  | "CANCELED"
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
  grant: AccessGrant,
  issue?: Issue
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
      if (issue) {
        if (issue.approvalStatus === Issue_ApprovalStatus.REJECTED) {
          return "REJECTED";
        }
        if (issue.status === IssueStatus.CANCELED) {
          return "CANCELED";
        }
      }
      return "PENDING";
    case AccessGrant_Status.ACTIVE:
      return "ACTIVE";
    case AccessGrant_Status.REVOKED:
      return "REVOKED";
    default:
      return "UNKNOWN";
  }
};

export const getAccessGrantDisplayStatusText = (
  grant: AccessGrant,
  issue?: Issue
) => {
  const displayStatus = getAccessGrantDisplayStatus(grant, issue);
  switch (displayStatus) {
    case "ACTIVE":
      return t("common.active");
    case "PENDING":
      return t("common.pending");
    case "EXPIRED":
      return t("sql-editor.expired");
    case "REVOKED":
      return t("common.revoked");
    case "REJECTED":
      return t("common.rejected");
    case "CANCELED":
      return t("common.canceled");
    default:
      return displayStatus;
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
    case "REJECTED":
      return "error";
    case "CANCELED":
      return "default";
    default:
      return "default";
  }
};
