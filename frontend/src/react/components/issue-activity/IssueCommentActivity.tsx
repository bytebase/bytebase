// Shared rendering for issue-comment activity items — the colored action icon
// badge and the human-readable action sentence (including rich plan-spec diff
// rows). Used by both the issue-detail comment list and the plan-detail review
// timeline so their icons, wordings, and detailed styles stay consistent.
import { Loader2 } from "lucide-react";
import { Fragment, type ReactNode, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import {
  type ActivityIconSpec,
  ICON_BADGE_TONE,
  ISSUE_EVENT_ICON,
  PLAN_CHANGE_ICON,
  REVIEW_DECISION_ICON,
} from "@/react/components/issue-activity/activityIcons";
import { ReadonlyDiffMonaco } from "@/react/components/monaco";
import { RouterLink } from "@/react/components/RouterLink";
import { UserAvatar } from "@/react/components/UserAvatar";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  diffEntryKey,
  diffPlanSpecsForEvent,
  type SpecDiffEntry,
} from "@/react/lib/plan/diffPlanSpecs";
import { cn } from "@/react/lib/utils";
import { router } from "@/react/router";
import { buildPlanDeployRouteFromPlanName } from "@/react/router/routeHelpers";
import { useAppStore } from "@/react/stores/app";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";
import { extractUserEmail } from "@/store";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_Type,
  IssueComment_Approval_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { extractPlanUID, getSpecDisplayInfo } from "@/utils";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import {
  enablePriorBackupOfSpec,
  sheetNameOfSpec,
  targetsOfSpec,
} from "@/utils/v1/issue/plan";
import { getSheetStatement } from "@/utils/v1/sheet";

interface ActivityProps {
  issue?: Issue;
  plan?: Plan;
  comment: IssueComment;
  // When true (the plan-detail review timeline, which already sits on the
  // plan), spec/plan references render as plain text instead of links — the
  // links would only redirect to the page you're already on (BYT-9710). The
  // issue-detail page leaves this off so its navigation links stay.
  linkless?: boolean;
}

// A DONE issue-update that created the rollout for a database-change plan — the
// timeline renders this as a "review done, rollout created" line rather than a
// plain "resolved issue".
function isDoneRolloutComment(
  issue: Issue | undefined,
  plan: Plan | undefined,
  comment: IssueComment
): boolean {
  if (comment.event.case !== "issueUpdate") {
    return false;
  }
  const { fromStatus, toStatus } = comment.event.value;
  return (
    issue?.type === Issue_Type.DATABASE_CHANGE &&
    plan?.hasRollout === true &&
    getIssueCommentType(comment) === IssueCommentType.ISSUE_UPDATE &&
    fromStatus !== undefined &&
    fromStatus !== IssueStatus.DONE &&
    toStatus === IssueStatus.DONE
  );
}

// Whether the current user may edit a comment: only user comments and approval
// decisions that carry a note are editable, and only by their author or someone
// with the update permission. Shared so both activity surfaces gate edits alike.
export function canEditIssueComment(
  comment: IssueComment,
  currentUserEmail: string,
  project: Parameters<typeof hasProjectPermissionV2>[0] | undefined
): boolean {
  if (!project) {
    return false;
  }
  const type = getIssueCommentType(comment);
  const editable =
    type === IssueCommentType.USER_COMMENT ||
    (type === IssueCommentType.APPROVAL && comment.comment !== "");
  if (!editable) {
    return false;
  }
  if (currentUserEmail === extractUserEmail(comment.creator)) {
    return true;
  }
  return hasProjectPermissionV2(project, "bb.issueComments.update");
}

