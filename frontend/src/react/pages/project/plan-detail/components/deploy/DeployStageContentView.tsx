import { create } from "@bufbuild/protobuf";
import { Play, Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import { pushNotification } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import {
  CreateRolloutRequestSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  canRolloutTasks,
  preloadRolloutPermissionContext,
  RUNNABLE_TASK_STATUSES,
} from "../../../issue-detail/utils/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { PlanDetailTaskRolloutActionPanel } from "../PlanDetailTaskRolloutActionPanel";
import { DeployStageRollbackSection } from "./DeployStageRollbackSection";
import { DeployTaskFilter } from "./DeployTaskFilter";
import { DeployTaskList } from "./DeployTaskList";

export function DeployStageContentView({
  selectedTaskName,
  stage,
}: {
  selectedTaskName?: string;
  stage: Stage;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = page.currentUser;
  const project = page.project;
  const [filterStatuses, setFilterStatuses] = useState<Task_Status[]>([]);
  const [rolloutPermissionReady, setRolloutPermissionReady] = useState(false);
  const [runStageOpen, setRunStageOpen] = useState(false);
  const stageTaskKey = stage.tasks.map((task) => task.name).join("|");

  const isStageCreated = stage.tasks.length > 0;

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      if (!isStageCreated) {
        setRolloutPermissionReady(true);
        return;
      }
      setRolloutPermissionReady(false);
      await preloadRolloutPermissionContext({
        environment: stage.environment,
        projectName: project.name,
        tasks: stage.tasks,
      });
      if (!canceled) setRolloutPermissionReady(true);
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [isStageCreated, project.name, stage.environment, stageTaskKey]);

  const canRunStage =
    rolloutPermissionReady &&
    isStageCreated &&
    stage.tasks.some((task) => RUNNABLE_TASK_STATUSES.includes(task.status)) &&
    canRolloutTasks({
      currentUser,
      environment: stage.environment,
      issue: page.issue,
      project,
      tasks: stage.tasks,
    });

  const filteredTasks =
    filterStatuses.length > 0
      ? stage.tasks.filter((task) => filterStatuses.includes(task.status))
      : stage.tasks;

  return (
    <div className="w-full">
      <div className="px-4">
        <div className="flex w-full flex-row items-center justify-between gap-2 py-4">
          <div className="flex flex-row items-center gap-3">
            {isStageCreated && (
              <DeployTaskFilter
                onChange={setFilterStatuses}
                selectedStatuses={filterStatuses}
                stage={stage}
              />
            )}
          </div>

          <div className="flex shrink-0 items-center gap-x-2">
            {isStageCreated ? (
              <Button
                disabled={!canRunStage}
                size="sm"
                onClick={() => setRunStageOpen(true)}
              >
                <Play className="h-4 w-4" />
                {t("rollout.stage.run-stage")}
              </Button>
            ) : (
              <Button
                onClick={async () => {
                  const confirmed = window.confirm(
                    t("rollout.stage.confirm-create")
                  );
                  if (!confirmed) return;
                  try {
                    await rolloutServiceClientConnect.createRollout(
                      create(CreateRolloutRequestSchema, {
                        parent: page.plan.name,
                        target: stage.environment,
                      })
                    );
                    pushNotification({
                      module: "bytebase",
                      style: "SUCCESS",
                      title: t("common.created"),
                    });
                    await page.refreshState();
                  } catch (error) {
                    pushNotification({
                      module: "bytebase",
                      style: "CRITICAL",
                      title: t("common.error"),
                      description: String(error),
                    });
                  }
                }}
                size="sm"
              >
                <Plus className="h-4 w-4" />
                {t("common.create")}
              </Button>
            )}
          </div>
        </div>
      </div>

      <div className="flex flex-col">
        <DeployTaskList
          readonly={!isStageCreated}
          selectedTaskName={selectedTaskName}
          stage={{
            ...stage,
            tasks: filteredTasks,
          }}
        />

        {isStageCreated && <DeployStageRollbackSection stage={stage} />}
      </div>

      <PlanDetailTaskRolloutActionPanel
        action="RUN"
        onConfirm={async () => {
          await page.refreshState();
        }}
        onOpenChange={setRunStageOpen}
        open={runStageOpen}
        target={{ type: "tasks", stage }}
      />
    </div>
  );
}
