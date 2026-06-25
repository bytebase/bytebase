import { FileText, Loader2, Pencil, Send } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import {
  ActivityRowShell,
  CommentCreator,
  CommentIconBadge,
  canEditIssueComment,
  IssueCommentRow,
} from "@/react/components/issue-activity/IssueCommentActivity";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { diffPlanSpecsForEvent } from "@/react/lib/plan/diffPlanSpecs";
import { useAppStore } from "@/react/stores/app";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";
import { pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { consolidateConsecutive } from "./consolidateTimeline";
import { foldTimeline } from "./foldTimeline";
import { ReviewCommentComposer } from "./ReviewCommentComposer";
import {
  buildTimelineEntries,
  type TimelineEntry,
  type TimelineSource,
} from "./timelineEvents";

export function ReviewActivityTimeline({
  comments,
  issue,
  plan,
}: {
  comments: IssueComment[];
  issue: Issue;
  plan: Plan;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const project = useProjectByName(`${projectNamePrefix}${page.projectId}`);
  const [expanded, setExpanded] = useState(false);

  const entries = useMemo(
    () =>
      buildTimelineEntries({
        planCreator: plan.creator,
        planCreateTime: plan.createTime,
        issueCreator: issue.creator,
        issueCreateTime: issue.createTime,
        comments,
      }),
    [comments, issue.createTime, issue.creator, plan.createTime, plan.creator]
  );
  const items = useMemo(() => {
    const renderable = entries.filter(isRenderableEntry);
    const consolidated = consolidateConsecutive(renderable);
    return foldTimeline(consolidated, expanded);
  }, [entries, expanded]);
  // Permission-gated only — comments are allowed regardless of issue status
  // (the backend and the issue-detail list allow commenting on done issues).
  const allowComment = Boolean(
    project && hasProjectPermissionV2(project, "bb.issueComments.create")
  );

  return (
    <div className="flex flex-col gap-y-2 px-4 pb-3 pt-4">
      <h4 className="text-sm font-medium text-main">
        {t("plan.review.activity.self")}
      </h4>
      <ul className="flex flex-col">
        {items.map((item, i) => {
          if (item.type === "fold") {
            return (
              <TornSeparator
                count={item.count}
                key={`fold-${i}`}
                onShowAll={() => setExpanded(true)}
              />
            );
          }
          const isLast = !items
            .slice(i + 1)
            .some((next) => next.type !== "fold");
          return (
            <ActivityRow
              entry={item.entry}
              issue={issue}
              isLast={isLast}
              key={item.entry.id}
              plan={plan}
            />
          );
        })}
      </ul>
      {allowComment && <ReviewCommentComposer issueName={issue.name} />}
    </div>
  );
}

// A plan update whose specs are proto-equal produces no sentence; drop it
// entirely so the timeline doesn't show a verbless row (and so the connector
// line's "last row" math stays correct).
function isRenderableEntry(entry: TimelineEntry): boolean {
  if (entry.source.type !== "comment") return true;
  const comment = entry.source.comment;
  if (
    getIssueCommentType(comment) === IssueCommentType.PLAN_UPDATE &&
    comment.event.case === "planUpdate"
  ) {
    return diffPlanSpecsForEvent(comment.event.value).length > 0;
  }
  return true;
}

// Torn-paper zigzag edge (a triangle-wave stroke) tiled horizontally on each
// side of the fold marker. Encoded as an inline SVG data URI because no Tailwind
// utility produces a sawtooth line.
const TORN_EDGE_STYLE = {
  backgroundImage:
    "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='8'%3E%3Cpath d='M0 6 L12 2 L24 6' fill='none' stroke='%23cfd8e6' stroke-width='1'/%3E%3C/svg%3E\")",
  backgroundRepeat: "repeat-x",
  backgroundPosition: "center",
} as const;

function TornSeparator({
  count,
  onShowAll,
}: {
  count: number;
  onShowAll: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-x-2 py-2">
      <div aria-hidden="true" className="h-2 flex-1" style={TORN_EDGE_STYLE} />
      <button
        className="group shrink-0 bg-white px-2 text-xs text-control-light hover:text-control"
        onClick={onShowAll}
        type="button"
      >
        {t("plan.review.activity.n-hidden-events", { count })}
        <span className="mx-1 text-control-placeholder">·</span>
        <span className="text-accent group-hover:underline">
          {t("plan.review.activity.show-all")}
        </span>
      </button>
      <div aria-hidden="true" className="h-2 flex-1" style={TORN_EDGE_STYLE} />
    </div>
  );
}

function ActivityRow({
  entry,
  issue,
  isLast,
  plan,
}: {
  entry: TimelineEntry;
  issue: Issue;
  isLast: boolean;
  plan: Plan;
}) {
  const source = entry.source;
  if (source.type === "comment") {
    return (
      <ReviewCommentRow
        comment={source.comment}
        isLast={isLast}
        issue={issue}
        plan={plan}
        similarCount={entry.similarCount}
      />
    );
  }
  // Synthetic, review-only events (plan created / marked ready) have no backing
  // issue comment, so render the shared row shell with a local icon + header.
  return (
    <ActivityRowShell
      header={<SyntheticHeader source={source} />}
      icon={<SyntheticIcon type={source.type} />}
      isLast={isLast}
    />
  );
}

function SyntheticIcon({
  type,
}: {
  type: "plan-created" | "ready-for-review";
}) {
  return (
    <CommentIconBadge
      className="bg-control-bg text-control"
      icon={
        type === "plan-created" ? (
          <FileText className="h-4 w-4" />
        ) : (
          <Send className="h-4 w-4" />
        )
      }
    />
  );
}

function SyntheticHeader({
  source,
}: {
  source: Extract<
    TimelineSource,
    { type: "plan-created" | "ready-for-review" }
  >;
}) {
  const { t } = useTranslation();
  return (
    <>
      <ActorName principal={source.creator} />
      <span className="wrap-break-word min-w-0 text-gray-600">
        {source.type === "plan-created"
          ? t("plan.review.activity.created-this-plan")
          : t("plan.review.activity.marked-ready-for-review")}
      </span>
      {source.time && (
        <HumanizeTs
          className="text-gray-500"
          ts={getTimeForPbTimestampProtoEs(source.time, 0) / 1000}
        />
      )}
    </>
  );
}

function ActorName({ principal }: { principal: string }) {
  const user = useAppStore((state) => state.getUserByIdentifier(principal));
  return <CommentCreator creator={user ?? unknownUser(principal)} />;
}

// A comment-backed timeline row: delegates layout/icon/header to the shared
// IssueCommentRow and owns the inline edit affordance (matching the issue page).
function ReviewCommentRow({
  comment,
  isLast,
  issue,
  plan,
  similarCount,
}: {
  comment: IssueComment;
  isLast: boolean;
  issue: Issue;
  plan: Plan;
  similarCount?: number;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = useCurrentUser();
  const project = useProjectByName(`${projectNamePrefix}${page.projectId}`);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(comment.comment);
  const [saving, setSaving] = useState(false);

  const allowEdit = canEditIssueComment(comment, currentUser.email, project);

  const save = async () => {
    if (!editContent || editContent === comment.comment) {
      setIsEditing(false);
      return;
    }
    try {
      setSaving(true);
      await useAppStore.getState().updateIssueComment({
        issueCommentName: comment.name,
        comment: editContent,
      });
      setIsEditing(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setSaving(false);
    }
  };

  let body: ReactNode;
  if (comment.comment) {
    body = isEditing ? (
      <div>
        <MarkdownEditor
          content={editContent}
          onChange={setEditContent}
          onSubmit={() => void save()}
        />
        <div className="mt-2 flex items-center justify-end gap-x-2">
          <Button onClick={() => setIsEditing(false)} size="xs" variant="ghost">
            {t("common.cancel")}
          </Button>
          <Button
            disabled={
              saving ||
              editContent.trim().length === 0 ||
              editContent === comment.comment
            }
            onClick={() => void save()}
            size="xs"
          >
            {saving && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            {t("common.save")}
          </Button>
        </div>
      </div>
    ) : (
      <MarkdownEditor content={comment.comment} mode="preview" />
    );
  }

  const subjectSuffix =
    allowEdit && !isEditing ? (
      <Button
        onClick={() => {
          setEditContent(comment.comment);
          setIsEditing(true);
        }}
        size="xs"
        variant="ghost"
      >
        <Pencil className="h-3.5 w-3.5" />
      </Button>
    ) : undefined;

  return (
    <IssueCommentRow
      body={body}
      comment={comment}
      isLast={isLast}
      issue={issue}
      linkless
      plan={plan}
      similarCount={similarCount}
      subjectSuffix={subjectSuffix}
    />
  );
}