// The timeline row shell shared by every activity item: the connector line, the
// left icon gutter, and a body that is a bordered card when present or a
// borderless inline header when absent. This is the single source of the
// activity-item layout for both the issue-detail list and the review timeline.
export function ActivityRowShell({
  body,
  header,
  icon,
  id,
  isLast,
  subjectSuffix,
}: {
  body?: ReactNode;
  header: ReactNode;
  icon: ReactNode;
  id?: string;
  isLast: boolean;
  subjectSuffix?: ReactNode;
}) {
  return (
    <li>
      <div className="relative pb-3" id={id}>
        {!isLast && (
          <span
            aria-hidden="true"
            className="absolute left-4 -ml-px h-full w-0.5 bg-gray-200"
          />
        )}
        <div className="relative flex items-start">
          <div className="pt-1.5">{icon}</div>
          <div className="min-w-0 flex-1">
            <div
              className={cn(
                "overflow-hidden rounded-lg border",
                body
                  ? "ml-3 border-gray-200 bg-white"
                  : "ml-1 border-transparent"
              )}
            >
              <div className={cn("px-3 py-2", body && "bg-gray-50")}>
                <div className="flex items-center justify-between">
                  <div className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
                    {header}
                  </div>
                  {subjectSuffix}
                </div>
              </div>
              {body && (
                <div className="wrap-break-word border-t border-gray-200 px-3 py-2 text-sm text-gray-700 [&_.markdown-body>div>:first-child]:mt-0 [&_.markdown-body>div>:last-child]:mb-0">
                  {body}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </li>
  );
}

export function CommentCreator({ creator }: { creator: User }) {
  return (
    <span className="font-medium text-main">
      {creator.title || creator.email}
    </span>
  );
}

// Header line for a comment-backed activity item: creator (suppressed for the
// done-rollout system comment, whose sentence already names the actor), action
// sentence, timestamp, and an "(edited)" marker for edited user comments.
function IssueCommentHeader({ comment, issue, linkless, plan }: ActivityProps) {
  const { t } = useTranslation();
  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(comment.creator)
  );
  const creator = creatorUser ?? unknownUser(comment.creator);
  const createdTs = getTimeForPbTimestampProtoEs(comment.createTime, 0);
  const updatedTs = getTimeForPbTimestampProtoEs(comment.updateTime, 0);
  const isEdited =
    createdTs !== updatedTs &&
    getIssueCommentType(comment) === IssueCommentType.USER_COMMENT;

  return (
    <>
      {!isDoneRolloutComment(issue, plan, comment) && (
        <CommentCreator creator={creator} />
      )}
      <IssueCommentActionSentence
        comment={comment}
        issue={issue}
        linkless={linkless}
        plan={plan}
      />
      {comment.createTime && (
        <HumanizeTs className="text-gray-500" ts={createdTs / 1000} />
      )}
      {isEdited && (
        <span className="text-xs text-gray-500">({t("common.edited")})</span>
      )}
    </>
  );
}

// A complete comment-backed activity row: icon + header assembled from the
// shared pieces, with an optional body and edit affordance supplied by the page.
export function IssueCommentRow({
  body,
  comment,
  isLast,
  issue,
  linkless,
  plan,
  subjectSuffix,
}: ActivityProps & {
  body?: ReactNode;
  isLast: boolean;
  subjectSuffix?: ReactNode;
}) {
  return (
    <ActivityRowShell
      body={body}
      header={
        <IssueCommentHeader
          comment={comment}
          issue={issue}
          linkless={linkless}
          plan={plan}
        />
      }
      icon={
        <IssueCommentActionIcon comment={comment} issue={issue} plan={plan} />
      }
      id={comment.name}
      isLast={isLast}
      subjectSuffix={subjectSuffix}
    />
  );
}

function IssueCommentActionIcon({ issue, plan, comment }: ActivityProps) {
  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(comment.creator)
  );
  const user = creatorUser ?? unknownUser(comment.creator);
  const commentType = getIssueCommentType(comment);

  if (
    commentType === IssueCommentType.APPROVAL &&
    comment.event.case === "approval"
  ) {
    const { status } = comment.event.value;
    if (status === IssueComment_Approval_Status.APPROVED) {
      return <ActivityBadge spec={REVIEW_DECISION_ICON.approved} />;
    }
    if (status === IssueComment_Approval_Status.REJECTED) {
      return <ActivityBadge spec={REVIEW_DECISION_ICON.rejected} />;
    }
    if (status === IssueComment_Approval_Status.PENDING) {
      return <ActivityBadge spec={REVIEW_DECISION_ICON.reRequested} />;
    }
  }

  if (
    commentType === IssueCommentType.ISSUE_UPDATE &&
    comment.event.case === "issueUpdate"
  ) {
    if (isDoneRolloutComment(issue, plan, comment)) {
      return <ActivityBadge spec={ISSUE_EVENT_ICON.resolved} />;
    }

    const { fromLabels, toDescription, toLabels, toTitle } =
      comment.event.value;
    if (
      toTitle !== undefined ||
      toDescription !== undefined ||
      toLabels.length !== 0 ||
      fromLabels.length !== 0
    ) {
      return <ActivityBadge spec={ISSUE_EVENT_ICON.fieldEdit} />;
    }
  }

  if (
    commentType === IssueCommentType.PLAN_UPDATE &&
    comment.event.case === "planUpdate"
  ) {
    const entries = diffPlanSpecsForEvent(comment.event.value);
    const allAdded =
      entries.length > 0 && entries.every((e) => e.kind === "added");
    const allRemoved =
      entries.length > 0 && entries.every((e) => e.kind === "removed");
    if (allAdded) return <ActivityBadge spec={PLAN_CHANGE_ICON.added} />;
    if (allRemoved) return <ActivityBadge spec={PLAN_CHANGE_ICON.deleted} />;
    return <ActivityBadge spec={PLAN_CHANGE_ICON.edited} />;
  }

  return (
    <div className="relative pl-0.5">
      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-white ring-4 ring-white">
        <UserAvatar
          className="h-7 w-7 text-[0.8rem] font-medium"
          size="sm"
          title={user.title || user.email}
        />
      </div>
    </div>
  );
}

