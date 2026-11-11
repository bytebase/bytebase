import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { UserAvatar } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { useCurrentUser } from "@/hooks/useAppState";
import type { LineRange } from "./threadModel";

// The composer that opens below the last selected line. Publishing creates a
// root comment anchored to those lines. The owner holds the draft, so it
// survives a failed publish and the form's own re-renders.
export function InlineThreadComposer({
  draft,
  onCancel,
  onDraftChange,
  onPublish,
  pending,
  range,
}: {
  draft: string;
  onCancel: () => void;
  onDraftChange: (draft: string) => void;
  onPublish: (comment: string) => void;
  pending: boolean;
  range: LineRange;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const canPublish = !pending && draft.trim() !== "";
  const publish = () => {
    if (canPublish) onPublish(draft);
  };

  return (
    <div
      className="flex flex-col rounded-sm border border-block-border bg-background shadow-sm"
      data-testid="inline-thread-composer"
    >
      <div
        className="flex min-h-8 min-w-0 items-center rounded-t-sm border-b border-block-border bg-control-bg px-3 text-xs text-control"
        data-testid="thread-bar"
      >
        <span className="min-w-0 truncate">
          {range.startLine === range.endLine
            ? t("plan.review.thread.add-comment-on-line", {
                line: range.startLine,
              })
            : t("plan.review.thread.add-comment-on-lines", {
                startLine: range.startLine,
                endLine: range.endLine,
              })}
        </span>
      </div>
      <div className="flex items-start gap-x-2 px-2 py-2 sm:px-3">
        <div
          className="flex h-6 w-5 shrink-0 items-center"
          data-testid="composer-avatar"
        >
          <UserAvatar
            colorSeed={currentUser.email}
            size="xs"
            title={currentUser.title || currentUser.email}
          />
        </div>
        <div className="flex min-w-0 flex-1 flex-col gap-y-2">
          <MarkdownEditor
            autoFocus
            compact
            content={draft}
            onChange={onDraftChange}
            onSubmit={publish}
            placeholder={t("plan.review.thread.write-comment")}
          />
          <div className="flex items-center justify-end gap-x-2">
            <Button onClick={onCancel} size="sm" appearance="secondary">
              {t("common.cancel")}
            </Button>
            <Button disabled={!canPublish} onClick={publish} size="sm">
              {pending && <Loader2 className="size-4 animate-spin" />}
              {t("plan.review.thread.publish")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
