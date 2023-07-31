import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { first } from "lodash-es";

import { IssueContext, IssuePhase } from "./context";
import { Stage, Task, Task_Type } from "@/types/proto/v1/rollout_service";
import {
  activeStageInRollout,
  activeTaskInRollout,
  activeTaskInStageV1,
  flattenTaskV1List,
  idFromSlug,
  indexOrUIDFromSlug,
  stageV1Slug,
  taskV1Slug,
} from "@/utils";
import { emptyStage, emptyTask, TaskTypeListWithStatement } from "@/types";
import { extractReviewContext } from "./review";
import { TenantMode } from "@/types/proto/v1/project_service";
import { stageForTask } from "./utils";

const state = {
  uid: -101,
};
export const nextUID = () => {
  return String(state.uid--);
};

export const useBaseIssueContext = (
  context: Pick<IssueContext, "isCreating" | "ready" | "issue" | "events">
): Partial<IssueContext> => {
  const { isCreating, issue, events } = context;
  const route = useRoute();
  const router = useRouter();

  const rollout = computed(() => issue.value.rolloutEntity);
  const project = computed(() => issue.value.projectEntity);

  const activeStage = computed((): Stage => {
    return activeStageInRollout(rollout.value);
  });
  const activeTask = computed((): Task => {
    return activeTaskInRollout(rollout.value);
  });

  const selectedStage = computed((): Stage => {
    const stageSlug = route.query.stage as string;
    const taskSlug = route.query.task as string;
    const stageList = rollout.value.stages;

    // Index is used when `isCreating === true`
    // UID is used when otherwise
    if (stageSlug) {
      const indexOrUID = indexOrUIDFromSlug(stageSlug);
      if (isCreating.value) {
        if (indexOrUID < stageList.length) {
          return stageList[indexOrUID];
        }
      } else {
        const stageFound = stageList.find(
          (stage) => stage.uid === String(indexOrUID)
        );
        if (stageFound) {
          return stageFound;
        }
      }
    } else if (!isCreating.value && taskSlug) {
      const taskUID = String(idFromSlug(taskSlug));
      for (const stage of stageList) {
        if (stage.tasks.findIndex((task) => task.uid === taskUID)) {
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
        const taskFound = tasks.find((task) => task.uid === String(indexOrUID));
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
    const stages = rollout.value.stages;
    const stage = stageForTask(issue.value, task);
    if (!stage) return;
    const stageParam = isCreating.value
      ? String(stages.indexOf(stage))
      : stageV1Slug(stage);
    const taskParam = isCreating.value
      ? String(stage.tasks.indexOf(task))
      : taskV1Slug(task);

    router.replace({
      name: "workspace.issue.detail.v1",
      query: {
        ...route.query,
        stage: stageParam,
        task: taskParam,
      },
      hash: route.hash,
    });
  });

  const reviewContext = extractReviewContext(issue);

  const phase = computed((): IssuePhase => {
    if (isCreating.value) return "CREATE";

    return reviewContext.done.value ? "ROLLOUT" : "REVIEW";
  });

  const isGhostMode = computed(() => {
    return flattenTaskV1List(rollout.value).some((task) => {
      return [
        Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
        Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER,
      ].includes(task.type);
    });
  });
  const isPITRMode = computed(() => {
    return flattenTaskV1List(rollout.value).some((task) => {
      return [
        Task_Type.DATABASE_RESTORE_RESTORE,
        Task_Type.DATABASE_RESTORE_CUTOVER,
        Task_Type.DATABASE_CREATE,
      ].includes(task.type);
    });
  });
  const isTenantMode = computed((): boolean => {
    // To sync databases schema in tenant mode, we use normal project logic to create issue.
    if (isCreating.value && route.query.mode !== "tenant") return false;
    if (project.value.tenantMode !== TenantMode.TENANT_MODE_ENABLED)
      return false;

    // We support single database migration in tenant mode projects.
    // So a pipeline should be tenant mode when it contains more
    // than one tasks.
    return (
      flattenTaskV1List(rollout.value).filter((task) =>
        TaskTypeListWithStatement.includes(task.type)
      ).length > 1
    );
  });

  return {
    phase,
    isGhostMode,
    isPITRMode,
    isTenantMode,
    events,
    reviewContext,
    activeStage,
    activeTask,
    selectedStage,
    selectedTask,
  };
};
