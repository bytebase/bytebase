import { create } from "@bufbuild/protobuf";
import { ConnectError, createContextValues } from "@connectrpc/connect";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { skipErrorNotificationContextKey } from "@/api/context-key";
import { pushNotification } from "@/stores";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { RunReviewRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { useSqlReviewV2 } from "./useSqlReviewV2";

// Starts the standard rule review again on the saved SQL. The SQL is not the
// only input: rule settings and the synced schema change what the rules
// find, so a reviewer with a result can still need another run. RunReview
// accepts it only for an open issue whose SQL no rollout has frozen.
export function useRuleReviewRerun() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const ruleReview = useSqlReviewV2()?.ruleReview;
  const [starting, setStarting] = useState(false);
  const { issue } = page;
  const allowed =
    issue?.status === IssueStatus.OPEN &&
    !page.rollout &&
    !page.readonly &&
    hasProjectPermissionV2(page.project, "bb.reviewRuns.run");

  const run = useCallback(async () => {
    if (!issue) return;
    setStarting(true);
    try {
      await issueServiceClientConnect.runReview(
        create(RunReviewRequestSchema, {
          name: `${issue.name}/reviewRuns/rule`,
        }),
        {
          contextValues: createContextValues().set(
            skipErrorNotificationContextKey,
            true
          ),
        }
      );
      await ruleReview?.reload();
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
  }, [issue, ruleReview, t]);

  return {
    allowed,
    // A run reviews the saved plan, not the edits on screen.
    disabled: starting || page.isEditing,
    run,
  };
}
