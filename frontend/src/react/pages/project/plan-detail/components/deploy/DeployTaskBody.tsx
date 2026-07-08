import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/react/components/monaco";
import {
  executionDurationOfTaskRun,
  getTaskRunWaitingMessage,
} from "@/react/lib/taskRun";
import { cn } from "@/react/lib/utils";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { humanizeDurationV1 } from "@/utils";
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
  databaseEngine,
  historyCount,
  isStatementLoading,
  isTruncated,
  latestTaskRun,
  onShowHistory,
  statement,
  task,
}: {
  databaseEngine?: Engine;
  historyCount: number;
  isStatementLoading: boolean;
  isTruncated: boolean;
  latestTaskRun?: TaskRun;
  onShowHistory: () => void;
  statement: string;
  task: Task;
}) {
  const { t } = useTranslation();
  const isReleaseTask = isReleaseBasedTask(task);
  const releaseName = releaseNameOfTaskV1(task);
  // The one state line that matters for a waiting task.
  const waitingComment = latestTaskRun
    ? getTaskRunWaitingMessage(latestTaskRun, t)
    : undefined;
  const latestRunDuration =
    latestTaskRun && task.status !== Task_Status.PENDING
      ? executionDurationOfTaskRun(latestTaskRun)
      : undefined;

  return (
    <div className="space-y-3">
      <DeployTaskSkippedReason task={task} />
      {waitingComment && (
        <div className="text-xs italic text-gray-500">{waitingComment}</div>
      )}
      {!isReleaseTask ? (
        <div>
          <div className="mb-1 text-sm font-medium text-gray-700">
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
                <div className="rounded-b border border-t-0 bg-gray-50 px-3 py-1.5 text-xs text-gray-500">
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
          databaseEngine={databaseEngine}
          duration={
            latestRunDuration
              ? humanizeDurationV1(latestRunDuration)
              : undefined
          }
          historyCount={historyCount}
          onShowHistory={onShowHistory}
          taskRun={latestTaskRun}
        />
      )}
    </div>
  );
}
