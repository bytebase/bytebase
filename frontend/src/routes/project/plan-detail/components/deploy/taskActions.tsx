import { useEffect, useMemo, useState } from "react";
import { useLatestRef } from "@/hooks/useLatestRef";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  canRolloutTasks,
  preloadRolloutPermissionContext,
} from "../../../issue-detail/utils/rollout";
import {
  type DeployTaskActionState,
  getDeployTaskActionState,
} from "./taskActionState";

// Page slices arrive as params (not via context) so the memoized card that
// calls this hook re-renders only when one of them actually changes.
export const useDeployTaskActions = ({
  currentUser,
  issue,
  project,
  environment,
  task,
}: {
  currentUser: User;
  issue?: Issue;
  project: Project;
  environment?: string;
  task: Task;
}): DeployTaskActionState & { permissionReady: boolean } => {
  const [permissionReady, setPermissionReady] = useState(false);

  // The permission context is per project + environment — the helpers below
  // never read task fields — so the effect and memo key on those stable
  // strings and read the latest task through this ref instead of re-running
  // when the task identity changes. Taking `environment` (not the whole stage
  // object, which the snapshot gate rebuilds every poll tick) keeps the memo
  // from recomputing on ticks that only touched a sibling task.
  const taskRef = useLatestRef(task);

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      setPermissionReady(false);
      await preloadRolloutPermissionContext({
        environment,
        projectName: project.name,
        tasks: [taskRef.current],
      });
      if (!canceled) {
        setPermissionReady(true);
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [project.name, environment]);

  const canPerformActions = useMemo(() => {
    if (!permissionReady || environment === undefined) {
      return false;
    }
    return canRolloutTasks({
      currentUser,
      environment,
      issue,
      project,
      tasks: [taskRef.current],
    });
  }, [currentUser, issue, permissionReady, project, environment]);

  return {
    ...getDeployTaskActionState({
      canPerformActions,
      status: task.status,
    }),
    permissionReady,
  };
};
