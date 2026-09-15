import { ChevronDown, CircleCheck, MessageSquare } from "lucide-react";
import { type ReactNode, type Ref, type SetStateAction, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { UserAvatar } from "@/components/UserAvatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useUserByIdentifier } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { CommentThreadCard } from "./CommentThreadCard";
import {
  compareByCreateTime,
  type EditorThread,
  formatLineRange,
  lineRangeLabel,
} from "./threadModel";

type ThreadGroups = { open: string[]; resolved: string[] };

// Capture group membership when a thread first appears in this open stack.
// Live status changes update the content, never the current reading order.
function includeNewThreads(
  groups: ThreadGroups,
  threads: EditorThread[]
): ThreadGroups {
  const known = new Set([...groups.open, ...groups.resolved]);
  const added = threads
    .filter((entry) => !known.has(entry.thread.root.name))
    .sort((a, b) => compareByCreateTime(a.thread.root, b.thread.root));
  if (added.length === 0) return groups;
  return {
    open: [
      ...groups.open,
      ...added
        .filter((entry) => !entry.thread.resolved)
        .map((entry) => entry.thread.root.name),
    ],
    resolved: [
      ...groups.resolved,
      ...added
        .filter((entry) => entry.thread.resolved)
        .map((entry) => entry.thread.root.name),
    ],
  };
}

// A mount is one viewing session. Reopening the gutter or switching lines
// remounts the stack, sorting Open before Resolved using the latest state.
export function ThreadStack({
  expandedRoots,
  expandedThreadRef,
  flashRoot,
  onFlashEnd,
  issueName,
  onCollapse,
  onExpand,
  replyDrafts,
  onReplyDraftChange,
  project,
  threads,
}: {
  expandedRoots: ReadonlySet<string>;
  expandedThreadRef?: Ref<HTMLDivElement>;
  // The thread the walker just landed on; its card flashes once.
  flashRoot?: string;
  onFlashEnd?: () => void;
  issueName: string;
  onCollapse: (rootName: string) => void;
  onExpand: (rootName: string) => void;
  replyDrafts: Readonly<Record<string, string>>;
  onReplyDraftChange: (rootName: string, draft: SetStateAction<string>) => void;
  project: Project | undefined;
  threads: EditorThread[];
}) {
  const { t } = useTranslation();
  const [groups, setGroups] = useState(() =>
    includeNewThreads({ open: [], resolved: [] }, threads)
  );
  const nextGroups = includeNewThreads(groups, threads);
  if (nextGroups !== groups) setGroups(nextGroups);
  const byName = new Map(
    threads.map((entry) => [entry.thread.root.name, entry])
  );
  const currentEntries = (names: string[]) =>
    names.flatMap((name) => {
      const entry = byName.get(name);
      return entry ? [entry] : [];
    });
  const open = currentEntries(nextGroups.open);
  const resolved = currentEntries(nextGroups.resolved);
  const ordered = [...open, ...resolved];
  const handleStateChanged = (name: string, resolved: boolean) => {
    if (!expandedRoots.has(name)) return;
    if (!resolved) {
      onExpand(name);
      return;
    }
    const index = ordered.findIndex((entry) => entry.thread.root.name === name);
    const next = [...ordered.slice(index + 1), ...ordered.slice(0, index)].find(
      (entry) => entry.thread.root.name !== name && !entry.thread.resolved
    );
    if (next) onExpand(next.thread.root.name);
    else onCollapse(name);
  };
  // Threads ending on the same line may cover different ranges; label each
  // with its range only when they differ.
  const showRanges =
    new Set(threads.map((entry) => formatLineRange(entry.range))).size > 1;
  const rangeLabel = (entry: EditorThread) =>
    showRanges ? (
      <span className="shrink-0 text-xs text-control-light">
        {lineRangeLabel(t, entry.range)}
      </span>
    ) : undefined;
  return (
    <div
      className="@container flex min-w-0 flex-col gap-y-2"
      data-testid="thread-stack"
    >
      {ordered.map((entry) => (
        <div
          className={cn(
            flashRoot === entry.thread.root.name && "bb-thread-card--flash"
          )}
          key={entry.thread.root.name}
          onAnimationEnd={
            flashRoot === entry.thread.root.name ? onFlashEnd : undefined
          }
          ref={
            expandedRoots.has(entry.thread.root.name)
              ? expandedThreadRef
              : undefined
          }
        >
          {expandedRoots.has(entry.thread.root.name) ? (
            <CommentThreadCard
              className="shadow-sm"
              collapsible={false}
              issueName={issueName}
              label={rangeLabel(entry)}
              onClose={() => onCollapse(entry.thread.root.name)}
              onThreadStateChanged={(resolved) =>
                handleStateChanged(entry.thread.root.name, resolved)
              }
              replyDraft={replyDrafts[entry.thread.root.name] ?? ""}
              onReplyDraftChange={(draft) =>
                onReplyDraftChange(entry.thread.root.name, draft)
              }
              project={project}
              thread={entry.thread}
            />
          ) : (
            <CollapsedThreadRow
              comment={entry.thread.root}
              label={rangeLabel(entry)}
              onSelect={() => onExpand(entry.thread.root.name)}
              replyCount={entry.thread.replies.length}
              resolved={entry.thread.resolved}
            />
          )}
        </div>
      ))}
    </div>
  );
}

function CollapsedThreadRow({
  comment,
  label,
  onSelect,
  replyCount,
  resolved,
}: {
  comment: IssueComment;
  label?: ReactNode;
  onSelect: () => void;
  replyCount: number;
  resolved: boolean;
}) {
  const { t } = useTranslation();
  const creator =
    useUserByIdentifier(comment.creator) ?? unknownUser(comment.creator);
  const firstLine = comment.comment.split("\n")[0] ?? "";
  return (
    <Button
      appearance="secondary"
      className="w-full min-w-0 justify-start rounded-sm border border-block-border bg-background text-left font-normal hover:bg-control-bg"
      data-testid="thread-row"
      data-thread-name={comment.name}
      aria-expanded={false}
      onClick={onSelect}
      size="md"
    >
      <UserAvatar
        colorSeed={creator.email}
        size="xs"
        title={creator.title || creator.email}
      />
      <span
        className="max-w-24 truncate font-medium text-main"
        title={creator.title || creator.email}
      >
        {creator.title || creator.email}
      </span>
      <span className="min-w-0 flex-1 truncate text-control" title={firstLine}>
        {firstLine}
      </span>
      {label}
      {replyCount > 0 && (
        <span
          className="flex shrink-0 items-center gap-x-1 text-xs text-control-light"
          title={t("plan.review.thread.n-replies", { count: replyCount })}
        >
          <MessageSquare className="size-3.5" />
          <span aria-hidden="true">{replyCount}</span>
          <span className="sr-only">
            {t("plan.review.thread.n-replies", { count: replyCount })}
          </span>
        </span>
      )}
      {comment.createTime && (
        <HumanizeTs
          className="hidden shrink-0 text-xs text-control-light @xl:inline"
          ts={getTimeForPbTimestampProtoEs(comment.createTime, 0) / 1000}
        />
      )}
      {resolved && (
        <Badge className="shrink-0 gap-x-1 px-2 text-xs" variant="success">
          <CircleCheck className="size-3" />
          {t("plan.review.thread.resolved")}
        </Badge>
      )}
      <ChevronDown className="size-4 shrink-0 text-control-light" />
    </Button>
  );
}
