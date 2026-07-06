// Deploy-phase lifecycle slot content, pointed at the frontier stage. "Run"
// opens the same run confirmation sheet the Deploy section uses; a user who
// can't run the stage — and every user while it is running — sees the frontier's
// computed status instead (consistent with the review phase: not actionable →
// status, not a disabled action; and with the Deploy section, which renders the
// same getStageStatus + TaskStatusIcon). Skip / cancel / rollback and per-task
// controls stay in the Deploy section beside their task context.
import { Play } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TaskStatusIcon } from "@/react/components/TaskStatusIcon";
import { Button } from "@/react/components/ui/button";
import { router } from "@/react/router";
import {
  type Stage,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { getStageStatus, stringifyTaskStatus } from "@/utils";
import {
  canRolloutTasks,
  preloadRolloutPermissionContext,
  RUNNABLE_TASK_STATUSES,
} from "../../../issue-detail/utils/rollout";
import { focusPlanPhase } from "../../shell/focusPhase";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { PlanDetailTaskRolloutActionPanel } from "../PlanDetailTaskRolloutActionPanel";
import { stageHasFailedOrCanceledTasks } from "./frontierStage";
import { LifecycleStamp } from "./LifecycleStamp";
import { useStageTitle } from "./useStageTitle";

// A middot separator between the action/status word and the stage name. Purely
// presentational (aria-hidden) and inherits the surrounding text color, so it
// matches the label — the stage is a sibling span, not baked into the i18n string.
function StageSeparator() {
  return (
    <span aria-hidden className="shrink-0">
      ·
    </span>
  );
}

export function RunStageAction({ stage }: { stage: Stage }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const project = page.project;
  const currentUser = page.currentUser;
  const [open, setOpen] = useState(false);
  const [permissionReady, setPermissionReady] = useState(false);
  const title = useStageTitle(stage);
  // If any task in the frontier stage already ran (failed / canceled), running
  // the stage re-runs it — surface that as "Rerun" rather than "Run".
  const isRerun = stageHasFailedOrCanceledTasks(stage);

  const runnableTasks = useMemo(
    () =>
      stage.tasks.filter((task) =>
        RUNNABLE_TASK_STATUSES.includes(task.status)
      ),
    [stage.tasks]
  );
  // Polling swaps the stage/rollout objects every tick, giving `runnableTasks` a
  // fresh identity even when the set is unchanged. Key the IAM preload on a
  // stable name string so it doesn't refetch (and flicker permissionReady) each
  // poll — only when the runnable set actually changes.
  const runnableTaskKey = runnableTasks.map((task) => task.name).join("|");

  // Gate the header action on the same permission the Deploy section enforces.
  useEffect(() => {
    let canceled = false;
    const load = async () => {
      setPermissionReady(false);
      await preloadRolloutPermissionContext({
        environment: stage.environment,
        projectName: project.name,
        tasks: runnableTasks,
      });
      if (!canceled) setPermissionReady(true);
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [project.name, runnableTaskKey, stage.environment]);

  // Wait for the permission check so we never flash an action a user can't take.
  if (!permissionReady) {
    return null;
  }

  const canRun = canRolloutTasks({
    currentUser,
    environment: stage.environment,
    issue: page.issue,
    project,
    tasks: runnableTasks,
  });

  // Not actionable → show the frontier status, not an action (consistent with
  // the review phase, and with the Deploy section's stage status display).
  if (!canRun) {
    return <FrontierStatusStamp stage={stage} />;
  }

  const runVerb = isRerun ? t("plan.lifecycle.rerun") : t("plan.lifecycle.run");

  return (
    <>
      <Button
        className="max-w-48 shrink-0 gap-x-1.5"
        onClick={() => setOpen(true)}
      >
        <Play className="size-4 shrink-0" />
        <span className="shrink-0">{runVerb}</span>
        <StageSeparator />
        <span className="truncate" title={title}>
          {title}
        </span>
      </Button>
      <PlanDetailTaskRolloutActionPanel
        action="RUN"
        onConfirm={async () => {
          await page.refreshState();
          // Land on the stage that was run: switch the deploy selection to it if
          // another stage is currently selected, then bring the phase into view.
          const stageId = stage.name.split("/").pop();
          if (stageId && page.routeStageId && page.routeStageId !== stageId) {
            void router.push({ query: { phase: "deploy", stageId } });
          }
          focusPlanPhase("deploy", page.expandPhase);
        }}
        onOpenChange={setOpen}
        open={open}
        target={{ stage, type: "tasks" }}
      />
    </>
  );
}

// The frontier stage's status, rendered exactly as the Deploy section renders a
// stage: the canonical TaskStatusIcon for the computed getStageStatus, with the
// same status word (stringifyTaskStatus). Tinted red only on failure. Shown
// while the stage is running (nobody can act) or when the current user can't run
// it — a status, never a disabled action.
export function FrontierStatusStamp({ stage }: { stage: Stage }) {
  const { t } = useTranslation();
  const title = useStageTitle(stage);
  const status = getStageStatus(stage);
  const statusText = stringifyTaskStatus(status, t);

  return (
    <LifecycleStamp
      className="max-w-48"
      size="md"
      tone={status === Task_Status.FAILED ? "error" : "neutral"}
    >
      <TaskStatusIcon size="tiny" status={status} />
      <span className="shrink-0">{statusText}</span>
      <StageSeparator />
      <span className="truncate" title={title}>
        {title}
      </span>
    </LifecycleStamp>
  );
}
