import Emittery from "emittery";
import { first } from "lodash-es";
import { useDialog } from "naive-ui";
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useUIStateStore } from "@/store";
import {
  EMPTY_ID,
  emptyStage,
  emptyTask,
  TaskTypeListWithStatement,
} from "@/types";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import { Task_Type, Stage, Task } from "@/types/proto/v1/rollout_service";
import { emptyPlanSpec } from "@/types/v1/issue/plan";
import {
  activeStageInRollout,
  activeTaskInRollout,
  activeTaskInStageV1,
  flattenTaskV1List,
  uidFromSlug,
  indexOrUIDFromSlug,
  stageV1Slug,
  taskV1Slug,
  flattenSpecList,
} from "@/utils";
import type { IssueContext, IssueEvents, IssuePhase } from "./context";
import { releaserCandidatesForIssue } from "./releaser";
import { extractReviewContext } from "./review";
import { specForTask, stageForTask } from "./utils";

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

  const plan = computed(() => issue.value.planEntity);
  const rollout = computed(() => issue.value.rolloutEntity);
  const project = computed(() => issue.value.projectEntity);
  const specs = computed(() => flattenSpecList(plan.value));

  const activeStage = computed((): Stage => {
    return activeStageInRollout(rollout.value);
  });
  const activeTask = computed((): Task => {
    return activeTaskInRollout(rollout.value);
  });

  const selectedSpec = computed((): Plan_Spec => {
    // Check if spec is selected from URL. (Not use yet)
    const specSlug = route.query.spec as string;
    if (specSlug) {
      const indexOrId = indexOrUIDFromSlug(specSlug);
      if (isCreating.value) {
        if (indexOrId < specs.value.length) {
          return specs.value[indexOrId];
        }
      } else {
        const specFound = specs.value.find(
          (spec) => spec.id === String(indexOrId)
        );
        if (specFound) {
          return specFound;
        }
      }
    }

    // Otherwise, fallback to selected task's spec.
    if (selectedTask.value && selectedTask.value.uid !== String(EMPTY_ID)) {
      return specForTask(plan.value, selectedTask.value) || emptyPlanSpec();
    }
    // Fallback to first spec.
    return first(specs.value) || emptyPlanSpec();
  });
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
          (stage) => stage.uid === String(indexOrUID)
        );
        if (stageFound) {
          return stageFound;
        }
      }
    } else if (!isCreating.value && taskSlug) {
      const taskUID = String(uidFromSlug(taskSlug));
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
  const isTenantMode = computed((): boolean => {
    // To sync databases schema in tenant mode, we use normal project logic to create issue.
    if (isCreating.value && route.query.batch !== "1") return false;
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
  const isLegacyIssue = computed(() => {
    return !issue.value.plan && !issue.value.planEntity;
  });
  const formatOnSave = computed({
    get: () => uiStateStore.editorFormatStatementOnSave,
    set: (value: boolean) => uiStateStore.setEditorFormatStatementOnSave(value),
  });

  return {
    phase,
    isGhostMode,
    isTenantMode,
    isLegacyIssue,
    events,
    releaserCandidates,
    reviewContext,
    activeStage,
    activeTask,
    selectedStage,
    selectedTask,
    selectedSpec,
    formatOnSave,
    dialog,
  };
};
