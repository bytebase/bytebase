import {
  Check,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Loader2,
  RotateCcw,
} from "lucide-react";
import {
  type MouseEvent,
  type ReactNode,
  type SetStateAction,
  useContext,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentBadge } from "@/components/EnvironmentLabel";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { MonacoViewZoneRevealContext } from "@/components/monaco/MonacoViewZone";
import { UserAvatar } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import { Tooltip } from "@/components/ui/tooltip";
import { useCurrentUser, usePlanFeature } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { AgentDiagnosticHelpButton } from "@/modules/agent/components/AgentDiagnosticHelpButton";
import { normalizeAgentDiagnosticCapability } from "@/modules/agent/logic/diagnosticCapability";
import type { AgentDiagnosticContext } from "@/modules/agent/logic/diagnosticHelp";
import { useAppStore } from "@/stores/app";
import {
  type Issue,
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ThreadState,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import "./CommentThreadCard.css";
import {
  productionEnvironmentsOf,
  requestMissingDatabases,
} from "./productionTargets";
import { ReviewPriorityBadge } from "./ReviewResult";
import { ThreadComment } from "./ThreadComment";
import { ThreadStateBadge } from "./ThreadStateBadge";
import {
  type AnchorState,
  type CommentThread,
  type LineRange,
  lineRangeLabel,
} from "./threadModel";
import {
  canReplyToThread,
  canSettleThread,
  useThreadActions,
} from "./useThreadActions";

// A long thread keeps its tail visible and folds the middle behind a count.
const VISIBLE_TAIL_REPLIES = 2;
const FOLD_REPLIES_ABOVE = 4;

// The same thread presentation for the statement editor and the Review
// Activity timeline: root and replies in one card, Reply and Resolve or Reopen
// in the footer. The reply composer carries a checkbox that mirrors the thread
// state and, when toggled, settles the thread after the reply is posted.
// Every card starts with a title bar that holds the fold control and the
// state, and folding leaves the bar alone.
export function CommentThreadCard({
  bar,
  className,
  excerpt,
  issue,
  onThreadStateChanged,
  replyDraft: savedReplyDraft,
  onReplyDraftChange,
  project,
  spec,
  thread,
}: {
  bar: ThreadBarProps;
  className?: string;
  // The recorded SQL the thread is on, above the first comment.
  excerpt?: ReactNode;
  issue: Issue;
  // Fired only after this card successfully changes state, never on refresh.
  onThreadStateChanged?: (resolved: boolean) => void;
  replyDraft?: string;
  onReplyDraftChange?: (draft: SetStateAction<string>) => void;
  project: Project | undefined;
  // The plan spec the thread sits on, when the owner knows it.
  spec?: Plan_Spec;
  thread: CommentThread;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const actions = useThreadActions(issue.name);
  const [ownFolded, setOwnFolded] = useState(thread.resolved);
  const [showAllReplies, setShowAllReplies] = useState(
    () => thread.replies.length <= FOLD_REPLIES_ABOVE
  );
  const [replyOpen, setReplyOpen] = useState(Boolean(savedReplyDraft));
  const replyFooterRef = useRef<HTMLDivElement>(null);
  const revealInEditor = useContext(MonacoViewZoneRevealContext);
  useEffect(() => {
    if (!replyOpen || !replyFooterRef.current) return;
    if (revealInEditor) revealInEditor(replyFooterRef.current);
    else replyFooterRef.current.scrollIntoView({ block: "nearest" });
  }, [replyOpen, revealInEditor]);
  const [localReplyDraft, setLocalReplyDraft] = useState("");
  const replyDraft = savedReplyDraft ?? localReplyDraft;
  const setReplyDraft = onReplyDraftChange ?? setLocalReplyDraft;
  const [submitting, setSubmitting] = useState(false);
  const [settleWithReply, setSettleWithReply] = useState(thread.resolved);
  const resolveCheckboxId = useId();
  const submissionPending = useRef(false);
  const busy = actions.pending || submitting;
  const draftRef = useRef(replyDraft);
  useLayoutEffect(() => {
    draftRef.current = replyDraft;
  }, [replyDraft]);
  const mounted = useRef(false);
  const stateChanged = useRef(onThreadStateChanged);
  useLayoutEffect(() => {
    mounted.current = true;
    stateChanged.current = onThreadStateChanged;
    return () => {
      mounted.current = false;
    };
  }, [onThreadStateChanged]);

  const settleThread = async () => {
    if (busy || submissionPending.current) return;
    const resolved = !thread.resolved;
    const updated = await (resolved
      ? actions.resolve(thread.root.name)
      : actions.reopen(thread.root.name));
    if (updated && mounted.current) stateChanged.current?.(resolved);
  };

  // A card that keeps its own fold state folds when the thread becomes
  // resolved and unfolds when it is reopened.
  useEffect(() => {
    setOwnFolded(thread.resolved);
  }, [thread.resolved]);
  const folded = bar.folded ?? ownFolded;
  const toggleFold =
    bar.onToggleFold ?? (() => setOwnFolded((current) => !current));

  const allowReply = canReplyToThread(project, issue);
  const allowSettle = canSettleThread(project);
  const diagnostic = findingDiagnosticContext(thread, issue, project, spec);
  // The reply composer's checkbox names the action for the thread's current
  // state and, when checked, performs it after the reply is posted:
  //   open thread     -> "Resolve thread", unchecked; a reply keeps it open.
  //   resolved thread -> "Reopen thread", checked; a reply reopens it.
  // The resolved default is deliberate: replying to a resolved thread is
  // rare and nearly always means the matter is not settled, so reopening is
  // the expected outcome and keeping it resolved is the opt-out. A state
  // change from elsewhere or a fresh composer resets the box to its default.
  useEffect(() => {
    setSettleWithReply(thread.resolved);
  }, [thread.resolved, replyOpen]);
  const settleAfterReply = allowSettle && settleWithReply;

  const { hiddenCount, visibleReplies } = useMemo(() => {
    const replies = thread.replies;
    if (showAllReplies || replies.length <= FOLD_REPLIES_ABOVE) {
      return { hiddenCount: 0, visibleReplies: replies };
    }
    return {
      hiddenCount: replies.length - VISIBLE_TAIL_REPLIES,
      visibleReplies: replies.slice(replies.length - VISIBLE_TAIL_REPLIES),
    };
  }, [showAllReplies, thread.replies]);

  const submitReply = async () => {
    if (busy || submissionPending.current) return;
    if (!allowReply) return;
    if (!replyDraft.trim()) return;
    submissionPending.current = true;
    setSubmitting(true);
    const submittedDraft = replyDraft;
    try {
      const created = await actions.reply(thread.root.name, submittedDraft);
      if (!created) return;
      // Once published, this draft must never be sent again when retrying a
      // failed status update. Preserve any new text typed while publishing.
      setReplyDraft((current) => (current === submittedDraft ? "" : current));
      if (mounted.current && draftRef.current === submittedDraft) {
        setReplyOpen(false);
      }
      if (!settleAfterReply) return;
      const failureTitle = t("plan.review.thread.reply-posted-status-failed");
      const resolved = !thread.resolved;
      const updated = await (resolved
        ? actions.resolve(thread.root.name, failureTitle)
        : actions.reopen(thread.root.name, failureTitle));
      if (updated && mounted.current) stateChanged.current?.(resolved);
    } finally {
      submissionPending.current = false;
      setSubmitting(false);
    }
  };

  return (
    <div
      className={cn(
        "@container/thread flex min-w-0 flex-col rounded-lg border border-control-border bg-background",
        className
      )}
      data-testid="comment-thread"
      data-thread-state={thread.resolved ? "resolved" : "open"}
      id={thread.root.name}
    >
      <ThreadBar
        {...bar}
        folded={folded}
        markers={<ThreadMarkers thread={thread} />}
        onToggleFold={toggleFold}
        resolved={thread.resolved}
      />
      {!folded && excerpt && (
        <div className="min-w-0 overflow-hidden border-b border-block-border">
          {excerpt}
        </div>
      )}
      {!folded && (
        <div className="relative">
          <ThreadComment
            actions={
              diagnostic && <AgentDiagnosticHelpButton context={diagnostic} />
            }
            comment={thread.root}
            onEdit={actions.edit}
            project={project}
            spec={spec}
          />
          {(hiddenCount > 0 || visibleReplies.length > 0) && (
            <div
              className="bb-thread-replies ml-7"
              data-testid="thread-replies"
            >
              {/* Joins the replies to the first comment, under its avatar. */}
              <Separator
                className="absolute top-9 bottom-2 left-4.5 h-auto sm:left-5.5"
                data-testid="thread-replies-rail"
                orientation="vertical"
              />
              {hiddenCount > 0 && (
                <div className="flex justify-center py-1">
                  <Button
                    onClick={() => setShowAllReplies(true)}
                    size="xs"
                    appearance="link"
                  >
                    <ChevronDown className="size-3.5" />
                    {t("plan.review.thread.n-replies-hidden", {
                      count: hiddenCount,
                    })}
                    <span className="text-control-placeholder">·</span>
                    {t("plan.review.activity.show-all")}
                  </Button>
                </div>
              )}
              {visibleReplies.map((reply) => (
                <ThreadComment
                  comment={reply}
                  key={reply.name}
                  onEdit={actions.edit}
                  project={project}
                />
              ))}
            </div>
          )}
        </div>
      )}
      {!folded && (allowReply || allowSettle) && (
        <div
          className="flex flex-wrap items-center gap-2 rounded-b-lg border-t border-block-border bg-background px-2 py-2 sm:px-3"
          ref={replyFooterRef}
          data-testid="thread-reply-footer"
        >
          {allowReply && (
            <>
              {replyOpen && (
                <div
                  className="hidden h-6 shrink-0 self-start items-center @xs/thread:flex"
                  data-testid="reply-composer-avatar"
                >
                  <UserAvatar
                    colorSeed={currentUser.email}
                    size="xs"
                    title={currentUser.title || currentUser.email}
                  />
                </div>
              )}
              {replyOpen ? (
                <div className="flex min-w-0 flex-1 flex-col gap-y-2">
                  <MarkdownEditor
                    autoFocus
                    compact
                    content={replyDraft}
                    onChange={setReplyDraft}
                    onSubmit={() => void submitReply()}
                    placeholder={t("plan.review.thread.reply-placeholder")}
                  />
                  <div className="flex flex-wrap items-center gap-2">
                    {allowSettle && (
                      <div className="flex items-center gap-x-2">
                        <Checkbox
                          id={resolveCheckboxId}
                          checked={settleWithReply}
                          disabled={busy}
                          onCheckedChange={setSettleWithReply}
                        />
                        <label
                          htmlFor={resolveCheckboxId}
                          className="cursor-pointer text-sm"
                        >
                          {thread.resolved
                            ? t("plan.review.thread.reopen-thread")
                            : t("plan.review.thread.resolve-thread")}
                        </label>
                      </div>
                    )}
                    <Button
                      className="ml-auto"
                      onClick={() => setReplyOpen(false)}
                      size="sm"
                      appearance="secondary"
                    >
                      {t("common.cancel")}
                    </Button>
                    <Button
                      size="sm"
                      disabled={busy || !replyDraft.trim()}
                      onClick={() => void submitReply()}
                    >
                      {busy && <Loader2 className="size-4 animate-spin" />}
                      {t("plan.review.thread.reply")}
                    </Button>
                  </div>
                </div>
              ) : (
                <Button
                  appearance="outline"
                  className="min-w-20 flex-1 justify-start bg-background text-control-placeholder"
                  onClick={() => setReplyOpen(true)}
                  size="sm"
                  type="button"
                >
                  {t("plan.review.thread.reply-placeholder")}
                </Button>
              )}
            </>
          )}
          {!allowReply && <span className="flex-1" />}
          {allowSettle && !replyOpen && (
            <Button
              appearance="outline"
              className="ml-auto shrink-0 bg-background"
              disabled={busy}
              onClick={() => void settleThread()}
              size="sm"
            >
              {thread.resolved ? (
                <RotateCcw className="size-3.5" />
              ) : (
                <Check className="size-3.5" />
              )}
              {thread.resolved
                ? t("common.reopen")
                : t("plan.review.thread.resolve")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

export type ThreadBarProps = {
  // Where the card is drawn.
  place: "changes" | "activity";
  // Names what the thread is on.
  title: ReactNode;
  // The fold control's name: the title's text, then the lines when the bar
  // shows them apart from the title.
  foldLabel: string;
  // Activity only: the lines on the right, the placement state, and the link.
  lines?: LineRange;
  anchorState?: AnchorState;
  stateTooltip?: string;
  onViewInStatement?: () => void;
  // The Changes section keeps the fold state; Activity leaves both unset and
  // the card keeps it.
  folded?: boolean;
  onToggleFold?: () => void;
};

// A click inside the bar folds unless it lands on a control, which keeps its
// own meaning, or ends a drag that selected text.
const shouldFoldFromBarClick = (event: MouseEvent<HTMLElement>): boolean => {
  const target = event.target as Element;
  if (target.closest("button, a, input, [role='button']")) return false;
  return !window.getSelection()?.toString();
};

// What a folded thread must still say about how urgent it is: the badge of a
// P0 or P1 finding, which needs someone to act, and the badge of each
// production environment a finding applies to. A person's thread shows
// neither.
function ThreadMarkers({ thread }: { thread: CommentThread }) {
  const review = thread.root.reviewMetadata;
  const targets = review?.targets ?? EMPTY_TARGETS;
  const databasesByName = useAppStore((state) => state.databasesByName);
  const environmentList = useAppStore((state) => state.environmentList);
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const targetKey = targets.join("\n");
  useEffect(() => {
    const names = targetKey.split("\n").filter(isValidDatabaseName);
    if (names.length === 0) return;
    const store = useAppStore.getState();
    void requestMissingDatabases(
      names,
      (name) => Boolean(useAppStore.getState().databasesByName[name]),
      (missing) => store.batchGetOrFetchDatabases(missing, true)
    ).catch(() => {
      // Without the databases the bar just shows no environment badge.
    });
  }, [targetKey]);
  if (!review) return null;
  const urgent =
    review.priority === IssueComment_ReviewMetadata_Priority.P0 ||
    review.priority === IssueComment_ReviewMetadata_Priority.P1;
  const productionEnvironments = productionEnvironmentsOf(
    targets,
    { databasesByName, environmentList },
    hasEnvTierFeature
  );
  return (
    <>
      {urgent && <ReviewPriorityBadge priority={review.priority} />}
      {productionEnvironments.map((environment) => (
        // The environment's own badge, unchanged; the bar only caps its width.
        <span
          className="inline-flex max-w-40 min-w-0 shrink-0"
          data-testid="thread-production"
          key={environment.name}
          title={environment.title}
        >
          <EnvironmentBadge
            className="max-w-full"
            environment={environment}
            hasEnvTierFeature={hasEnvTierFeature}
          />
        </span>
      ))}
    </>
  );
}

const EMPTY_TARGETS: readonly string[] = [];

// The bar's words and icons are text-control: text-control-light falls under
// the 4.5:1 contrast minimum.
function ThreadBar({
  anchorState,
  place,
  folded,
  foldLabel,
  lines,
  markers,
  onToggleFold,
  onViewInStatement,
  resolved,
  stateTooltip,
  title,
}: Omit<ThreadBarProps, "folded" | "onToggleFold"> & {
  folded: boolean;
  markers: ReactNode;
  onToggleFold: () => void;
  resolved: boolean;
}) {
  const { t } = useTranslation();
  const labels = (
    <>
      {resolved && <ThreadStateBadge state="resolved" />}
      {anchorState === "OUTDATED" &&
        (stateTooltip ? (
          <Tooltip
            content={stateTooltip}
            render={<span className="inline-flex" tabIndex={0} />}
          >
            <ThreadStateBadge state="outdated" />
          </Tooltip>
        ) : (
          <ThreadStateBadge state="outdated" />
        ))}
      {anchorState === "UNAVAILABLE" && (
        <ThreadStateBadge state="unavailable" />
      )}
    </>
  );
  return (
    // A click on the bar folds too: anywhere in the Changes section, and on the
    // title, markers or state labels in Activity, whose right end holds the
    // lines and a link. It is a pointer shortcut: the fold control stays the
    // one focusable control, and the keyboard's path.
    <div
      className={cn(
        "flex min-h-8 min-w-0 flex-wrap items-center gap-x-2 gap-y-1 bg-background px-2 py-1 text-xs text-control",
        place === "changes" && "cursor-pointer",
        folded ? "rounded-lg" : "rounded-t-lg border-b border-block-border"
      )}
      data-anchor-state={anchorState}
      data-testid="thread-bar"
      onClick={(event) => {
        if (!shouldFoldFromBarClick(event)) return;
        const target = event.target as Element;
        if (place === "activity" && !target.closest("[data-bar-lead]")) return;
        onToggleFold();
      }}
    >
      <Button
        appearance="secondary"
        aria-expanded={!folded}
        aria-label={foldLabel}
        className="shrink-0"
        onClick={onToggleFold}
        size="xs"
      >
        {folded ? (
          <ChevronRight className="size-3.5" />
        ) : (
          <ChevronDown className="size-3.5" />
        )}
      </Button>
      {/* The title is as wide as its text, at least 64px, so the labels stay
          beside it; it shrinks to 64px before the bar wraps. */}
      <span
        className={cn("contents", place === "activity" && "cursor-pointer")}
        data-bar-lead
        data-testid="thread-bar-lead"
      >
        {typeof title === "string" ? (
          <span
            className="min-w-16 max-w-max flex-1 basis-0 truncate"
            title={title}
          >
            {title}
          </span>
        ) : (
          <span className="flex min-w-16 max-w-max flex-1 basis-0">
            {title}
          </span>
        )}
        {markers}
        {labels}
      </span>
      {(lines || onViewInStatement) && (
        <span
          className="ml-auto flex shrink-0 items-center gap-x-2"
          data-testid="thread-bar-end"
        >
          {lines && (
            <span data-testid="thread-bar-lines">
              {lineRangeLabel(t, lines)}
            </span>
          )}
          {onViewInStatement && (
            <Button
              appearance="link"
              className="h-auto p-0"
              onClick={onViewInStatement}
              size="xs"
            >
              {t("plan.review.thread.anchor.view-in-statement")}
              <ExternalLink className="size-3" />
            </Button>
          )}
        </span>
      )}
    </div>
  );
}

function findingDiagnosticContext(
  thread: CommentThread,
  issue: Issue,
  project: Project | undefined,
  spec: Plan_Spec | undefined
): AgentDiagnosticContext | undefined {
  const { root } = thread;
  const review = root.reviewMetadata;
  if (
    !review ||
    thread.resolved ||
    root.threadState !== IssueComment_ThreadState.OPEN
  )
    return undefined;
  const projectName = issue.name.split("/").slice(0, 2).join("/");
  const issueScope = normalizeAgentDiagnosticCapability({
    kind: "diagnostic",
    resource: {
      kind: "issue",
      name: issue.name,
      project: projectName,
      label: issue.name,
    },
  });
  if (
    !issueScope ||
    (project && project.name !== projectName) ||
    !root.name.startsWith(`${issue.name}/issueComments/`) ||
    !/^[a-zA-Z0-9-]+$/.test(
      root.name.slice(`${issue.name}/issueComments/`.length)
    ) ||
    (spec && root.statementAnchor && spec.id !== root.statementAnchor.spec)
  )
    return undefined;
  const scope = issue.plan
    ? normalizeAgentDiagnosticCapability({
        kind: "diagnostic",
        resource: {
          kind: "plan",
          name: issue.plan,
          project: projectName,
          label: issue.plan,
        },
      })
    : issueScope;
  if (!scope) return undefined;
  const anchor = root.statementAnchor;
  const facts = [
    { label: "Issue", value: issue.name },
    {
      label: "Review run",
      value: `${issue.name}/reviewRun, attempt ${review.attempt}`,
    },
    {
      label: "Priority",
      value: IssueComment_ReviewMetadata_Priority[review.priority],
    },
    { label: "Targets", value: review.targets.join(", ") },
    { label: "Superseded", value: String(review.superseded) },
    {
      label: "Evidence",
      value:
        "Partial: original SQL snapshot unavailable; use the original revision identifiers, not current SQL.",
    },
  ];
  if (review.ruleType !== ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED) {
    facts.push({
      label: "Rule",
      value: [ReviewRuleType[review.ruleType], ...review.codes].join(": "),
    });
  }
  if (anchor) {
    facts.push(
      { label: "Spec", value: anchor.spec },
      { label: "Original sheet SHA256", value: anchor.sheetSha256 }
    );
    if (anchor.startPosition && anchor.endPosition)
      facts.push({
        label: "Original SQL range",
        value: `${anchor.startPosition.line}:${anchor.startPosition.column}-${anchor.endPosition.line}:${anchor.endPosition.column}`,
      });
  }
  return {
    resource: scope.resource,
    kind: "review-finding",
    sourceId: root.name,
    summary: review.title || "Automated review finding",
    detail: root.comment,
    ...(anchor?.spec.trim() && /^[a-f0-9]{64}$/i.test(anchor.sheetSha256)
      ? {
          sqlFixSource: {
            specId: anchor.spec,
            sha256: anchor.sheetSha256,
            targets: [...review.targets],
          },
        }
      : {}),
    facts,
  };
}
