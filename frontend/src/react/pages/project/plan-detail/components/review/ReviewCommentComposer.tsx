import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";

const draftKey = (issueName: string) => `bb.plan-review.draft.${issueName}`;

export function ReviewCommentComposer({ issueName }: { issueName: string }) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const [expanded, setExpanded] = useState(false);
  const [content, setContent] = useState("");
  const [posting, setPosting] = useState(false);

  // Restore the unsent draft on mount / issue change.
  useEffect(() => {
    const draft = localStorage.getItem(draftKey(issueName)) ?? "";
    setContent(draft);
    setExpanded(false);
  }, [issueName]);

  const persistDraft = useCallback(
    (value: string) => {
      setContent(value);
      if (value) {
        localStorage.setItem(draftKey(issueName), value);
      } else {
        localStorage.removeItem(draftKey(issueName));
      }
    },
    [issueName]
  );

  const post = async () => {
    if (!content.trim() || posting) return;
    try {
      setPosting(true);
      await useAppStore.getState().createIssueComment({
        issueName,
        comment: content,
      });
      persistDraft("");
      setExpanded(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setPosting(false);
    }
  };

  if (!expanded) {
    return (
      <div className="flex items-center gap-x-3">
        <div className="shrink-0 pl-0.5">
          <UserAvatar
            size="sm"
            title={currentUser.title || currentUser.email}
          />
        </div>
        <button
          className="min-w-0 flex-1 rounded-md border px-3 py-1.5 text-left text-sm text-control-placeholder hover:border-control-border"
          onClick={() => setExpanded(true)}
          type="button"
        >
          {t("plan.review.activity.add-a-comment")}
        </button>
      </div>
    );
  }

  return (
    <div className="flex items-start gap-x-3">
      <div className="shrink-0 pl-0.5 pt-1">
        <UserAvatar size="sm" title={currentUser.title || currentUser.email} />
      </div>
      <div className="min-w-0 flex-1">
        <MarkdownEditor
          content={content}
          onChange={persistDraft}
          onSubmit={() => void post()}
          placeholder={t("plan.review.activity.add-a-comment")}
        />
        <div className="mt-2 flex items-center justify-end gap-x-2">
          <Button onClick={() => setExpanded(false)} size="sm" variant="ghost">
            {t("common.cancel")}
          </Button>
          <Button
            disabled={posting || content.trim().length === 0}
            onClick={() => void post()}
            size="sm"
          >
            {posting && <Loader2 className="size-4 animate-spin" />}
            {t("common.comment")}
          </Button>
        </div>
      </div>
    </div>
  );
}
