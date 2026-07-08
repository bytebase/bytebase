import { create } from "@bufbuild/protobuf";
import { Plus } from "lucide-react";
import { type ComponentProps, memo, useMemo, useState } from "react";
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

// One instance per stage, all kept mounted; the parent hides the inactive
// ones. Because an instance is permanently bound to its stage, per-stage view
// state (the status filter here, selection/expansion in the list) is scoped by
// construction — a filter can never leak onto another stage's tasks
// (BYT-9762), and it survives switching away and back.
//
// memo: with stage identity stable across renders (identity-preserving
// snapshot), parent-local re-renders in DeployBranch (e.g. the optimistic tab
// highlight) skip this whole subtree. Context changes still propagate to the
// consumers below regardless of memo.
export const DeployStageContentView = memo(function DeployStageContentView({
  stage,
  active = true,
  onOpenedTaskChange,
  selfWrittenTaskRef,
}: {
  stage: Stage;
  active?: boolean;
  onOpenedTaskChange?: ComponentProps<
    typeof DeployTaskList
  >["onOpenedTaskChange"];
  selfWrittenTaskRef?: ComponentProps<
    typeof DeployTaskList
  >["selfWrittenTaskRef"];
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [filterStatuses, setFilterStatuses] = useState<Task_Status[]>([]);

  const isStageCreated = stage.tasks.length > 0;

  // Pass `stage` through untouched when no filter is active (the common case)
  // and memoize the filtered variant — a fresh stage object every render would
  // re-render the whole task list for nothing.
  const filteredStage = useMemo(
    () =>
      filterStatuses.length > 0
        ? {
            ...stage,
            tasks: stage.tasks.filter((task) =>
              filterStatuses.includes(task.status)
            ),
          }
        : stage,
    [filterStatuses, stage]
  );

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
          active={active}
          onOpenedTaskChange={onOpenedTaskChange}
          readonly={!isStageCreated}
          selfWrittenTaskRef={selfWrittenTaskRef}
          stage={filteredStage}
        />

        {isStageCreated && <DeployStageRollbackSection stage={stage} />}
      </div>
    </div>
  );
});
