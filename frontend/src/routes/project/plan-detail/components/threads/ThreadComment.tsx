import { Loader2, Pencil } from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { canEditIssueComment } from "@/components/issue-activity/IssueCommentActivity";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import { UserAvatar } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { useCurrentUser, useUserByIdentifier } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

// Roots and replies share one layout: author identity in the header, with
// the body aligned to the comment region rather than indented below the name.
export function ThreadComment({
  actions,
  className,
  comment,
  onEdit,
  project,
  status,
}: {
  // Header actions after the edit button (thread collapse or close).
  actions?: ReactNode;
  className?: string;
  comment: IssueComment;
  onEdit: (issueCommentName: string, comment: string) => Promise<unknown>;
  project: Project | undefined;
  // Thread status shown after the timestamp (resolved).
  status?: ReactNode;
}) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const creator =
    useUserByIdentifier(comment.creator) ?? unknownUser(comment.creator);
  const createdTs = getTimeForPbTimestampProtoEs(comment.createTime, 0);
  const updatedTs = getTimeForPbTimestampProtoEs(comment.updateTime, 0);
  const isEdited = createdTs !== updatedTs;
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(comment.comment);
  const [saving, setSaving] = useState(false);
  const allowEdit = canEditIssueComment(comment, currentUser.email, project);

  useEffect(() => {
    if (!isEditing) setDraft(comment.comment);
  }, [comment.comment, isEditing]);

  const save = async () => {
    if (!draft.trim() || draft === comment.comment) {
      setIsEditing(false);
      return;
    }
    setSaving(true);
    const updated = await onEdit(comment.name, draft);
    setSaving(false);
    if (updated) setIsEditing(false);
  };

  return (
    <div
      className={cn(
        "@container/comment flex items-start gap-x-2 px-2 py-2 sm:px-3",
        className
      )}
      data-testid="thread-comment"
      id={comment.name}
    >
      <div
        className="flex min-w-0 flex-1 flex-col gap-y-2"
        data-testid="comment-content"
      >
        <div className="flex min-h-6 items-center gap-x-2">
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1">
            <div className="flex min-w-0 max-w-full items-center gap-x-2">
              <div className="shrink-0" data-testid="comment-avatar">
                <UserAvatar
                  colorSeed={creator.email}
                  size="xs"
                  title={creator.title || creator.email}
                />
              </div>
              <span
                className="min-w-0 truncate text-sm font-medium text-main"
                title={creator.title || creator.email}
              >
                {creator.title || creator.email}
              </span>
            </div>
            {comment.createTime && (
              <HumanizeTs
                className="hidden shrink-0 whitespace-nowrap text-xs text-control-light @2xs/comment:inline"
                ts={createdTs / 1000}
              />
            )}
            {isEdited && (
              <span className="hidden shrink-0 whitespace-nowrap text-xs text-control-light @2xs/comment:inline">
                ({t("common.edited")})
              </span>
            )}
            {status}
          </div>
          <div className="flex shrink-0 items-center gap-x-1">
            {allowEdit && !isEditing && (
              <Button
                aria-label={t("common.edit")}
                onClick={() => {
                  setDraft(comment.comment);
                  setIsEditing(true);
                }}
                size="xs"
                appearance="secondary"
              >
                <Pencil className="size-3.5" />
              </Button>
            )}
            {actions}
          </div>
        </div>
        {isEditing ? (
          <div className="flex flex-col gap-y-2">
            <MarkdownEditor
              content={draft}
              onChange={setDraft}
              onSubmit={() => void save()}
            />
            <div className="flex items-center justify-end gap-x-2">
              <Button
                onClick={() => setIsEditing(false)}
                size="xs"
                appearance="secondary"
              >
                {t("common.cancel")}
              </Button>
              <Button
                disabled={
                  saving ||
                  draft.trim().length === 0 ||
                  draft === comment.comment
                }
                onClick={() => void save()}
                size="xs"
              >
                {saving && <Loader2 className="size-3.5 animate-spin" />}
                {t("common.save")}
              </Button>
            </div>
          </div>
        ) : (
          comment.comment && (
            <div className="wrap-break-word text-sm text-control">
              <MarkdownEditor content={comment.comment} mode="preview" />
            </div>
          )
        )}
      </div>
    </div>
  );
}
