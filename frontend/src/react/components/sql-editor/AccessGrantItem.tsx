import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  getAccessGrantDisplayStatus,
  getAccessGrantDisplayStatusText,
  getAccessGrantExpirationText,
  getAccessGrantExpireTimeMs,
  getAccessGrantStatusTagType,
} from "@/utils/accessGrant";

// Map naive-ui tag type to shadcn Badge variant.
function mapTagTypeToBadgeVariant(
  tagType: "success" | "warning" | "error" | "default"
): "default" | "secondary" | "destructive" | "warning" | "success" {
  if (tagType === "success") return "success";
  if (tagType === "warning") return "warning";
  if (tagType === "error") return "destructive";
  return "default";
}

type Props = {
  readonly grant: AccessGrant;
  readonly highlight?: boolean;
  readonly issue?: Issue;
  readonly onRun: (grant: AccessGrant) => void;
  readonly onRequest: (grant: AccessGrant) => void;
};

export function AccessGrantItem({
  grant,
  highlight = false,
  issue,
  onRun,
  onRequest,
}: Props) {
  const { t } = useTranslation();

  const displayStatus = getAccessGrantDisplayStatus(grant, issue);
  const isActive = displayStatus === "ACTIVE";
  const isExpired = displayStatus === "EXPIRED";
  const isRejectedOrCanceled =
    displayStatus !== "ACTIVE" && displayStatus !== "PENDING";
  const statusLabel = getAccessGrantDisplayStatusText(grant, issue);
  const expireTimeMs = getAccessGrantExpireTimeMs(grant);

  const statusTagType = getAccessGrantStatusTagType(displayStatus);
  const badgeVariant = mapTagTypeToBadgeVariant(statusTagType);

  const expirationText = (() => {
    if (displayStatus !== "ACTIVE" && displayStatus !== "EXPIRED") return;
    const info = getAccessGrantExpirationText(grant);
    if (info.type === "never" || info.type === "duration") return;

    if (!isExpired && expireTimeMs !== undefined) {
      const diff = expireTimeMs - Date.now();
      const hours = Math.floor(diff / (1000 * 60 * 60));
      if (hours >= 24) {
        return t("sql-editor.expire-at", { time: info.value });
      }
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const dur = hours > 0 ? `${hours}h${minutes}m` : `${minutes}m`;
      return t("sql-editor.expire-in", { time: dur });
    }
    return `${t("issue.access-grant.expired-at")} ${info.value}`;
  })();

  const allDatabaseNames = grant.targets.map((tgt) => {
    const match = tgt.match(/databases\/(.+)$/);
    return match ? match[1] : tgt;
  });
  const databaseNamesDisplay =
    allDatabaseNames.length <= 2
      ? allDatabaseNames.join(", ")
      : `${allDatabaseNames.slice(0, 2).join(", ")} ${t("sql-editor.and-n-more-databases", { n: allDatabaseNames.length - 2 })}`;

  const issueLink = grant.issue
    ? grant.issue.startsWith("/")
      ? grant.issue
      : `/${grant.issue}`
    : "";

  return (
    <div
      className={cn(
        "w-full p-2 gap-y-2 border-b flex flex-col justify-start items-start hover:bg-control-bg",
        highlight
          ? "bb-access-grant-highlight"
          : "transition-colors duration-1000"
      )}
    >
      {/*
       * `flex-wrap` + `justify-between` keeps the original "badges left,
       * expiration right" layout when both fit on the row, but lets the
       * row wrap when the panel is narrow. Pairing it with `shrink-0`
       * on the badges container preserves each pill at its natural size
       * so the label never wraps inside the pill (`脱敏豁免` → "脱敏豁
       * \n免"). When the expiration wraps to a second row it falls back
       * to the row's start alignment (justify-between has no effect on
       * a single-item row), so it reads naturally left-to-right under
       * the badges instead of being stranded on the right.
       */}
      <div className="w-full flex flex-wrap items-center justify-between gap-x-2 gap-y-1">
        <div className="flex items-center gap-x-1 shrink-0">
          <Badge
            variant={badgeVariant}
            className="text-[10px] px-1.5 py-0 rounded-full"
          >
            {statusLabel}
          </Badge>
          {grant.unmask && (
            <Badge
              variant="default"
              className="text-[10px] px-1.5 py-0 rounded-full"
            >
              {t("sql-editor.grant-type-unmask")}
            </Badge>
          )}
        </div>
        {expirationText && (
          <span className="text-xs text-control-placeholder shrink-0">
            {expirationText}
          </span>
        )}
      </div>

      <Tooltip
        content={
          <pre className="max-w-lg whitespace-pre-wrap text-xs">
            {grant.query}
          </pre>
        }
        side="right"
      >
        <p
          className={cn(
            "max-w-full text-xs wrap-break-word whitespace-pre-wrap font-mono line-clamp-2",
            (isExpired || isRejectedOrCanceled) &&
              "line-through text-control-placeholder"
          )}
        >
          {grant.query}
        </p>
      </Tooltip>

      <div className="w-full flex flex-col gap-y-2">
        {allDatabaseNames.length <= 2 ? (
          <span className="text-xs text-control-placeholder truncate">
            {databaseNamesDisplay}
          </span>
        ) : (
          <Tooltip
            content={
              <div className="flex flex-col">
                {allDatabaseNames.map((n) => (
                  <span key={n}>{n}</span>
                ))}
              </div>
            }
            side="right"
          >
            <span className="text-xs text-control-placeholder truncate">
              {databaseNamesDisplay}
            </span>
          </Tooltip>
        )}

        <div className="flex items-center justify-between gap-x-1">
          <div>
            {isActive && (
              <Button
                size="sm"
                variant="default"
                className="h-6 text-xs"
                data-run-btn
                onClick={(e) => {
                  e.stopPropagation();
                  onRun(grant);
                }}
              >
                {t("common.run")}
              </Button>
            )}
          </div>
          <div className="flex items-center gap-x-1">
            {isRejectedOrCanceled && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 text-xs"
                data-re-request-btn
                onClick={(e) => {
                  e.stopPropagation();
                  onRequest(grant);
                }}
              >
                {t("sql-editor.re-request")}
              </Button>
            )}
            {grant.issue && (
              <a
                href={issueLink}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="inline-flex items-center justify-center h-6 text-xs px-2 rounded-xs hover:bg-control-bg text-control"
              >
                {t("sql-editor.view-issue")}
              </a>
            )}
          </div>
        </div>
      </div>

      {/* Highlight pulse animation styles */}
      <style>{`
        .bb-access-grant-highlight {
          animation: bb-access-grant-highlight-fade 3s ease-in-out;
        }
        @keyframes bb-access-grant-highlight-fade {
          0% { background-color: rgb(219 234 254); }
          60% { background-color: rgb(219 234 254); }
          100% { background-color: transparent; }
        }
      `}</style>
    </div>
  );
}
