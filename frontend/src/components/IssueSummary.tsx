import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { IssueStatusIcon } from "@/components/IssueStatusIcon";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { extractIssueUID } from "@/utils";

interface IssueSummaryProps {
  issue: Issue;
  titleContent?: ReactNode;
  titleTrailing?: ReactNode;
  renderCreator?: (creator: User) => ReactNode;
  metadataTrailing?: ReactNode;
  timeTooltip?: boolean;
}

export function IssueSummary({
  issue,
  titleContent,
  titleTrailing,
  renderCreator,
  metadataTrailing,
  timeTooltip = false,
}: IssueSummaryProps) {
  const { t } = useTranslation();
  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(issue.creator)
  );
  const creator = creatorUser || unknownUser(issue.creator);
  const createTimeMs = getTimeForPbTimestampProtoEs(issue.createTime);
  const statusLabel =
    issue.status === IssueStatus.OPEN
      ? t("issue.table.open")
      : issue.status === IssueStatus.DONE
        ? t("common.done")
        : issue.status === IssueStatus.CANCELED
          ? t("common.canceled")
          : undefined;

  return (
    <div data-slot="issue-summary" className="min-w-0 flex-1">
      <div className="flex items-center gap-x-1.5">
        <div className="h-6 flex justify-center items-center">
          <IssueStatusIcon status={issue.status} />
        </div>
        {statusLabel && <span className="sr-only">{statusLabel}</span>}
        {titleContent ?? (
          <span
            className={cn(
              "min-w-0 block truncate font-medium text-base",
              issue.title ? "text-main" : "italic text-control-placeholder"
            )}
            title={issue.title || undefined}
          >
            {issue.title || t("common.untitled")}
          </span>
        )}
        {titleTrailing}
      </div>
      <div className="flex items-center flex-wrap gap-x-1 text-xs text-control-light mt-1">
        <span className="opacity-80">#{extractIssueUID(issue.name)}</span>
        {createTimeMs !== undefined && (
          <>
            <span>&middot;</span>
            {t("common.created")}
            <HumanizeTs tsMs={createTimeMs} tooltip={timeTooltip} />
          </>
        )}
        <span>&middot;</span>
        {renderCreator ? renderCreator(creator) : <span>{creator.title}</span>}
        {metadataTrailing}
      </div>
    </div>
  );
}
