import Emittery from "emittery";
import { first, isEqual } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { emptyStage, emptyTask } from "@/types";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  activeStageInRollout,
  activeTaskInStageV1,
  extractStageUID,
  extractTaskUID,
  flattenTaskV1List,
  indexOrUIDFromSlug,
  stageV1Slug,
  taskV1Slug,
  uidFromSlug,
} from "@/utils";
import type { IssueContext, IssueEvents } from "./context";
import { planCheckRunListForTask } from "./plan-check";
import { stageForTask } from "./utils";

const state = {
  uid: -101,
};
export const nextUID = () => {
  return String(state.uid--);
};

export const useBaseIssueContext = (
  context: Pick<IssueContext, "isCreating" | "ready" | "issue">
): Partial<IssueContext> => {
  const { isCreating, issue } = context;
  const route = useRoute();
  const router = useRouter();

  const events: IssueEvents = new Emittery();

  const rollout = computed(() => issue.value.rolloutEntity);
  const tasks = computed(() => flattenTaskV1List(rollout.value));

  const selectedStage = computed((): Stage => {
    const stageSlug = route.query.stage as string;
    const taskSlug = route.query.task as string;
    const stageList = rollout.value?.stages || [];

    // Stage slug is now always the environment ID
    if (stageSlug) {
      const stageFound = stageList.find(
        (stage) => extractStageUID(stage.name) === stageSlug
      );
      if (stageFound) {
        return stageFound;
      }
    } else if (taskSlug) {
      const taskUID = String(uidFromSlug(taskSlug));
      for (const stage of stageList) {
        if (
          stage.tasks.findIndex(
            (task) => extractTaskUID(task.name) === taskUID
          ) >= 0
        ) {
          return stage;
        }
      }
    }

    // fallback
    if (isCreating.value) {
      return first(stageList) ?? emptyStage();
    }
    return activeStageInRollout(rollout.value);
  });
  const selectedTask = computed((): Task => {
    const taskSlug = route.query.task as string;
    const { tasks } = selectedStage.value;
    // Index is used when `isCreating === true`
    // UID is used when otherwise
    if (taskSlug) {
      const indexOrUID = indexOrUIDFromSlug(taskSlug);
      if (isCreating.value) {
        if (indexOrUID < tasks.length) {
          return tasks[indexOrUID];
        }
      } else {
        const taskFound = tasks.find(
          (task) => extractTaskUID(task.name) === String(indexOrUID)
        );
        if (taskFound) {
          return taskFound;
        }
      }
    }

    // fallback
    if (isCreating.value) {
      return first(tasks) ?? emptyTask();
    }
    return activeTaskInStageV1(selectedStage.value);
  });

  events.on("select-task", ({ task }) => {
    const stage = stageForTask(issue.value, task);
    if (!stage) return;
    // Always use stage slug (environment ID)
    const stageParam = stageV1Slug(stage);
    const taskParam = isCreating.value
      ? String(stage.tasks.indexOf(task))
      : taskV1Slug(task);

    router.replace({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      query: {
        ...route.query,
        stage: stageParam,
        task: taskParam,
      },
      hash: route.hash,
    });
  });

  const planCheckRunMapForTask = ref<Map<string, PlanCheckRun[]>>(new Map());

  const getPlanCheckRunsForTask = (task: Task) => {
    return planCheckRunMapForTask.value.get(task.name) || [];
  };

  watch(
    () => issue.value,
    (now, old) => {
      // Only recompute planCheckRunMapForTask when planCheckRunList changes.
      if (isEqual(old?.planCheckRunList, now.planCheckRunList)) return;
      const cacheMap = new Map<string, PlanCheckRun[]>();
      for (const task of tasks.value) {
        cacheMap.set(task.name, planCheckRunListForTask(issue.value, task));
      }
      planCheckRunMapForTask.value = cacheMap;
    },
    {
      deep: true,
      immediate: true,
    }
  );

  return {
    events,
    selectedStage,
    selectedTask,
    getPlanCheckRunsForTask,
  };
};
