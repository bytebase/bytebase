import { create } from "@bufbuild/protobuf";
import { Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import { pushNotification } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import {
  CreateRolloutRequestSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
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
  const [filterStatuses, setFilterStatuses] = useState<Task_Status[]>([]);
  // Scope the task filter to the stage: when switching to a different stage,
  // drop the previous stage's filter so it can't silently hide the new stage's
  // tasks (BYT-9762). Resetting during render — rather than in a post-paint
  // effect — keeps the new stage's first paint correct instead of briefly
  // showing the wrong filtered set.
  const [filteredStageName, setFilteredStageName] = useState(stage.name);
  if (stage.name !== filteredStageName) {
    setFilteredStageName(stage.name);
    setFilterStatuses([]);
  }

  const isStageCreated = stage.tasks.length > 0;

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

          {/* The bulk "run this stage" advance now lives in the page header's
              lifecycle slot (Run · {stage}, frontier-only — BYT-9722), so this
              row no longer duplicates it. "Create" is a separate action for a
              stage that doesn't exist in the rollout yet; per-task actions and
              the multi-select toolbar below still cover ad hoc runs. */}
          {!isStageCreated && (
            <div className="flex shrink-0 items-center gap-x-2">
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
            </div>
          )}
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
    </div>
  );
}