export function CommentIconBadge({
  className,
  icon,
}: {
  className: string;
  icon: ReactNode;
}) {
  return (
    <div className="relative pl-0.5">
      <div
        className={cn(
          "flex h-7 w-7 items-center justify-center rounded-full ring-4 ring-white",
          className
        )}
      >
        {icon}
      </div>
    </div>
  );
}

// Renders an activity badge from a design-module icon spec — the white glyph on
// a tone-colored circle. The single place activity badges are built.
function ActivityBadge({ spec }: { spec: ActivityIconSpec }) {
  const { Icon, tone } = spec;
  return (
    <CommentIconBadge
      className={ICON_BADGE_TONE[tone]}
      icon={<Icon className="size-4" />}
    />
  );
}

function IssueCommentActionSentence({
  issue,
  plan,
  comment,
  linkless,
}: ActivityProps) {
  const { t } = useTranslation();
  const commentType = getIssueCommentType(comment);

  if (
    commentType === IssueCommentType.APPROVAL &&
    comment.event.case === "approval"
  ) {
    const { status } = comment.event.value;
    if (status === IssueComment_Approval_Status.APPROVED) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("custom-approval.issue-review.approved-issue")}
        </span>
      );
    }
    if (status === IssueComment_Approval_Status.REJECTED) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("custom-approval.issue-review.rejected-issue")}
        </span>
      );
    }
    if (status === IssueComment_Approval_Status.PENDING) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("custom-approval.issue-review.re-requested-review")}
        </span>
      );
    }
  }

  if (
    commentType === IssueCommentType.ISSUE_UPDATE &&
    comment.event.case === "issueUpdate"
  ) {
    const {
      fromDescription,
      fromLabels,
      fromStatus,
      fromTitle,
      toDescription,
      toLabels,
      toStatus,
      toTitle,
    } = comment.event.value;
    if (fromTitle !== undefined && toTitle !== undefined) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("activity.sentence.changed-from-to", {
            name: t("issue.issue-name").toLowerCase(),
            newValue: toTitle,
            oldValue: fromTitle,
          })}
        </span>
      );
    }
    if (fromDescription !== undefined && toDescription !== undefined) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("activity.sentence.changed-description")}
        </span>
      );
    }
    if (fromStatus !== undefined && toStatus !== undefined) {
      if (toStatus === IssueStatus.DONE) {
        if (isDoneRolloutComment(issue, plan, comment)) {
          // On the plan-detail timeline (linkless) the rollout is for the plan
          // you're already viewing, so drop the "for plan #id" link entirely.
          const planUID = linkless || !plan ? "" : extractPlanUID(plan.name);
          const planRoute =
            linkless || !plan
              ? undefined
              : buildPlanDeployRouteFromPlanName(plan.name);
          const sentence =
            issue?.approvalStatus === ApprovalStatus.APPROVED
              ? planUID
                ? t("activity.sentence.review-done-rollout-created-for-plan")
                : t("activity.sentence.review-done-rollout-created")
              : planUID
                ? t("activity.sentence.review-skipped-rollout-created-for-plan")
                : t("activity.sentence.review-skipped-rollout-created");

          return (
            <span className="wrap-break-word min-w-0 text-gray-600">
              {sentence}
              {planUID && planRoute && (
                <>
                  {" "}
                  <RouterLink
                    className="font-medium text-accent hover:underline"
                    to={planRoute}
                  >
                    #{planUID}
                  </RouterLink>
                </>
              )}
            </span>
          );
        }
        return (
          <span className="wrap-break-word min-w-0 text-gray-600">
            {t("activity.sentence.resolved-issue")}
          </span>
        );
      }
      if (toStatus === IssueStatus.CANCELED) {
        return (
          <span className="wrap-break-word min-w-0 text-gray-600">
            {t("activity.sentence.canceled-issue")}
          </span>
        );
      }
      if (toStatus === IssueStatus.OPEN) {
        return (
          <span className="wrap-break-word min-w-0 text-gray-600">
            {t("activity.sentence.reopened-issue")}
          </span>
        );
      }
    }
    if (fromLabels.length !== 0 || toLabels.length !== 0) {
      return (
        <span className="wrap-break-word min-w-0 text-gray-600">
          {t("activity.sentence.changed-labels")}
        </span>
      );
    }
  }

  if (
    commentType === IssueCommentType.PLAN_UPDATE &&
    comment.event.case === "planUpdate"
  ) {
    const entries = diffPlanSpecsForEvent(comment.event.value);
    if (entries.length === 0) return null;
    if (entries.length === 1) {
      return (
        <SpecDiffRow
          entry={entries[0]}
          linkless={linkless}
          multi={false}
          plan={plan}
        />
      );
    }
    return (
      <div className="flex flex-col gap-1">
        {entries.map((entry) => (
          <SpecDiffRow
            entry={entry}
            key={diffEntryKey(entry)}
            linkless={linkless}
            multi
            plan={plan}
          />
        ))}
      </div>
    );
  }

  return <span className="wrap-break-word min-w-0 text-gray-600" />;
}

