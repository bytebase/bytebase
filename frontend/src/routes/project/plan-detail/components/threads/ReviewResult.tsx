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
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { PlanTargetDisplay } from "../PlanTargetDisplay";

// A long target list keeps its head visible and folds the rest behind a count.
const VISIBLE_TARGETS = 3;
const FOLD_TARGETS_ABOVE = 4;

// A plain database target, in the form review results name theirs.
const DATABASE_NAME = /^instances\/[^/]+\/databases\/[^/]+$/;

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

// The databases a review result applies to. A result without targets has
// nothing to list. A result for exactly the databases of its spec says
// nothing the spec does not, so the row is left out then too, but only when
// that equality is proven.
export function ReviewTargets({
  spec,
  targets,
}: {
  // The spec the result is anchored to, when known.
  spec: Plan_Spec | undefined;
  targets: string[];
}) {
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);
  if (targets.length === 0 || equalsSpecDatabases(targets, spec)) return null;
  const folded = !showAll && targets.length > FOLD_TARGETS_ABOVE;
  const visible = folded ? targets.slice(0, VISIBLE_TARGETS) : targets;
  return (
    <div
      className="flex min-w-0 items-baseline gap-x-2 text-xs"
      data-testid="review-targets"
    >
      <span className="shrink-0 text-control-light">
        {t("plan.review.thread.applies-to")}
      </span>
      <ul className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1">
        {visible.map((target) => (
          <li
            className={cn(badgeVariants(), "max-w-full min-w-0")}
            key={target}
          >
            <PlanTargetDisplay className="text-xs" target={target} />
          </li>
        ))}
        {folded && (
          <li>
            <Button
              appearance="link"
              onClick={() => setShowAll(true)}
              size="xs"
            >
              {t("common.n-more", { n: targets.length - VISIBLE_TARGETS })}
            </Button>
          </li>
        )}
      </ul>
    </div>
  );
}

// Whether targets, as a set, are exactly the databases the spec lists. Only
// a sheet-backed change spec naming plain databases is compared; an unknown
// spec, a release-backed one, or one naming a database group never matches.
function equalsSpecDatabases(
  targets: readonly string[],
  spec: Plan_Spec | undefined
): boolean {
  if (
    spec?.config.case !== "changeDatabaseConfig" ||
    !spec.config.value.sheet ||
    spec.config.value.release
  ) {
    return false;
  }
  const specTargets = spec.config.value.targets;
  if (
    specTargets.length === 0 ||
    !specTargets.every((target) => DATABASE_NAME.test(target))
  ) {
    return false;
  }
  const expected = new Set(specTargets);
  const actual = new Set(targets);
  return (
    actual.size === expected.size &&
    [...actual].every((target) => expected.has(target))
  );
}
