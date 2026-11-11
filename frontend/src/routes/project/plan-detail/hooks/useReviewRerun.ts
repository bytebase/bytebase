import { create } from "@bufbuild/protobuf";
import { ConnectError, createContextValues } from "@connectrpc/connect";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { skipErrorNotificationContextKey } from "@/api/context-key";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  ReviewRun_State,
  RunReviewRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  aiReviewEnabled,
  hasReviewableSQL,
  isReviewRunActive,
  loadedMatchedDatabaseCount,
  reviewRunName,
} from "../utils/reviewRun";
import { useSqlReviewV2 } from "./useSqlReviewV2";

// Starts the issue's review again on the saved SQL. The SQL is not the only
// input: rule settings, the review policy, and the synced schema change what
// a review finds, so a review with a result can still need another run.
// RunReview accepts it only for an open issue whose SQL no rollout has frozen
// and that has SQL to review. It replaces whatever execution the run holds,
// so a run cannot start while one is queued or running, nor while the page
// has not read the run, since an unseen run may be either.
export function useReviewRerun() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const review = useSqlReviewV2()?.review;
  const serverInfo = useAppStore((state) => state.serverInfo);
  const dbGroupsByName = useAppStore((state) => state.dbGroupsByName);
  const dbGroupViewByName = useAppStore((state) => state.dbGroupViewByName);
  const [starting, setStarting] = useState(false);
  const { issue } = page;
  const allowed =
    issue?.status === IssueStatus.OPEN &&
    !page.rollout &&
    !page.readonly &&
    hasReviewableSQL(
      page.plan,
      loadedMatchedDatabaseCount({ dbGroupsByName, dbGroupViewByName })
    ) &&
    hasProjectPermissionV2(page.project, "bb.reviewRuns.run");
  const current = review?.run;

  const run = useCallback(async () => {
    if (!issue) return;
    setStarting(true);
    try {
      await issueServiceClientConnect.runReview(
        create(RunReviewRequestSchema, { name: reviewRunName(issue.name) }),
        {
          contextValues: createContextValues().set(
            skipErrorNotificationContextKey,
            true
          ),
        }
      );
      await review?.reload();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("plan.automated-review.failed-to-rerun"),
        description: ConnectError.from(error).rawMessage,
      });
    } finally {
      setStarting(false);
    }
  }, [issue, review, t]);

  return {
    allowed,
    // A run reviews the saved plan, not the edits on screen.
    editing: page.isEditing,
    disabled:
      starting ||
      page.isEditing ||
      current === undefined ||
      isReviewRunActive(current),
    // A finished run of the AI review is paid for in model calls, so running
    // it again asks first.
    confirmFirst:
      aiReviewEnabled(serverInfo) &&
      current?.state === ReviewRun_State.SUCCEEDED &&
      !current.outdated,
    run,
  };
}