function SpecDiffRow({
  entry,
  linkless,
  multi,
  plan,
}: {
  entry: SpecDiffEntry;
  linkless?: boolean;
  multi?: boolean;
  plan?: Plan;
}) {
  const { t } = useTranslation();
  const planName = plan?.name ?? "";

  if (entry.kind === "added") {
    return (
      <SpecChangeRow
        linkless={linkless}
        plan={plan}
        showIndex={multi}
        specRef={specResourceName(planName, entry.spec)}
      >
        {t("activity.sentence.added-spec")}
      </SpecChangeRow>
    );
  }

  if (entry.kind === "removed") {
    return (
      <SpecChangeRow
        linkless={linkless}
        plan={plan}
        showIndex={multi}
        specRef={specResourceName(planName, entry.spec)}
      >
        {t("activity.sentence.removed-spec")}
      </SpecChangeRow>
    );
  }

  // updated. The verb fragments ("modified SQL of", "changed targets of")
  // precede the spec chip; the detail for each (SQL diff button, target +/-
  // list) trails *after* the chip so the sentence reads
  // "<verb> Change <detail>" rather than splicing the detail before the noun.
  const fragments: ReactNode[] = [];
  const trailingItems: ReactNode[] = [];
  if (entry.sheetChanged) {
    const fromSheet = sheetNameOfSpec(entry.from);
    const toSheet = sheetNameOfSpec(entry.to);
    fragments.push(
      <span key="sheet">{t("activity.sentence.modified-sql-of")}</span>
    );
    trailingItems.push(
      <IssueStatementUpdateButton
        key="sql"
        newSheet={toSheet}
        oldSheet={fromSheet}
      />
    );
  }
  if (entry.targetsChanged) {
    const fromTargets = targetsOfSpec(entry.from);
    const toTargets = targetsOfSpec(entry.to);
    const fromSet = new Set(fromTargets);
    const toSet = new Set(toTargets);
    const added = toTargets.filter((x) => !fromSet.has(x));
    const removed = fromTargets.filter((x) => !toSet.has(x));
    const diffText = [
      added.length > 0 ? `+${added.join(", ")}` : null,
      removed.length > 0 ? `-${removed.join(", ")}` : null,
    ]
      .filter(Boolean)
      .join("  ");
    fragments.push(
      <span key="targets">{t("activity.sentence.changed-targets-of")}</span>
    );
    // The raw target resource paths are noise on the plan-detail timeline
    // (linkless) — you're already viewing the plan — so only the issue page
    // shows the +/- target diff.
    if (diffText && !linkless) {
      trailingItems.push(
        <span className="text-xs text-control-light" key="targets-diff">
          {diffText}
        </span>
      );
    }
  }
  if (entry.priorBackupChanged) {
    const flipped = enablePriorBackupOfSpec(entry.to);
    fragments.push(
      <span key="backup">
        {flipped
          ? t("activity.sentence.enabled-prior-backup-on")
          : t("activity.sentence.disabled-prior-backup-on")}
      </span>
    );
  }
  if (fragments.length === 0 && entry.otherChanged) {
    // Unknown attribute change — generic fallback with a JSON-diff toggle.
    return (
      <SpecChangeRow
        linkless={linkless}
        plan={plan}
        showIndex={multi}
        specRef={specResourceName(planName, entry.to)}
      >
        <span>{t("common.updated")}</span>
        <details className="ml-2 text-xs">
          <summary className="cursor-pointer text-control-light">
            {t("common.detail")}
          </summary>
          <pre className="mt-1 max-h-48 overflow-auto rounded bg-control-bg p-2">
            {JSON.stringify({ from: entry.from, to: entry.to }, null, 2)}
          </pre>
        </details>
      </SpecChangeRow>
    );
  }

  return (
    <SpecChangeRow
      linkless={linkless}
      plan={plan}
      showIndex={multi}
      specRef={specResourceName(planName, entry.to)}
      trailing={trailingItems.length > 0 ? <>{trailingItems}</> : null}
    >
      {joinFragments(fragments, t("common.and"))}
    </SpecChangeRow>
  );
}

