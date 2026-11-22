import type { TFunction } from "i18next";
import {
  ChevronDown,
  ExternalLink,
  FolderTree,
  ListChecks,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/app/router/handles";
import { markdownSummary } from "@/components/MarkdownEditor";
import { RouterLink } from "@/components/RouterLink";
import { FindingCodeBadge } from "@/components/sql-review/FindingCodeBadge";
import { standardRuleText } from "@/components/sql-review/standardRules";
import { UserAvatar } from "@/components/UserAvatar";
import { Badge, type BadgeProps, badgeVariants } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { Tooltip } from "@/components/ui/tooltip";
import { usePlanFeature, useUserByIdentifier } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { getProjectNameAndDatabaseGroupName } from "@/stores/modules/v1/common";
import { isValidDatabaseGroupName, unknownUser } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type IssueComment,
  type IssueComment_ReviewMetadata,
  IssueComment_ReviewMetadata_Priority,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { extractDatabaseGroupName } from "@/utils/v1/databaseGroup";
import { reviewerName } from "../../utils/reviewLabels";
import { getDatabaseGroupRouteParams } from "../../utils/targets";
import { PlanTargetDisplay } from "../PlanTargetDisplay";
import {
  groupTargetsByEnvironment,
  isPlainDatabaseName,
  type TargetGroup,
  targetGroupKey,
  useTargetDatabases,
} from "./productionTargets";
import {
  ReviewTargetList,
  TARGET_SEARCH_THRESHOLD,
  TargetGroupBadge,
  targetGroupLabel,
} from "./ReviewTargetList";
import { isStandardRuleResult } from "./threadModel";

// A list longer than this is summarized as a count per environment that opens
// the whole list; a shorter one is shown whole.
const SUMMARIZE_TARGETS_ABOVE = 4;
// How many environments a summary names before counting the rest.
const SUMMARY_ENVIRONMENTS = 3;

// The avatar and name of a comment's author. A review result has no creator,
// so the review stands in for a person.
export function useCommentAuthor(comment: IssueComment): {
  avatar: ReactNode;
  name: string;
} {
  const { t } = useTranslation();
  const review = comment.reviewMetadata;
  const user = useUserByIdentifier(review ? undefined : comment.creator);
  if (review) {
    return {
      avatar: (
        <span
          aria-hidden="true"
          className="flex size-5 shrink-0 items-center justify-center rounded-full bg-control-bg text-control"
          data-testid="reviewer-avatar"
        >
          <ReviewerIcon className="size-3.5" />
        </span>
      ),
      name: reviewerName(t),
    };
  }
  const creator = user ?? unknownUser(comment.creator);
  const name = creator.title || creator.email;
  return {
    avatar: <UserAvatar colorSeed={creator.email} size="xs" title={name} />,
    name,
  };
}

export function ReviewerIcon({ className }: { className: string }) {
  return <ListChecks className={className} />;
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

// A priority's badge color and, for P1 and P2, what resolving its thread
// means. Any open thread holds the rollout, whatever its priority.
export function priorityStyle(
  t: TFunction,
  priority: IssueComment_ReviewMetadata_Priority
): { description?: string; variant: BadgeProps["variant"] } | undefined {
  switch (priority) {
    case IssueComment_ReviewMetadata_Priority.P0:
      return { variant: "destructive" };
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

// A review result's heading line: its priority, then for the result of a
// standard rule the rule and the finding codes, and for any other result its
// title.
export function ReviewHeading({
  review,
}: {
  review: IssueComment_ReviewMetadata;
}) {
  const isRule = isStandardRuleResult(review);
  return (
    <div
      className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1"
      data-testid="review-heading"
    >
      <ReviewPriorityBadge priority={review.priority} />
      {isRule && <ReviewRule ruleType={review.ruleType} />}
      {isRule &&
        review.codes.map((code) => <FindingCodeBadge code={code} key={code} />)}
      {!isRule && review.title && (
        <span
          className="min-w-0 text-sm font-medium text-main"
          data-testid="review-title"
        >
          {markdownSummary(review.title)}
        </span>
      )}
    </div>
  );
}

// The standard rule a result reports, in the reader's language, with what the
// rule checks on hover.
export function ReviewRule({ ruleType }: { ruleType: ReviewRuleType }) {
  const { t } = useTranslation();
  if (ruleType === ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED) return null;
  const { title, description } = standardRuleText(t, ruleType);
  return (
    <Tooltip
      content={description}
      render={
        <span
          className="min-w-0 text-sm font-medium text-main"
          data-testid="review-rule"
          tabIndex={0}
        />
      }
    >
      {title}
    </Tooltip>
  );
}

// The databases a review result applies to, grouped by environment, closest to
// production first. A result without targets has nothing to list. A result for
// exactly the databases of its spec says nothing the spec does not, so the row
// is left out then too, but only when that equality is proven: for a spec on a
// database group, against the databases the group matches once they load.
export function ReviewTargets({
  spec,
  targets,
}: {
  // The spec the result is anchored to, when known.
  spec: Plan_Spec | undefined;
  targets: string[];
}) {
  const { t } = useTranslation();
  const loaded = useTargetDatabases(targets);
  const databaseGroup = useSpecDatabaseGroup(spec);
  const databasesByName = useAppStore((state) => state.databasesByName);
  const environmentList = useAppStore((state) => state.environmentList);
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const groups = useMemo(
    () =>
      loaded
        ? groupTargetsByEnvironment(
            targets,
            { databasesByName, environmentList },
            hasEnvTierFeature
          )
        : undefined,
    [databasesByName, environmentList, hasEnvTierFeature, loaded, targets]
  );
  if (targets.length === 0 || equalsSpecDatabases(targets, spec)) return null;
  if (
    databaseGroup &&
    (!databaseGroup.settled || sameSet(targets, databaseGroup.databases))
  ) {
    return null;
  }

  const total = databaseGroup
    ? databaseGroupTotal(targets, databaseGroup)
    : undefined;
  const countText =
    total === undefined
      ? t("plan.review.thread.targets.n-databases", { count: targets.length })
      : t("plan.review.thread.targets.n-of-m-databases", {
          count: targets.length,
          total,
        });

  return (
    <div
      className="flex min-w-0 items-center gap-x-2 text-xs"
      data-testid="review-targets"
    >
      <span className="shrink-0 text-control-light">
        {t("plan.review.thread.applies-to")}
      </span>
      {targets.length > SUMMARIZE_TARGETS_ABOVE ? (
        <TargetSummary
          countText={countText}
          databaseGroup={databaseGroup?.name}
          groups={groups}
          hasEnvTierFeature={hasEnvTierFeature}
          targets={targets}
        />
      ) : (
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1">
          {databaseGroup && (
            <>
              <DatabaseGroupMark name={databaseGroup.name} />
              <span className="shrink-0 text-control">{countText}</span>
            </>
          )}
          <ul className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            {groups
              ? groups.map((group) => (
                  <li
                    className="flex max-w-full min-w-0 flex-wrap items-center gap-x-2 gap-y-1"
                    key={targetGroupKey(group)}
                  >
                    <TargetGroupBadge
                      group={group}
                      hasEnvTierFeature={hasEnvTierFeature}
                    />
                    {group.targets.map((target) => (
                      <TargetChip key={target} target={target} />
                    ))}
                  </li>
                ))
              : targets.map((target) => (
                  <li className="flex max-w-full min-w-0" key={target}>
                    <TargetChip target={target} />
                  </li>
                ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function TargetChip({ target }: { target: string }) {
  return (
    <span className={cn(badgeVariants(), "max-w-full min-w-0")}>
      <PlanTargetDisplay className="text-xs" target={target} />
    </span>
  );
}

// A long list as one button: the database group when there is one, a count
// per environment, and the total, opening the whole list. Until the databases
// load it shows the total alone.
function TargetSummary({
  countText,
  databaseGroup,
  groups,
  hasEnvTierFeature,
  targets,
}: {
  countText: string;
  databaseGroup: string | undefined;
  groups: TargetGroup[] | undefined;
  hasEnvTierFeature: boolean;
  targets: string[];
}) {
  const { t } = useTranslation();
  const searchRef = useRef<HTMLInputElement>(null);
  const popupRef = useRef<HTMLDivElement>(null);
  const named = groups?.slice(0, SUMMARY_ENVIRONMENTS) ?? [];
  const unnamed = (groups?.length ?? 0) - named.length;
  const label = [
    t("plan.review.thread.applies-to"),
    databaseGroup &&
      `${t("common.database-group")} ${extractDatabaseGroupName(databaseGroup)}`,
    countText,
    ...(groups ?? []).map(
      (group) => `${targetGroupLabel(t, group)} ${group.targets.length}`
    ),
  ]
    .filter(Boolean)
    .join(", ");
  return (
    <Popover>
      <PopoverTrigger
        aria-label={label}
        className="-ml-1 flex min-w-0 cursor-pointer flex-wrap items-center gap-x-2 gap-y-1 rounded-xs px-1 py-0.5 text-left hover:bg-control-bg focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent data-[popup-open]:bg-control-bg"
      >
        {databaseGroup && <DatabaseGroupMark name={databaseGroup} />}
        {named.map((group) => (
          <span
            className="flex min-w-0 items-center gap-x-1"
            key={targetGroupKey(group)}
          >
            <TargetGroupBadge
              group={group}
              hasEnvTierFeature={hasEnvTierFeature}
            />
            <span className="text-control tabular-nums">
              {group.targets.length}
            </span>
          </span>
        ))}
        {unnamed > 0 && (
          <span className="text-control-light tabular-nums">
            {t("plan.review.thread.targets.more-environments", { n: unnamed })}
          </span>
        )}
        <span className="flex items-center gap-x-1 whitespace-nowrap text-accent">
          {countText}
          <ChevronDown className="size-3.5" />
        </span>
      </PopoverTrigger>
      <PopoverContent
        ref={popupRef}
        align="start"
        aria-label={t("plan.review.thread.targets.list-label")}
        className="flex w-90 flex-col gap-y-2"
        initialFocus={
          targets.length > TARGET_SEARCH_THRESHOLD ? searchRef : popupRef
        }
        side="bottom"
      >
        {databaseGroup && (
          <>
            <DatabaseGroupHeader countText={countText} name={databaseGroup} />
            <Separator />
          </>
        )}
        <ReviewTargetList
          groups={groups}
          hasEnvTierFeature={hasEnvTierFeature}
          searchRef={searchRef}
          targets={targets}
        />
      </PopoverContent>
    </Popover>
  );
}

function DatabaseGroupMark({ name }: { name: string }) {
  const { t } = useTranslation();
  return (
    <span className="flex min-w-0 items-center gap-x-1 text-control">
      <FolderTree className="size-3.5 shrink-0" />
      <Badge className="shrink-0">{t("common.database-group")}</Badge>
      <span className="min-w-0 truncate font-medium">
        {extractDatabaseGroupName(name)}
      </span>
    </span>
  );
}

function DatabaseGroupHeader({
  countText,
  name,
}: {
  countText: string;
  name: string;
}) {
  const { t } = useTranslation();
  const [projectName, databaseGroupName] =
    getProjectNameAndDatabaseGroupName(name);
  return (
    <div className="flex min-w-0 items-center gap-x-2 text-xs">
      <FolderTree className="size-3.5 shrink-0 text-control" />
      <span className="min-w-0 truncate font-medium text-main">
        {extractDatabaseGroupName(name)}
      </span>
      <span className="shrink-0 text-control-light">{countText}</span>
      <RouterLink
        className="ml-auto flex shrink-0 items-center gap-x-1 text-accent hover:underline"
        rel="noreferrer"
        target="_blank"
        to={{
          name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
          params: getDatabaseGroupRouteParams({
            databaseGroupName,
            projectName,
          }),
        }}
      >
        {t("plan.review.thread.targets.open-group")}
        <ExternalLink className="size-3.5" />
      </RouterLink>
    </div>
  );
}

type SpecDatabaseGroup = {
  name: string;
  // The databases the group matches now, once loaded.
  databases: string[] | undefined;
  // The project's sampling size for reviews, once the project is loaded.
  samplingSize: number | undefined;
  // Whether the group has loaded or failed to.
  settled: boolean;
};

// The database group a spec targets, loaded with the databases it matches.
function useSpecDatabaseGroup(
  spec: Plan_Spec | undefined
): SpecDatabaseGroup | undefined {
  const name = specDatabaseGroupName(spec);
  const group = useAppStore((state) =>
    name && state.dbGroupViewByName[name] === DatabaseGroupView.FULL
      ? state.dbGroupsByName[name]
      : undefined
  );
  const projectId = name ? getProjectNameAndDatabaseGroupName(name)[0] : "";
  const samplingSize = useAppStore((state) =>
    projectId
      ? state.projectsByName[`projects/${projectId}`]?.ciSamplingSize
      : undefined
  );
  const [failedName, setFailedName] = useState<string>();
  useEffect(() => {
    if (!name || group) return;
    let cancelled = false;
    void useAppStore
      .getState()
      .fetchDBGroup(name, DatabaseGroupView.FULL)
      .then((fetched) => {
        if (!cancelled && !fetched) setFailedName(name);
      });
    return () => {
      cancelled = true;
    };
  }, [group, name]);
  const databases = useMemo(
    () => group?.matchedDatabases.map((database) => database.name),
    [group]
  );
  if (!name) return undefined;
  return {
    name,
    databases,
    samplingSize,
    settled: Boolean(group) || failedName === name,
  };
}

// The group a sheet-backed change spec targets as its one target.
function specDatabaseGroupName(spec: Plan_Spec | undefined) {
  if (
    spec?.config.case !== "changeDatabaseConfig" ||
    !spec.config.value.sheet ||
    spec.config.value.release
  ) {
    return undefined;
  }
  const { targets } = spec.config.value;
  return targets.length === 1 && isValidDatabaseGroupName(targets[0])
    ? targets[0]
    : undefined;
}

// How many databases the group holds, when the result's count can be read
// against it: the review read every database the group matches, and none of
// the result's databases has left the group since. A sampled review read only
// some, so it gives none.
function databaseGroupTotal(
  targets: readonly string[],
  { databases, samplingSize }: SpecDatabaseGroup
): number | undefined {
  if (!databases || samplingSize === undefined) return undefined;
  if (samplingSize > 0 && samplingSize < databases.length) return undefined;
  const members = new Set(databases);
  return targets.every((target) => members.has(target))
    ? databases.length
    : undefined;
}

function sameSet(
  targets: readonly string[],
  databases: readonly string[] | undefined
): boolean {
  if (!databases) return false;
  const expected = new Set(databases);
  const actual = new Set(targets);
  return (
    actual.size === expected.size &&
    [...actual].every((target) => expected.has(target))
  );
}

// Whether targets, as a set, are exactly the databases the spec lists. Only
// a sheet-backed change spec naming plain databases is compared; an unknown
// spec, a release-backed one, or one naming a database group never matches.
// Names compare as exact strings, since a project instance's database and a
// workspace instance's database of the same names are different databases.
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
  if (specTargets.length === 0 || !specTargets.every(isPlainDatabaseName)) {
    return false;
  }
  return sameSet(targets, specTargets);
}
