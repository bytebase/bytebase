import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/components/monaco";
import { getTaskRunWaitingMessage } from "@/lib/taskRun";
import { cn } from "@/lib/utils";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  isReleaseBasedTask,
  releaseNameOfTaskV1,
} from "@/utils/v1/issue/rollout";
import { DeployLatestTaskRunInfo } from "./DeployLatestTaskRunInfo";
import { DeployReleaseInfoCard } from "./DeployReleaseInfoCard";
import { DeployTaskSkippedReason } from "./DeployTaskSkippedReason";

// The expanded card body: a skipped reason, a waiting note, the SQL statement
// (or release info), and the latest task run. Only mounts when the card is
// expanded, so everything derived here is off the collapsed poll path. Stateful
// inputs (the statement, store lookups) stay in the orchestrating
// DeployTaskItem; pure derivations of `task`/`latestTaskRun` live here.
export function DeployTaskBody({
  active,
  databaseEngine,
  historyCount,
  isStatementLoading,
  isTruncated,
  latestTaskRun,
  onShowHistory,
  statement,
  task,
  timingDisplay,
}: {
  // Whether this card's stage is the visible one; forwarded to pause live polls.
  active?: boolean;
  databaseEngine?: Engine;
  historyCount: number;
  isStatementLoading: boolean;
  isTruncated: boolean;
  latestTaskRun?: TaskRun;
  onShowHistory: () => void;
  statement: string;
  task: Task;
  // The latest run's duration, formatted by DeployTaskItem (single source).
  timingDisplay: string;
}) {
  const { t } = useTranslation();
  const isReleaseTask = isReleaseBasedTask(task);
  const releaseName = releaseNameOfTaskV1(task);
  // The one state line that matters for a waiting task.
  const waitingComment = latestTaskRun
    ? getTaskRunWaitingMessage(latestTaskRun, t)
    : undefined;
  // A pending task shows no elapsed time; otherwise reuse the item's string.
  const duration =
    task.status === Task_Status.PENDING
      ? undefined
      : timingDisplay || undefined;

  return (
    <div className="flex flex-col gap-3">
      <DeployTaskSkippedReason task={task} />
      {waitingComment && (
        <div className="text-xs italic text-control-light">
          {waitingComment}
        </div>
      )}
      {!isReleaseTask ? (
        <div>
          <div className="mb-1 text-sm font-medium text-control">
            {t("common.statement")}
          </div>
          {isStatementLoading ? (
            // min-h-32 matches the editor's min height (min={128}) so the
            // statement fills this box in place instead of reflowing the card.
            <div className="flex min-h-32 items-center justify-center rounded-sm border p-3 text-sm text-control-light">
              {t("common.loading")}
            </div>
          ) : statement ? (
            <>
              <ReadonlyMonaco
                className={cn(
                  "relative rounded border text-sm",
                  isTruncated && "rounded-b-none"
                )}
                content={statement}
                language="sql"
                min={128}
                max={256}
              />
              {isTruncated && (
                <div className="rounded-b border border-t-0 bg-control-bg px-3 py-1.5 text-xs text-control-light">
                  {t("rollout.task.statement-truncated-hint")}
                </div>
              )}
            </>
          ) : (
            <div className="rounded-sm border p-3 text-sm text-control-light">
              {t("common.no-data")}
            </div>
          )}
        </div>
      ) : releaseName ? (
        <DeployReleaseInfoCard releaseName={releaseName} />
      ) : null}

      {latestTaskRun && (
        <DeployLatestTaskRunInfo
          active={active}
          databaseEngine={databaseEngine}
          duration={duration}
          historyCount={historyCount}
          onShowHistory={onShowHistory}
          taskRun={latestTaskRun}
        />
      )}
    </div>
  );
}