function specResourceName(planName: string, spec: Plan_Spec): string {
  return planName ? `${planName}/specs/${spec.id}` : `specs/${spec.id}`;
}

function SpecChangeRow({
  children,
  linkless,
  plan,
  showIndex,
  specRef,
  trailing,
}: {
  children: ReactNode;
  linkless?: boolean;
  plan?: Plan;
  showIndex?: boolean;
  specRef: string;
  trailing?: ReactNode;
}) {
  const { t } = useTranslation();
  const specs = plan?.specs ?? [];
  const specInfo = getSpecDisplayInfo(specs, specRef);
  const specIdFromRef = specRef.match(/\/specs\/([^/]+)$/)?.[1] ?? "";
  const specId = specInfo?.specId ?? specIdFromRef;
  const specIdShort = specId.slice(0, 8);
  // Only link to specs that still exist in the live plan — otherwise the spec
  // view would silently bounce back to specs[0]. On the plan-detail timeline
  // (linkless) we're already on the plan, so the link would just re-navigate to
  // the current page and the id slice is meaningless noise — render plain text.
  const specRoute =
    !linkless && specInfo?.specId
      ? {
          query: {
            ...router.currentRoute.value.query,
            spec: specInfo.specId,
          },
        }
      : null;

  // On the plan-detail timeline (linkless) we identify the change by its 1-based
  // index in the plan — shown for multi-change and edit events — or mark it
  // "(Deleted)" when the spec no longer exists. The issue page keeps the linked
  // id slice.
  let chipSuffix: ReactNode = null;
  if (linkless) {
    if (specInfo == null) {
      chipSuffix = (
        <span className="text-xs text-control-placeholder">
          ({t("common.deleted")})
        </span>
      );
    } else if (showIndex) {
      chipSuffix = (
        <span className="rounded-full bg-control-bg px-1.5 py-0.5 text-xs text-main">
          {specInfo.displayIndex}
        </span>
      );
    }
  } else if (specIdShort !== "") {
    chipSuffix = (
      <span className="rounded-full bg-control-bg px-1.5 py-0.5 text-xs text-main">
        {specIdShort}
      </span>
    );
  }

  const chip = (
    <span className="inline-flex items-center gap-1">
      {t("plan.spec.change")}
      {chipSuffix}
    </span>
  );

  return (
    <span className="inline-flex items-center gap-1 whitespace-nowrap text-gray-600">
      {children}{" "}
      {specRoute != null ? (
        <RouterLink
          className="inline-flex items-center gap-1 hover:underline"
          to={specRoute}
        >
          {chip}
        </RouterLink>
      ) : (
        chip
      )}
      {trailing != null ? <> {trailing}</> : null}
    </span>
  );
}

