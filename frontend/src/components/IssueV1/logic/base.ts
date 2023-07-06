import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import Emittery from "emittery";
import { first } from "lodash-es";

import { IssueContext } from "./context";
import { Stage, Task } from "@/types/proto/v1/rollout_service";
import {
  activeStageInRollout,
  activeTaskInRollout,
  activeTaskInStageV1,
  idFromSlug,
  indexOrUIDFromSlug,
  stageV1Slug,
  taskV1Slug,
} from "@/utils";
import { emptyStage, emptyTask } from "@/types";
import { extractReviewContext } from "./review";

export const useBaseIssueContext = (
  context: Pick<IssueContext, "isCreating" | "ready" | "issue">
): Partial<IssueContext> => {
  const { isCreating, issue } = context;
  const route = useRoute();
  const router = useRouter();
  const events: IssueContext["events"] = new Emittery();

  const activeStage = computed((): Stage => {
    return activeStageInRollout(issue.value.rolloutEntity);
  });
  const activeTask = computed((): Task => {
    return activeTaskInRollout(issue.value.rolloutEntity);
  });

  const selectedStage = computed((): Stage => {
    const stageSlug = route.query.stage as string;
    const taskSlug = route.query.task as string;
    const rollout = issue.value.rolloutEntity;
    const stageList = rollout.stages;

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

    return activeStageInRollout(rollout);
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
    const stages = issue.value.rolloutEntity.stages;
    const stage = stages.find(
      (stage) => stage.tasks.findIndex((t) => t === task) >= 0
    );
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

  return {
    events,
    reviewContext,
    activeStage,
    activeTask,
    selectedStage,
    selectedTask,
  };
};
