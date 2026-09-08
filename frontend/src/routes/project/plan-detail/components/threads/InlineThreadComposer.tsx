import { Loader2, Send } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { UserAvatar } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { useCurrentUser } from "@/hooks/useAppState";
import type { LineRange } from "./threadModel";

// The composer that opens below the last selected line. Publishing creates a
// root comment anchored to those lines; the draft survives a failed publish.
export function InlineThreadComposer({
  onCancel,
  onPublish,
  pending,
  range,
}: {
  onCancel: () => void;
  onPublish: (comment: string) => Promise<boolean>;
  pending: boolean;
  range: LineRange;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const [draft, setDraft] = useState("");
  const canPublish = !pending && draft.trim() !== "";

  const publish = async () => {
    if (!canPublish) return;
    const published = await onPublish(draft);
    if (published) setDraft("");
  };

  return (
    <div
      className="flex flex-col rounded-sm border border-block-border bg-background"
      data-testid="inline-thread-composer"
    >
      <div className="flex items-center gap-x-2 border-b border-block-border bg-control-bg/50 px-3 py-1.5 text-sm">
        <UserAvatar
          colorSeed={currentUser.email}
          size="xs"
          title={currentUser.title || currentUser.email}
        />
        <span className="min-w-0 font-medium text-main">
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
      <div className="flex flex-col gap-y-2 px-3 py-2">
        <MarkdownEditor
          compact
          content={draft}
          onChange={setDraft}
          onSubmit={() => void publish()}
          placeholder={t("plan.review.thread.write-comment")}
        />
        <div className="flex items-center justify-end gap-x-2">
          <Button onClick={onCancel} size="sm" appearance="secondary">
            {t("common.cancel")}
          </Button>
          <Button
            disabled={!canPublish}
            onClick={() => void publish()}
            size="sm"
          >
            {pending ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : (
              <Send className="size-3.5" />
            )}
            {t("plan.review.thread.publish")}
          </Button>
        </div>
      </div>
    </div>
  );
}