function joinFragments(fragments: ReactNode[], separator: string): ReactNode {
  if (fragments.length === 0) return null;
  if (fragments.length === 1) return fragments[0];
  return fragments.map((f, i) => (
    <Fragment key={i}>
      {i > 0 ? <span> {separator} </span> : null}
      {f}
    </Fragment>
  ));
}

function IssueStatementUpdateButton({
  newSheet,
  oldSheet,
}: {
  newSheet: string;
  oldSheet: string;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [oldStatement, setOldStatement] = useState("");
  const [newStatement, setNewStatement] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    let canceled = false;
    const load = async () => {
      setIsLoading(true);
      try {
        // One side may be empty when the sheet was attached (no oldSheet)
        // or cleared (no newSheet). Treat that side as empty content
        // rather than fetching by an empty resource name.
        const fetchStatement = async (sheetName: string) => {
          if (!sheetName) return "";
          const sheet = await useAppStore
            .getState()
            .getOrFetchSheetByName(sheetName);
          return sheet ? getSheetStatement(sheet) : "";
        };
        const [oldValue, newValue] = await Promise.all([
          fetchStatement(oldSheet),
          fetchStatement(newSheet),
        ]);
        if (!canceled) {
          setOldStatement(oldValue);
          setNewStatement(newValue);
        }
      } finally {
        if (!canceled) {
          setIsLoading(false);
        }
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [newSheet, oldSheet, open]);

  return (
    <>
      <button
        className="inline-flex items-center text-accent hover:underline"
        onClick={() => setOpen(true)}
        type="button"
      >
        {t("common.view-details")}
      </button>
      <Dialog onOpenChange={setOpen} open={open}>
        <DialogContent className="max-w-none border-0 p-0 sm:w-[calc(100vw-9rem)] 2xl:max-w-none">
          <div className="px-6 pt-5">
            <DialogTitle>{t("common.detail")}</DialogTitle>
          </div>
          <div className="px-6 pb-6 pt-2">
            {(() => {
              // Fill the dialog instead of letting the editor shrink to
              // content height — short SQL diffs were leaving most of the
              // dialog blank. Cap at a sensible max so very tall viewports
              // don't get an awkwardly-stretched editor.
              const height = Math.min(
                900,
                Math.max(400, window.innerHeight - 240)
              );
              if (isLoading) {
                return (
                  <div
                    className="flex w-full items-center justify-center rounded-md border"
                    style={{ height }}
                  >
                    <Loader2 className="h-6 w-6 animate-spin text-control-light" />
                  </div>
                );
              }
              return (
                <ReadonlyDiffMonaco
                  className="w-full"
                  max={height}
                  min={height}
                  modified={newStatement}
                  original={oldStatement}
                />
              );
            })()}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
