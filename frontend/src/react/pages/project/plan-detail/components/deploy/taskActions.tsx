import { useEffect, useMemo, useState } from "react";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  canRolloutTasks,
  preloadRolloutPermissionContext,
} from "../../../issue-detail/utils/rollout";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import {
  type DeployTaskActionState,
  getDeployTaskActionState,
} from "./taskActionState";

export const useDeployTaskActions = ({
  stage,
  task,
}: {
  stage?: Stage;
  task: Task;
}): DeployTaskActionState & { permissionReady: boolean } => {
  const page = usePlanDetailContext();
  const currentUser = page.currentUser;
  const project = page.project;
  const [permissionReady, setPermissionReady] = useState(false);

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      setPermissionReady(false);
      await preloadRolloutPermissionContext({
        environment: stage?.environment,
        projectName: project.name,
        tasks: [task],
      });
      if (!canceled) {
        setPermissionReady(true);
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [project.name, stage?.environment, task]);

  const canPerformActions = useMemo(() => {
    if (!permissionReady || !stage) {
      return false;
    }
    return canRolloutTasks({
      currentUser,
      environment: stage.environment,
      issue: page.issue,
      project,
      tasks: [task],
    });
  }, [currentUser, page.issue, permissionReady, project, stage, task]);

  return {
    ...getDeployTaskActionState({
      canPerformActions,
      status: task.status,
    }),
    permissionReady,
  };
};
