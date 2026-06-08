import dayjs from "dayjs";
import i18n from "@/react/i18n";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { formatAbsoluteDateTime } from "@/utils/datetime";

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

// Days/hours/minutes formatter for grant durations. Extracted so the
// same shape ("4h", "2d3h", "30m") applies to both the input TTL (on
// pending grants) and the recovered original duration on active grants
// (`expireTime - createTime`, see `getAccessGrantExpirationText`).
const formatGrantDurationFromSeconds = (totalSeconds: number): string => {
  const dur = dayjs.duration(totalSeconds, "seconds");
  const days = Math.floor(dur.asDays());
  const hours = dur.hours();
  const minutes = dur.minutes();
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  return parts.join("") || "<1m";
};

export const getAccessGrantExpirationText = (
  grant: AccessGrant
):
  | { type: "datetime"; value: string; datetime: string; duration?: string }
  | { type: "duration"; value: string; duration: string }
  | { type: "never" } => {
  if (grant.expiration.case === "expireTime") {
    const expireMs = getTimeForPbTimestampProtoEs(grant.expiration.value);
    const datetime = formatAbsoluteDateTime(expireMs);
    // Recover the originally-configured duration from
    // (expireTime - createTime). This is exact when the grant was
    // activated immediately on creation (auto-approval / no-approval
    // flow); for slow approvals it drifts upward by the approval
    // latency. We expose it as a hint, not a contract — UIs should
    // treat it as best-effort and fall back to just `datetime` when
    // `createTime` is missing.
    let duration: string | undefined;
    if (grant.createTime) {
      const createMs = getTimeForPbTimestampProtoEs(grant.createTime);
      if (createMs > 0 && expireMs > createMs) {
        duration = formatGrantDurationFromSeconds(
          Math.floor((expireMs - createMs) / 1000)
        );
      }
    }
    // `value` retained for backward-compat callers that only need the
    // absolute datetime (e.g. `AccessGrantItem`, the project access
    // grants table).
    return { type: "datetime", value: datetime, datetime, duration };
  }
  if (grant.expiration.case === "ttl") {
    const totalSeconds = Number(grant.expiration.value.seconds);
    const duration = formatGrantDurationFromSeconds(totalSeconds);
    return { type: "duration", value: duration, duration };
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
        if (issue.approvalStatus === ApprovalStatus.REJECTED) {
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
      return i18n.t("common.active");
    case "PENDING":
      return i18n.t("common.pending");
    case "EXPIRED":
      return i18n.t("sql-editor.expired");
    case "REVOKED":
      return i18n.t("common.revoked");
    case "REJECTED":
      return i18n.t("common.rejected");
    case "CANCELED":
      return i18n.t("common.canceled");
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
