import { create } from "@bufbuild/protobuf";
import { ConnectError, createContextValues } from "@connectrpc/connect";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { skipErrorNotificationContextKey } from "@/api/context-key";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import { RunReviewRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  aiReviewRunName,
  isAIReviewRunActive,
  loadedMatchedDatabaseCount,
  readsAIReviewRun,
} from "../utils/aiReviewRun";

// The issue's AI review: whether the page shows it, and starting a run.
// RunReview replaces whatever execution the slot holds and the backend has no
// same-version no-op, so a run cannot start while one is queued or running,
// nor while the page has not read the slot, since an unseen run may be
// either. The started run joins the page snapshot, whose polling turns active
// until the run settles. A run reviews the saved plan, so unsaved SQL would
// get findings about a statement the user has already changed.
export function useAIReviewRun() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { issue, patchState, refreshState } = page;
  const serverInfo = useAppStore((state) => state.serverInfo);
  const dbGroupsByName = useAppStore((state) => state.dbGroupsByName);
  const dbGroupViewByName = useAppStore((state) => state.dbGroupViewByName);
  const [starting, setStarting] = useState(false);
  const shown =
    readsAIReviewRun({
      plan: page.plan,
      project: page.project,
      serverInfo,
      matchedDatabaseCount: loadedMatchedDatabaseCount({
        dbGroupsByName,
        dbGroupViewByName,
      }),
    }) &&
    issue?.status === IssueStatus.OPEN &&
    !page.rollout;
  const allowRun =
    shown && hasProjectPermissionV2(page.project, "bb.reviewRuns.run");
  const run = page.aiReviewRun;
  const canStart =
    allowRun &&
    !starting &&
    !page.isEditing &&
    run !== undefined &&
    !isAIReviewRunActive(run);

  const runReview = useCallback(async () => {
    if (!issue) return;
    setStarting(true);
    try {
      const started = await issueServiceClientConnect.runReview(
        create(RunReviewRequestSchema, { name: aiReviewRunName(issue.name) }),
        {
          contextValues: createContextValues().set(
            skipErrorNotificationContextKey,
            true
          ),
        }
      );
      patchState({ aiReviewRun: started });
      await refreshState().catch(() => undefined);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("plan.ai-review.failed-to-run"),
        description: ConnectError.from(error).rawMessage,
      });
    } finally {
      setStarting(false);
    }
  }, [issue, patchState, refreshState, t]);

  const editing = page.isEditing;
  return useMemo(
    () => ({ allowRun, canStart, editing, run, runReview, shown, starting }),
    [allowRun, canStart, editing, run, runReview, shown, starting]
  );
}
