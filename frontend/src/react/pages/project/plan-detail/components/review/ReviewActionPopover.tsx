import { create } from "@bufbuild/protobuf";
import { Loader2, MessageCircle } from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import {
  type ActivityIconSpec,
  ICON_TEXT_TONE,
  REVIEW_DECISION_ICON,
} from "@/react/components/issue-activity/activityIcons";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  ApproveIssueRequestSchema,
  RejectIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";

type ReviewAction = "COMMENT" | "APPROVE" | "REJECT";

export function ReviewActionPopover({
  issue,
  onClose,
}: {
  issue: Issue;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [action, setAction] = useState<ReviewAction>("COMMENT");
  const [comment, setComment] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setAction("COMMENT");
    setComment("");
  }, [issue.name]);

  const commentMissing = comment.trim().length === 0;
  const submitDisabled =
    loading ||
    (action === "COMMENT" && commentMissing) ||
    (action === "REJECT" && commentMissing);

  const submit = async () => {
    if (submitDisabled) return;
    try {
      setLoading(true);
      if (action === "APPROVE") {
        const response = await issueServiceClientConnect.approveIssue(
          create(ApproveIssueRequestSchema, { comment, name: issue.name })
        );
        page.patchState({ issue: response });
      } else if (action === "REJECT") {
        const response = await issueServiceClientConnect.rejectIssue(
          create(RejectIssueRequestSchema, { comment, name: issue.name })
        );
        page.patchState({ issue: response });
      } else {
        await useAppStore.getState().createIssueComment({
          issueName: issue.name,
          comment,
        });
      }
      await page.refreshState();
      onClose();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex w-[min(34rem,calc(100vw-2rem))] flex-col gap-y-3">
      <MarkdownEditor
        content={comment}
        onChange={setComment}
        onSubmit={() => void submit()}
      />
      <div className="flex flex-col gap-y-2.5">
        <ReviewOption
          description={t("issue.review.comment-description")}
          icon={<MessageCircle className="size-4 text-control" />}
          label={t("common.comment")}
          onSelect={() => setAction("COMMENT")}
          selected={action === "COMMENT"}
        />
        <ReviewOption
          description={t("issue.review.approve-description")}
          icon={<DecisionIcon spec={REVIEW_DECISION_ICON.approved} />}
          label={t("common.approve")}
          onSelect={() => setAction("APPROVE")}
          selected={action === "APPROVE"}
        />
        <ReviewOption
          description={t("issue.review.reject-description")}
          icon={<DecisionIcon spec={REVIEW_DECISION_ICON.rejected} />}
          label={t("common.reject")}
          onSelect={() => setAction("REJECT")}
          selected={action === "REJECT"}
        />
      </div>
      <div className="flex items-center justify-start gap-x-2 pt-1">
        <Tooltip
          content={
            action === "REJECT" && commentMissing
              ? t("plan.review.comment-required-to-reject")
              : undefined
          }
        >
          <span className="inline-flex">
            <Button
              disabled={submitDisabled}
              onClick={() => void submit()}
              size="sm"
            >
              {loading && <Loader2 className="size-4 animate-spin" />}
              {t("common.submit")}
            </Button>
          </span>
        </Tooltip>
        <Button onClick={onClose} size="sm" variant="ghost">
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );
}

// Tints the shared review-decision glyph for the action menu (text tone, vs the
// filled badge the same spec produces in the activity timeline).
function DecisionIcon({ spec }: { spec: ActivityIconSpec }) {
  const { Icon, tone } = spec;
  return <Icon className={cn("size-4", ICON_TEXT_TONE[tone])} />;
}

function ReviewOption({
  description,
  icon,
  label,
  onSelect,
  selected,
}: {
  description?: string;
  icon?: ReactNode;
  label: string;
  onSelect: () => void;
  selected: boolean;
}) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-start gap-3 text-left transition-colors",
        selected ? "text-main" : "text-control"
      )}
    >
      <input
        checked={selected}
        className="mt-1 size-4 accent-accent"
        onChange={onSelect}
        type="radio"
      />
      {icon && <span className="mt-1 shrink-0">{icon}</span>}
      <span className="flex flex-col">
        <span className="text-sm font-medium leading-6">{label}</span>
        {description && (
          <span className="text-xs text-control-light">{description}</span>
        )}
      </span>
    </label>
  );
}
