import type { TFunction } from "i18next";
import { Bot, ListChecks } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { UserAvatar } from "@/components/UserAvatar";
import { Badge, type BadgeProps, badgeVariants } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useUserByIdentifier } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { unknownUser } from "@/types";
import {
  type IssueComment,
  IssueComment_ReviewMetadata_Priority,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanTargetDisplay } from "../PlanTargetDisplay";

// A long target list keeps its head visible and folds the rest behind a count.
const VISIBLE_TARGETS = 3;
const FOLD_TARGETS_ABOVE = 4;

// The avatar and name of a comment's author. A review result has no creator,
// so the reviewer its review metadata names stands in for a person.
export function useCommentAuthor(comment: IssueComment): {
  avatar: ReactNode;
  name: string;
} {
  const { t } = useTranslation();
  const review = comment.reviewMetadata;
  const user = useUserByIdentifier(review ? undefined : comment.creator);
  if (review) {
    const Icon = review.runType === ReviewRun_Type.AI ? Bot : ListChecks;
    return {
      avatar: (
        <span
          aria-hidden="true"
          className="flex size-5 shrink-0 items-center justify-center rounded-full bg-control-bg text-control"
          data-testid="reviewer-avatar"
        >
          <Icon className="size-3.5" />
        </span>
      ),
      name: reviewerName(t, review.runType),
    };
  }
  const creator = user ?? unknownUser(comment.creator);
  const name = creator.title || creator.email;
  return {
    avatar: <UserAvatar colorSeed={creator.email} size="xs" title={name} />,
    name,
  };
}

function reviewerName(t: TFunction, runType: ReviewRun_Type): string {
  switch (runType) {
    case ReviewRun_Type.AI:
      return t("sql-review.ai-review.self");
    case ReviewRun_Type.RULE:
      return t("sql-review.standard-rules.self");
    default:
      return t("sql-review.title");
  }
}

export function ReviewPriorityBadge({
  priority,
}: {
  priority: IssueComment_ReviewMetadata_Priority;
}) {
  const { t } = useTranslation();
  const style = priorityStyle(t, priority);
  if (!style) return null;
  return (
    <Badge
      className="shrink-0"
      data-testid="review-priority"
      title={style.description}
      variant={style.variant}
    >
      {IssueComment_ReviewMetadata_Priority[priority]}
    </Badge>
  );
}

function priorityStyle(
  t: TFunction,
  priority: IssueComment_ReviewMetadata_Priority
): { description: string; variant: BadgeProps["variant"] } | undefined {
  switch (priority) {
    case IssueComment_ReviewMetadata_Priority.P0:
      return {
        description: t("plan.review.thread.priority.p0"),
        variant: "destructive",
      };
    case IssueComment_ReviewMetadata_Priority.P1:
      return {
        description: t("plan.review.thread.priority.p1"),
        variant: "warning",
      };
    case IssueComment_ReviewMetadata_Priority.P2:
      return {
        description: t("plan.review.thread.priority.p2"),
        variant: "default",
      };
    default:
      return undefined;
  }
}

export function ReviewTargets({ targets }: { targets: string[] }) {
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);
  if (targets.length === 0) return null;
  const folded = !showAll && targets.length > FOLD_TARGETS_ABOVE;
  const visible = folded ? targets.slice(0, VISIBLE_TARGETS) : targets;
  return (
    <div
      className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs"
      data-testid="review-targets"
    >
      <span className="shrink-0 text-control-light">
        {t("plan.review.thread.applies-to")}
      </span>
      <ul className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        {visible.map((target) => (
          <li
            className={cn(badgeVariants(), "max-w-full min-w-0")}
            key={target}
          >
            <PlanTargetDisplay className="text-xs" target={target} />
          </li>
        ))}
      </ul>
      {folded && (
        <Button appearance="link" onClick={() => setShowAll(true)} size="xs">
          {t("common.n-more", { n: targets.length - VISIBLE_TARGETS })}
        </Button>
      )}
    </div>
  );
}
