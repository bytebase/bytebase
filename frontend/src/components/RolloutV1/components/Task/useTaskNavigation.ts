import { useRouter } from "vue-router";
import { buildTaskDetailRoute } from "@/router/dashboard/projectV1RouteHelpers";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";

export const useTaskNavigation = () => {
  const router = useRouter();

  const navigateToTaskDetail = (task: Task) => {
    router.push(buildTaskDetailRoute(task.name));
  };

  return {
    navigateToTaskDetail,
  };
};
