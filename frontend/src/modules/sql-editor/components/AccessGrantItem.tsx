import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { RouterLink } from "@/components/RouterLink";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { useTimeReading } from "@/hooks/useTimeReading";
import { cn } from "@/lib/utils";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  accessGrantStatusReading,
  getAccessGrantDisplayStatusText,
  getAccessGrantStatusTagType,
  getActiveAccessGrantDeadlineMs,
} from "@/utils/accessGrant";
import { countdownReading, formatAbsoluteDateTime } from "@/utils/datetime";

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

  // Two readings, each on its own boundary. The countdown's last step lands on
  // the deadline, the same instant the status turns expired, so the status
  // subscription adds no wake today -- it is here so the badge does not depend
  // on that coincidence holding.
  const deadlineMs = getActiveAccessGrantDeadlineMs(grant);
  const countdown = useTimeReading(countdownReading, deadlineMs);
  const displayStatus = useTimeReading(accessGrantStatusReading, {
    grant,
    issue,
  });
  const isActive = displayStatus === "ACTIVE";
  const isExpired = displayStatus === "EXPIRED";
  const isRejectedOrCanceled =
    displayStatus !== "ACTIVE" && displayStatus !== "PENDING";
  const statusLabel = getAccessGrantDisplayStatusText(displayStatus);

  const statusTagType = getAccessGrantStatusTagType(displayStatus);
  const badgeVariant = mapTagTypeToBadgeVariant(statusTagType);

  // The countdown is the one form that hides the deadline, so it alone carries
  // it in a tooltip; the other two are interpolated sentences and state the
  // deadline in full themselves.
  const expiration = (() => {
    if (deadlineMs === undefined || countdown === undefined) return undefined;
    const deadline = formatAbsoluteDateTime(deadlineMs);
    switch (countdown.kind) {
      case "passed":
        return { text: `${t("issue.access-grant.expired-at")} ${deadline}` };
      case "beyondDay":
        return { text: t("sql-editor.expire-at", { time: deadline }) };
      case "within": {
        const { hours, minutes } = countdown;
        const left = hours > 0 ? `${hours}h${minutes}m` : `${minutes}m`;
        return {
          text: t("sql-editor.expire-in", { time: left }),
          hiddenDeadline: deadline,
        };
      }
    }
  })();

  const visibleTargets = grant.targets.slice(0, 2);
  const remainingTargetCount = grant.targets.length - visibleTargets.length;
  const databaseTargets = (
    <div className="flex w-full min-w-0 flex-col gap-y-1">
      {visibleTargets.map((target, index) => (
        <DatabaseTargetDisplay
          key={`${target}-${index}`}
          target={target}
          showEnvironment
        />
      ))}
      {remainingTargetCount > 0 && (
        <span className="text-xs text-control-placeholder">
          {t("sql-editor.and-n-more-databases", {
            n: remainingTargetCount,
          })}
        </span>
      )}
    </div>
  );

  const issueLink = grant.issue
    ? grant.issue.startsWith("/")
      ? grant.issue
      : `/${grant.issue}`
    : "";

  return (
    <div
      className={cn(
        "w-full min-w-0 p-2 gap-y-2 border-b flex flex-col justify-start items-start hover:bg-control-bg",
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
            className="text-xs px-1.5 py-0 rounded-full"
          >
            {statusLabel}
          </Badge>
          {grant.unmask && (
            <Badge
              variant="default"
              className="text-xs px-1.5 py-0 rounded-full"
            >
              {t("sql-editor.grant-type-unmask")}
            </Badge>
          )}
          {grant.export && (
            <Badge
              variant="default"
              className="text-xs px-1.5 py-0 rounded-full"
            >
              {t("sql-editor.grant-type-export")}
            </Badge>
          )}
        </div>
        {expiration && (
          // The span stays outside the tooltip: it carries this row's layout,
          // and a tooltip with nothing to say renders its children bare.
          <span className="text-xs text-control-placeholder shrink-0">
            <Tooltip content={expiration.hiddenDeadline}>
              {expiration.text}
            </Tooltip>
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
            "w-full min-w-0 text-xs wrap-anywhere whitespace-pre-wrap font-mono line-clamp-2",
            (isExpired || isRejectedOrCanceled) &&
              "line-through text-control-placeholder"
          )}
        >
          {grant.query}
        </p>
      </Tooltip>

      <div className="w-full flex flex-col gap-y-2">
        {remainingTargetCount === 0 ? (
          databaseTargets
        ) : (
          <Tooltip
            content={
              <div className="flex max-w-lg flex-col gap-y-1">
                {grant.targets.map((target, index) => (
                  <DatabaseTargetDisplay
                    key={`${target}-${index}`}
                    target={target}
                    showEnvironment
                  />
                ))}
              </div>
            }
            side="right"
          >
            {databaseTargets}
          </Tooltip>
        )}

        <div className="flex items-center justify-between gap-x-1">
          <div>
            {isActive && (
              <Button
                size="xs"
                variant="default"
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
                appearance="secondary"
                size="xs"
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
              <RouterLink
                to={issueLink}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="inline-flex items-center justify-center h-6 text-xs px-2 rounded-xs hover:bg-control-bg text-control"
              >
                {t("sql-editor.view-issue")}
              </RouterLink>
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
