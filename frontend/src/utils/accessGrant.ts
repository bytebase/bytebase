import dayjs from "dayjs";
import { h } from "vue";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import YouTag from "@/components/misc/YouTag.vue";
import { RichDatabaseName } from "@/components/v2";
import { t } from "@/plugins/i18n";
import { useCurrentUserV1, useDatabaseV1Store, useUserStore } from "@/store";
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
import { formatAbsoluteDateTime } from "@/utils/datetime";
import { getDefaultPagination } from "@/utils/pagination";
import { extractDatabaseResourceName } from "@/utils/v1/database";

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
    return { type: "datetime", value: formatAbsoluteDateTime(ms) };
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

export const getAccessSearchOptions = ({
  project,
  showCreator,
}: {
  project: string;
  showCreator: boolean;
}): ScopeOption[] => {
  const databaseStore = useDatabaseV1Store();
  const userStore = useUserStore();
  const me = useCurrentUserV1();

  const options: ScopeOption[] = [
    {
      id: "status",
      title: t("common.status"),
      allowMultiple: true,
      options: [
        {
          value: AccessGrant_Status[AccessGrant_Status.ACTIVE],
          keywords: ["active"],
          render: () => t("common.active"),
        },
        {
          value: AccessGrant_Status[AccessGrant_Status.PENDING],
          keywords: ["pending"],
          render: () => t("common.pending"),
        },
        {
          value: "EXPIRED",
          keywords: ["expired"],
          render: () => t("sql-editor.expired"),
        },
        {
          value: AccessGrant_Status[AccessGrant_Status.REVOKED],
          keywords: ["revoked"],
          render: () => t("common.revoked"),
        },
      ],
    },
    {
      id: "database",
      title: t("common.database"),
      search: ({ keyword, nextPageToken: pageToken }) =>
        databaseStore
          .fetchDatabases({
            parent: project,
            pageToken: pageToken,
            pageSize: getDefaultPagination(),
            filter: { query: keyword },
          })
          .then((resp) => ({
            nextPageToken: resp.nextPageToken,
            options: resp.databases.map<ValueOption>((db) => {
              const { database: dbName } = extractDatabaseResourceName(db.name);
              return {
                value: db.name,
                keywords: [dbName, db.name],
                custom: true,
                render: () =>
                  h(RichDatabaseName, {
                    database: db,
                    showInstance: true,
                    showEngineIcon: true,
                  }),
              };
            }),
          })),
    },
  ];

  if (showCreator) {
    options.push({
      id: "creator",
      title: t("common.creator"),
      search: ({ keyword, nextPageToken: pageToken }) =>
        userStore
          .fetchUserList({
            pageToken,
            pageSize: getDefaultPagination(),
            filter: { query: keyword },
          })
          .then((resp) => ({
            nextPageToken: resp.nextPageToken,
            options: resp.users.map<ValueOption>((user) => ({
              value: user.email,
              keywords: [user.email, user.title],
              render: () => {
                const children = [
                  h(BBAvatar, { size: "TINY", username: user.title }),
                  h("span", user.title),
                ];
                if (user.name === me.value.name) {
                  children.push(h(YouTag));
                }
                return h(
                  "div",
                  { class: "flex items-center gap-x-1" },
                  children
                );
              },
            })),
          })),
    });
  }

  return options;
};
