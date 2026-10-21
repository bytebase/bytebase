import { create } from "@bufbuild/protobuf";
import { ConnectError, createContextValues } from "@connectrpc/connect";
import { Loader2, Play, XCircle } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/api";
import { skipErrorNotificationContextKey } from "@/api/context-key";
import { HumanizeTs } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  type ReviewRun,
  ReviewRun_Status,
  RunReviewRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  aiReviewRunName,
  isAIReviewRunActive,
  loadedMatchedDatabaseCount,
  readsAIReviewRun,
} from "../utils/aiReviewRun";

// The issue's AI review, beside the plan checks at the foot of the CHANGES
// card: its status and a Run button. Findings are review comments, and
// nothing gates on a run, so a finished run is not a pass.
export function PlanDetailAIReview() {
  const { t } = useTranslation();
  const { allowRun, editing, run, runReview, shown, starting } =
    useAIReviewRun();
  if (!shown) return null;

  return (
    <>
      <Separator />
      <div className="flex flex-col gap-y-1 px-4 py-4" data-testid="ai-review">
        <div className="flex items-center justify-between gap-2">
          <h3 className="textlabel uppercase">
            {t("sql-review.ai-review.self")}
          </h3>
          {allowRun && (
            <Button
              appearance="outline"
              aria-label={t("plan.ai-review.run")}
              disabled={
                starting ||
                editing ||
                run === undefined ||
                isAIReviewRunActive(run)
              }
              onClick={() => void runReview()}
              size="xs"
            >
              <Play className="size-3.5" />
              {t("common.run")}
            </Button>
          )}
        </div>
        <AIReviewRunStatus run={run} />
        {allowRun && editing && (
          <span className="text-sm text-control-light">
            {t("plan.ai-review.save-first")}
          </span>
        )}
      </div>
    </>
  );
}

function AIReviewRunStatus({ run }: { run: ReviewRun | null | undefined }) {
  const { t } = useTranslation();
  if (run === undefined) {
    return (
      <span className="text-sm text-control-placeholder">
        {t("plan.ai-review.status-unknown")}
      </span>
    );
  }
  if (run === null) {
    return (
      <span className="text-sm text-control-placeholder">
        {t("plan.ai-review.not-run")}
      </span>
    );
  }
  switch (run.status) {
    case ReviewRun_Status.AVAILABLE:
    case ReviewRun_Status.RUNNING:
      return (
        <span className="flex items-center gap-x-1 text-sm text-control">
          <Loader2 className="size-4 animate-spin" />
          {run.status === ReviewRun_Status.AVAILABLE
            ? t("plan.ai-review.queued")
            : t("common.running")}
        </span>
      );
    case ReviewRun_Status.DONE:
      return (
        <span className="flex flex-wrap items-center gap-x-1 text-sm text-control">
          {t("plan.ai-review.finished")}
          <HumanizeTs
            className="text-control-light"
            tsMs={getTimeForPbTimestampProtoEs(run.endTime)}
          />
        </span>
      );
    case ReviewRun_Status.FAILED:
      return (
        <div className="flex flex-col gap-y-1 text-sm">
          <span className="flex flex-wrap items-center gap-x-1 text-error">
            <XCircle className="size-4" />
            {t("common.failed")}
            <HumanizeTs
              className="text-control-light"
              tsMs={getTimeForPbTimestampProtoEs(run.endTime)}
            />
          </span>
          <p
            className="wrap-break-word whitespace-pre-wrap text-control"
            data-testid="ai-review-error"
          >
            {run.error}
          </p>
        </div>
      );
    default:
      return null;
  }
}

// RunReview replaces whatever execution the slot holds and the backend has no
// same-version no-op, so the button stays disabled while a run is queued or
// running, and while the page has not read the slot, since an unseen run may
// be either. The started run joins the page snapshot, whose polling turns
// active until the run settles.
function useAIReviewRun() {
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

  const runReview = useCallback(async () => {
    if (!issue) return;
    setStarting(true);
    try {
      const run = await issueServiceClientConnect.runReview(
        create(RunReviewRequestSchema, { name: aiReviewRunName(issue.name) }),
        {
          contextValues: createContextValues().set(
            skipErrorNotificationContextKey,
            true
          ),
        }
      );
      patchState({ aiReviewRun: run });
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

  // A run reviews the saved plan, so unsaved SQL would get findings about
  // the statement the user has already changed.
  return {
    allowRun,
    editing: page.isEditing,
    run: page.aiReviewRun,
    runReview,
    shown,
    starting,
  };
}
