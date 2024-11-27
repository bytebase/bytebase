import Emittery from "emittery";
import { first, isEqual } from "lodash-es";
import { useDialog } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useUIStateStore } from "@/store";
import { emptyStage, emptyTask } from "@/types";
import type { PlanCheckRun } from "@/types/proto/v1/plan_service";
import { Task_Type, Stage, Task } from "@/types/proto/v1/rollout_service";
import {
  activeStageInRollout,
  activeTaskInStageV1,
  flattenTaskV1List,
  uidFromSlug,
  indexOrUIDFromSlug,
  stageV1Slug,
  taskV1Slug,
  extractTaskUID,
  extractStageUID,
} from "@/utils";
import type { IssueContext, IssueEvents, IssuePhase } from "./context";
import { planCheckRunListForTask } from "./plan-check";
import { releaserCandidatesForIssue } from "./releaser";
import { extractReviewContext } from "./review";
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
  const uiStateStore = useUIStateStore();
  const route = useRoute();
  const router = useRouter();
  const dialog = useDialog();

  const events: IssueEvents = new Emittery();

  const rollout = computed(() => issue.value.rolloutEntity);
  const tasks = computed(() => flattenTaskV1List(rollout.value));

  const selectedStage = computed((): Stage => {
    const stageSlug = route.query.stage as string;
    const taskSlug = route.query.task as string;
    const stageList = rollout.value?.stages || [];

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
          (stage) => extractStageUID(stage.name) === String(indexOrUID)
        );
        if (stageFound) {
          return stageFound;
        }
      }
    } else if (!isCreating.value && taskSlug) {
      const taskUID = String(uidFromSlug(taskSlug));
      for (const stage of stageList) {
        if (
          stage.tasks.findIndex((task) => extractTaskUID(task.name) === taskUID)
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
    const stages = rollout.value?.stages || [];
    const stage = stageForTask(issue.value, task);
    if (!stage) return;
    const stageParam = isCreating.value
      ? String(stages.indexOf(stage))
      : stageV1Slug(stage);
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

  const releaserCandidates = computed(() => {
    return releaserCandidatesForIssue(issue.value);
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
  const isLegacyIssue = computed(() => {
    return !issue.value.plan && !issue.value.planEntity;
  });
  const formatOnSave = computed({
    get: () => uiStateStore.editorFormatStatementOnSave,
    set: (value: boolean) => uiStateStore.setEditorFormatStatementOnSave(value),
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
    phase,
    isGhostMode,
    isLegacyIssue,
    events,
    releaserCandidates,
    reviewContext,
    selectedStage,
    selectedTask,
    formatOnSave,
    dialog,
    getPlanCheckRunsForTask,
  };
};
