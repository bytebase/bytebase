import { Ellipsis } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { writeTextToClipboard } from "@/lib/clipboard";
import { useAppStore } from "@/stores/app";
import { useCommentLinks } from "../../shell/CommentLinkContext";

// A comment's actions behind one button: Copy link where the comment has an
// address, and Edit for a reader who may edit. Nothing renders with neither.
export function CommentActionsMenu({
  authorName,
  commentName,
  onEdit,
}: {
  authorName: string;
  commentName: string;
  onEdit?: () => void;
}) {
  const { t } = useTranslation();
  const url = useCommentLinks()?.urlOf(commentName);
  if (!url && !onEdit) return null;

  const copyLink = async () => {
    if (!url) return;
    const ok = await writeTextToClipboard(url);
    useAppStore.getState().notify({
      module: "bytebase",
      style: ok ? "SUCCESS" : "CRITICAL",
      title: ok ? t("common.copied") : t("common.copy-failed"),
    });
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            appearance="secondary"
            aria-label={t("plan.review.thread.comment-actions", {
              name: authorName,
            })}
            size="xs"
          >
            <Ellipsis className="size-3.5" />
          </Button>
        }
      />
      <DropdownMenuContent>
        {url && (
          <DropdownMenuItem onClick={() => void copyLink()}>
            {t("common.copy-link")}
          </DropdownMenuItem>
        )}
        {onEdit && (
          <DropdownMenuItem onClick={onEdit}>
            {t("common.edit")}
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
