import {
  Check,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Loader2,
  RotateCcw,
} from "lucide-react";
import {
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
import { HumanizeTs } from "@/components/HumanizeTs";
import { MarkdownEditor, markdownSummary } from "@/components/MarkdownEditor";
import { MonacoViewZoneRevealContext } from "@/components/monaco/MonacoViewZone";
import { UserAvatar } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useCurrentUser } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import "./CommentThreadCard.css";
import { ReviewPriorityBadge, useCommentAuthor } from "./ReviewResult";
import { ThreadComment } from "./ThreadComment";
import { ThreadStateBadge } from "./ThreadStateBadge";
import type { CommentThread } from "./threadModel";
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
// With a `bar`, the card starts with a title bar that holds the fold control
// and the state, and folding leaves the bar alone; the owner keeps the fold
// state. Without one, a resolved thread collapses to a summary row.
export function CommentThreadCard({
  bar,
  className,
  collapsible = true,
  renderContext,
  issue,
  onThreadStateChanged,
  replyDraft: savedReplyDraft,
  onReplyDraftChange,
  project,
  spec,
  thread,
}: {
  bar?: ThreadBarProps;
  className?: string;
  // Whether a resolved thread may collapse to its summary row.
  collapsible?: boolean;
  // Full-width statement context above the root author and discussion.
  renderContext?: (onCollapse?: () => void) => ReactNode;
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
  const [collapsed, setCollapsed] = useState(collapsible && thread.resolved);
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

  // Resolving collapses the thread; reopening expands it again.
  useEffect(() => {
    if (!collapsible) return;
    setCollapsed(thread.resolved);
  }, [collapsible, thread.resolved]);

  const allowReply = canReplyToThread(project, issue);
  const allowSettle = canSettleThread(project);
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

  if (collapsed && !bar) {
    return (
      <div
        className={cn(
          "rounded-sm border border-block-border bg-background",
          className
        )}
        data-testid="comment-thread"
        data-thread-state="resolved"
        id={thread.root.name}
      >
        <ThreadSummary thread={thread} onExpand={() => setCollapsed(false)} />
      </div>
    );
  }

  return (
    <div
      className={cn(
        "@container/thread flex min-w-0 flex-col rounded-sm border border-block-border bg-background",
        className
      )}
      data-testid="comment-thread"
      data-thread-state={thread.resolved ? "resolved" : "open"}
      id={thread.root.name}
    >
      {bar && <ThreadBar {...bar} resolved={thread.resolved} />}
      {!bar?.folded && renderContext && (
        <div className="min-w-0 overflow-hidden rounded-t-sm border-b border-block-border">
          {renderContext(
            thread.resolved && collapsible
              ? () => setCollapsed(true)
              : undefined
          )}
        </div>
      )}
      {!bar?.folded && (
        <ThreadComment
          actions={
            thread.resolved &&
            collapsible &&
            !bar &&
            !renderContext && (
              <Button
                aria-label={t("common.collapse")}
                onClick={() => setCollapsed(true)}
                size="xs"
                appearance="secondary"
              >
                <ChevronUp className="size-3.5" />
              </Button>
            )
          }
          comment={thread.root}
          onEdit={actions.edit}
          project={project}
          spec={spec}
        />
      )}
      {!bar?.folded && (hiddenCount > 0 || visibleReplies.length > 0) && (
        <div
          className="bb-thread-replies bg-control-bg/50"
          data-testid="thread-replies"
        >
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
      {!bar?.folded &&
        (allowReply || allowSettle || (!bar && thread.resolved)) && (
          <div
            className="flex flex-wrap items-center gap-2 border-t border-block-border bg-control-bg/50 px-2 py-2 sm:px-3"
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
            {!bar && thread.resolved && !replyOpen && <ThreadStateBadge />}
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

type ThreadBarProps = {
  // Names what the thread is on, and names the fold control.
  title: string;
  folded: boolean;
  onToggleFold: () => void;
};

// Every word and icon in the bar is text-control: text-control-light falls
// under the 4.5:1 contrast minimum on the bar's background.
function ThreadBar({
  folded,
  onToggleFold,
  resolved,
  title,
}: ThreadBarProps & { resolved: boolean }) {
  return (
    <div
      className={cn(
        "flex min-h-8 min-w-0 items-center gap-x-2 bg-control-bg px-2 text-xs text-control",
        folded ? "rounded-sm" : "rounded-t-sm border-b border-block-border"
      )}
      data-testid="thread-bar"
    >
      <Button
        appearance="secondary"
        aria-expanded={!folded}
        aria-label={title}
        className="shrink-0 hover:bg-control-bg-hover"
        onClick={onToggleFold}
        size="xs"
      >
        {folded ? (
          <ChevronRight className="size-3.5" />
        ) : (
          <ChevronDown className="size-3.5" />
        )}
      </Button>
      <span className="min-w-0 flex-1 truncate" title={title}>
        {title}
      </span>
      {resolved && <ThreadStateBadge />}
    </div>
  );
}

function ThreadSummary({
  thread,
  onExpand,
}: {
  thread: CommentThread;
  onExpand: () => void;
}) {
  const { t } = useTranslation();
  const author = useCommentAuthor(thread.root);
  const review = thread.root.reviewMetadata;
  // An AI result is summarized by its title, any other comment by its text.
  const summary = markdownSummary(review?.title || thread.root.comment);
  return (
    <Button
      appearance="secondary"
      className="w-full min-w-0 justify-start text-left font-normal"
      onClick={onExpand}
      size="md"
    >
      {author.avatar}
      <span className="max-w-24 truncate font-medium text-main">
        {author.name}
      </span>
      {review && <ReviewPriorityBadge priority={review.priority} />}
      <span className="min-w-0 flex-1 truncate text-control" title={summary}>
        {summary}
      </span>
      {thread.replies.length > 0 && (
        <span className="shrink-0 text-xs text-control-light">
          {t("plan.review.thread.n-replies", { count: thread.replies.length })}
        </span>
      )}
      <HumanizeTs
        className="hidden shrink-0 text-xs text-control-light sm:inline"
        tsMs={getTimeForPbTimestampProtoEs(thread.root.createTime)}
      />
      <ThreadStateBadge />
      <ChevronDown className="size-4 shrink-0 text-control-light" />
    </Button>
  );
}
