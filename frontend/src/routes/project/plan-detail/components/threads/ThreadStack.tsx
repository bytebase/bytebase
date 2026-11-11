import { type Ref, type SetStateAction, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { CommentThreadCard } from "./CommentThreadCard";
import {
  commentTitle,
  compareByCreateTime,
  type EditorThread,
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
  issue,
  onCollapse,
  onExpand,
  replyDrafts,
  onReplyDraftChange,
  project,
  revealRoot,
  spec,
  threads,
}: {
  expandedRoots: ReadonlySet<string>;
  expandedThreadRef?: Ref<HTMLDivElement>;
  // The thread the walker just landed on; its card flashes once.
  flashRoot?: string;
  onFlashEnd?: () => void;
  issue: Issue;
  onCollapse: (rootName: string) => void;
  onExpand: (rootName: string) => void;
  replyDrafts: Readonly<Record<string, string>>;
  onReplyDraftChange: (rootName: string, draft: SetStateAction<string>) => void;
  project: Project | undefined;
  revealRoot?: string;
  // The spec whose statement the stack sits on.
  spec: Plan_Spec;
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
  return (
    <div
      className="@container flex min-w-0 flex-col gap-y-2"
      data-testid="thread-stack"
    >
      {ordered.map((entry) => {
        const rootName = entry.thread.root.name;
        const expanded = expandedRoots.has(rootName);
        const title = commentTitle(t, entry.range);
        return (
          <div
            className={cn(flashRoot === rootName && "bb-thread-card--flash")}
            key={rootName}
            onAnimationEnd={flashRoot === rootName ? onFlashEnd : undefined}
            ref={
              revealRoot === rootName && expanded
                ? expandedThreadRef
                : undefined
            }
          >
            <CommentThreadCard
              bar={{
                title,
                foldLabel: title,
                folded: !expanded,
                onToggleFold: () =>
                  expanded ? onCollapse(rootName) : onExpand(rootName),
              }}
              issue={issue}
              // Only the reader's own Resolve folds the card; a remote resolve
              // must not shift the editor under them.
              onThreadStateChanged={(resolved) => {
                if (resolved) onCollapse(rootName);
              }}
              replyDraft={replyDrafts[rootName] ?? ""}
              onReplyDraftChange={(draft) =>
                onReplyDraftChange(rootName, draft)
              }
              project={project}
              spec={spec}
              thread={entry.thread}
            />
          </div>
        );
      })}
    </div>
  );
}
