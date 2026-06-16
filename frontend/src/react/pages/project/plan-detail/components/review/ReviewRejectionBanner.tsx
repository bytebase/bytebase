import { create } from "@bufbuild/protobuf";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { Alert } from "@/react/components/ui/alert";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";
import { pushNotification } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueComment_Approval_Status,
  RequestIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";

function findLastRejection(comments: IssueComment[]): IssueComment | undefined {
  for (let i = comments.length - 1; i >= 0; i--) {
    const comment = comments[i];
    if (
      getIssueCommentType(comment) === IssueCommentType.APPROVAL &&
      comment.event.case === "approval" &&
      comment.event.value.status === IssueComment_Approval_Status.REJECTED
    ) {
      return comment;
    }
  }
  return undefined;
}

export function ReviewRejectionBanner({
  comments,
  issue,
}: {
  comments: IssueComment[];
  issue: Issue;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = useCurrentUser();
  const [reRequesting, setReRequesting] = useState(false);
  const rejection = useMemo(() => findLastRejection(comments), [comments]);
  const rejectorUser = useAppStore((state) =>
    rejection ? state.getUserByIdentifier(rejection.creator) : undefined
  );

  if (issue.approvalStatus !== ApprovalStatus.REJECTED || !rejection) {
    return null;
  }

  const rejector = rejectorUser ?? unknownUser(rejection.creator);
  const isCreator =
    issue.creator === `${userNamePrefix}${currentUser?.email ?? ""}`;

  const handleReRequest = async () => {
    if (reRequesting) return;
    try {
      setReRequesting(true);
      const response = await issueServiceClientConnect.requestIssue(
        create(RequestIssueRequestSchema, { name: issue.name })
      );
      page.patchState({ issue: response });
      await page.refreshState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setReRequesting(false);
    }
  };

  return (
    <Alert
      className="mx-4 mt-3 w-auto px-3 py-2"
      showIcon={false}
      variant="error"
      title={
        <span className="flex items-center gap-x-2 text-error">
          <span>
            {t("plan.review.rejection.title", { user: rejector.title })}
          </span>
          {rejection.createTime && (
            <>
              <span className="text-error/50">·</span>
              <HumanizeTs
                className="text-xs font-normal text-error/70"
                ts={
                  getTimeForPbTimestampProtoEs(rejection.createTime, 0) / 1000
                }
              />
            </>
          )}
        </span>
      }
    >
      <div className="mt-1 flex flex-col gap-y-1 text-xs text-control-light">
        {rejection.comment && (
          <span className="wrap-anywhere whitespace-pre-wrap text-sm text-control">
            {rejection.comment}
          </span>
        )}
        <span>
          {t("plan.review.rejection.guidance-prefix")}{" "}
          {isCreator && !page.readonly ? (
            <button
              className="font-medium text-accent hover:underline disabled:opacity-60"
              disabled={reRequesting}
              onClick={() => void handleReRequest()}
              type="button"
            >
              {t("plan.review.rejection.re-request-review")}
            </button>
          ) : (
            <span>{t("plan.review.rejection.re-request-review")}</span>
          )}{" "}
          {t("plan.review.rejection.guidance-suffix")}
        </span>
      </div>
    </Alert>
  );
}
