import {
  Check,
  ChevronDown,
  ChevronUp,
  CircleCheck,
  Loader2,
  RotateCcw,
} from "lucide-react";
import {
  type ReactNode,
  type SetStateAction,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { MonacoViewZoneRevealContext } from "@/components/monaco/MonacoViewZone";
import { UserAvatar } from "@/components/UserAvatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent } from "@/components/ui/popover";
import { useCurrentUser, useUserByIdentifier } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ThreadComment } from "./ThreadComment";
import type { CommentThread } from "./threadModel";
import {
  canReplyToThread,
  canSettleThread,
  useThreadActions,
} from "./useThreadActions";

// A long thread keeps its tail visible and folds the middle behind a count.
const VISIBLE_TAIL_REPLIES = 2;
const FOLD_REPLIES_ABOVE = 4;
type ReplyAction = "reply" | "resolve" | "reopen";

// The same thread presentation for the statement editor and the Review
// Activity timeline: root and replies in one card, Reply and Resolve or Reopen
// in the footer. Resolved threads collapse to a summary row.
export function CommentThreadCard({
  className,
  collapsible = true,
  context,
  issueName,
  label,
  onClose,
  onThreadStateChanged,
  replyDraft: savedReplyDraft,
  onReplyDraftChange,
  project,
  thread,
}: {
  className?: string;
  // Whether a resolved thread may collapse to its summary row.
  collapsible?: boolean;
  // Full-width statement context above the root author and discussion.
  context?: ReactNode;
  issueName: string;
  // Extra header text after the timestamp (the anchored line range).
  label?: ReactNode;
  // Editor placement: closes the expanded thread back to its marker.
  onClose?: () => void;
  // Fired only after this card successfully changes state, never on refresh.
  onThreadStateChanged?: (resolved: boolean) => void;
  replyDraft?: string;
  onReplyDraftChange?: (draft: SetStateAction<string>) => void;
  project: Project | undefined;
  thread: CommentThread;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const actions = useThreadActions(issueName);
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
  const [replyAction, setReplyAction] = useState<ReplyAction>("reply");
  const [replyAttempted, setReplyAttempted] = useState(false);
  const replyButtonRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (replyDraft.trim() || !replyOpen) setReplyAttempted(false);
  }, [replyDraft, replyOpen]);
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

  const allowReply = canReplyToThread(project);
  const allowSettle = canSettleThread(thread.root, currentUser.email, project);
  const selectedAction =
    allowSettle && replyAction === (thread.resolved ? "reopen" : "resolve")
      ? replyAction
      : "reply";
  const replyActionLabel =
    selectedAction === "resolve"
      ? t("plan.review.thread.resolve-with-comment")
      : selectedAction === "reopen"
        ? t("plan.review.thread.reopen-with-comment")
        : t("plan.review.thread.reply");
  useEffect(() => {
    setReplyAction("reply");
  }, [thread.resolved, allowSettle]);

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

  const submitReply = async (action: ReplyAction = "reply") => {
    if (busy || submissionPending.current) return;
    if (!allowReply || (action !== "reply" && !allowSettle)) return;
    if (!replyDraft.trim()) {
      setReplyAttempted(true);
      return;
    }
    submissionPending.current = true;
    setSubmitting(true);
    const submittedDraft = replyDraft;
    try {
      const created = await actions.reply(thread.root.name, submittedDraft);
      if (!created) return;
      setReplyAttempted(false);
      setReplyAction("reply");
      // Once published, this draft must never be sent again when retrying a
      // failed status update. Preserve any new text typed while publishing.
      setReplyDraft((current) => (current === submittedDraft ? "" : current));
      if (mounted.current && draftRef.current === submittedDraft) {
        setReplyOpen(false);
      }
      if (action === "reply") return;
      const failureTitle = t("plan.review.thread.reply-posted-status-failed");
      const resolved = action === "resolve";
      const updated = await (resolved
        ? actions.resolve(thread.root.name, failureTitle)
        : actions.reopen(thread.root.name, failureTitle));
      if (updated && mounted.current) stateChanged.current?.(resolved);
    } finally {
      submissionPending.current = false;
      setSubmitting(false);
    }
  };

  if (collapsed) {
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
      {context && (
        <div className="min-w-0 overflow-hidden rounded-t-sm border-b border-block-border">
          {context}
        </div>
      )}
      <ThreadComment
        actions={
          <>
            {thread.resolved && collapsible && (
              <Button
                aria-label={t("common.collapse")}
                onClick={() => setCollapsed(true)}
                size="xs"
                appearance="secondary"
              >
                <ChevronUp className="size-3.5" />
              </Button>
            )}
            {onClose && (
              <Button
                aria-label={t("common.close")}
                onClick={onClose}
                size="xs"
                appearance="secondary"
              >
                <ChevronUp className="size-3.5" />
              </Button>
            )}
          </>
        }
        comment={thread.root}
        onEdit={actions.edit}
        project={project}
        status={
          <>
            {label}
            {thread.resolved && (
              <Badge className="gap-x-1 px-2 text-xs" variant="success">
                <CircleCheck className="size-3" />
                {t("plan.review.thread.resolved")}
              </Badge>
            )}
          </>
        }
      />
      {hiddenCount > 0 && (
        <div className="flex justify-center border-t border-block-border bg-control-bg/50 py-1">
          <Button
            onClick={() => setShowAllReplies(true)}
            size="xs"
            appearance="link"
          >
            <ChevronDown className="size-3.5" />
            {t("plan.review.thread.n-replies-hidden", { count: hiddenCount })}
            <span className="text-control-placeholder">·</span>
            {t("plan.review.activity.show-all")}
          </Button>
        </div>
      )}
      {visibleReplies.map((reply) => (
        <ThreadComment
          className="border-t border-block-border"
          comment={reply}
          key={reply.name}
          onEdit={actions.edit}
          project={project}
        />
      ))}
      {(allowReply || allowSettle) && (
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
                    onSubmit={() => void submitReply(selectedAction)}
                    placeholder={t("plan.review.thread.reply-placeholder")}
                  />
                  <div className="flex flex-wrap items-center justify-end gap-2">
                    <Button
                      onClick={() => setReplyOpen(false)}
                      size="sm"
                      appearance="secondary"
                    >
                      {t("common.cancel")}
                    </Button>
                    <div className="inline-flex items-center">
                      <Button
                        ref={replyButtonRef}
                        size="sm"
                        className="rounded-r-none"
                        disabled={busy}
                        onClick={() => void submitReply(selectedAction)}
                      >
                        {busy && <Loader2 className="size-4 animate-spin" />}
                        {replyActionLabel}
                      </Button>
                      <Popover
                        open={replyAttempted && !replyDraft.trim()}
                        onOpenChange={(open) => {
                          if (!open) setReplyAttempted(false);
                        }}
                      >
                        <PopoverContent
                          anchor={replyButtonRef}
                          side="top"
                          align="end"
                          initialFocus={false}
                          className="max-w-64 px-3 py-2 text-xs"
                        >
                          <p role="alert">
                            {t("plan.review.thread.comment-required")}
                          </p>
                        </PopoverContent>
                      </Popover>
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          aria-label={t("common.actions")}
                          disabled={busy}
                          render={
                            <Button
                              size="sm"
                              className="rounded-l-none border-l border-accent-text/20"
                            />
                          }
                        >
                          <ChevronDown className="size-4" />
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            disabled={busy}
                            onClick={() => setReplyAction("reply")}
                          >
                            {t("plan.review.thread.reply")}
                          </DropdownMenuItem>
                          {allowSettle && (
                            <DropdownMenuItem
                              disabled={busy}
                              onClick={() =>
                                setReplyAction(
                                  thread.resolved ? "reopen" : "resolve"
                                )
                              }
                            >
                              {thread.resolved
                                ? t("plan.review.thread.reopen-with-comment")
                                : t("plan.review.thread.resolve-with-comment")}
                            </DropdownMenuItem>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
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

function ThreadSummary({
  thread,
  onExpand,
}: {
  thread: CommentThread;
  onExpand: () => void;
}) {
  const { t } = useTranslation();
  const creator =
    useUserByIdentifier(thread.root.creator) ??
    unknownUser(thread.root.creator);
  const summary = thread.root.comment.split("\n")[0] ?? "";
  return (
    <Button
      appearance="secondary"
      className="w-full min-w-0 justify-start text-left font-normal"
      onClick={onExpand}
      size="md"
    >
      <UserAvatar
        colorSeed={creator.email}
        size="xs"
        title={creator.title || creator.email}
      />
      <span className="max-w-24 truncate font-medium text-main">
        {creator.title || creator.email}
      </span>
      <span className="min-w-0 flex-1 truncate text-control" title={summary}>
        {summary}
      </span>
      {thread.replies.length > 0 && (
        <span className="shrink-0 text-xs text-control-light">
          {t("plan.review.thread.n-replies", { count: thread.replies.length })}
        </span>
      )}
      {thread.root.createTime && (
        <HumanizeTs
          className="hidden shrink-0 text-xs text-control-light sm:inline"
          ts={getTimeForPbTimestampProtoEs(thread.root.createTime, 0) / 1000}
        />
      )}
      <Badge className="shrink-0 gap-x-1 px-2 text-xs" variant="success">
        <CircleCheck className="size-3" />
        {t("plan.review.thread.resolved")}
      </Badge>
      <ChevronDown className="size-4 shrink-0 text-control-light" />
    </Button>
  );
}
