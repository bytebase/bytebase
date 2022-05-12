import { computed, Ref } from "vue";
import { isEmpty } from "lodash-es";
import {
  Issue,
  IssueCreate,
  Project,
  Stage,
  StageCreate,
  StageId,
  Task,
  TaskCreate,
} from "@/types";
import { useRoute, useRouter } from "vue-router";
import {
  activeStage,
  activeTaskInStage,
  idFromSlug,
  indexFromSlug,
  isDev,
  issueSlug,
  stageSlug,
  taskSlug,
} from "@/utils";
import { useIssueStore, useProjectStore } from "@/store";
import { TaskTypeWithStatement } from "./common";

export const useBaseIssueLogic = (params: {
  create: Ref<boolean>;
  issue: Ref<Issue | IssueCreate>;
}) => {
  const { create, issue } = params;
  const route = useRoute();
  const router = useRouter();
  const issueStore = useIssueStore();
  const projectStore = useProjectStore();

  const project = computed((): Project => {
    if (create.value) {
      return projectStore.getProjectById(
        (issue.value as IssueCreate).projectId
      );
    }
    return (issue.value as Issue).project;
  });

  const createIssue = (issue: IssueCreate) => {
    // Set issue.pipeline and issue.payload to empty
    // because we are no longer passing parameters via issue.pipeline
    // we are using issue.createContext instead
    delete issue.pipeline;
    issue.payload = {};

    issueStore.createIssue(issue).then((createdIssue) => {
      // Use replace to omit the new issue url in the navigation history.
      router.replace(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
    });
  };

  const selectedStage = computed((): Stage | StageCreate => {
    const stageSlug = router.currentRoute.value.query.stage as string;
    const taskSlug = router.currentRoute.value.query.task as string;
    // For stage slug, we support both index based and id based.
    // Index based is used when creating the new task and is the one used when clicking the UI.
    // Id based is used when the context only has access to the stage id (e.g. Task only contains StageId)
    if (stageSlug) {
      const index = indexFromSlug(stageSlug);
      if (index < issue.value.pipeline!.stageList.length) {
        return issue.value.pipeline!.stageList[index];
      }
      const stageId = idFromSlug(stageSlug);
      const stageList = (issue.value as Issue).pipeline.stageList;
      for (const stage of stageList) {
        if (stage.id == stageId) {
          return stage;
        }
      }
    } else if (!create.value && taskSlug) {
      const taskId = idFromSlug(taskSlug);
      const stageList = (issue.value as Issue).pipeline.stageList;
      for (const stage of stageList) {
        for (const task of stage.taskList) {
          if (task.id == taskId) {
            return stage;
          }
        }
      }
    }
    if (create.value) {
      return issue.value.pipeline!.stageList[0];
    }
    return activeStage((issue.value as Issue).pipeline);
  });

  const selectStageOrTask = (
    stageId: StageId,
    taskSlug: string | undefined = undefined
  ) => {
    const stageList = issue.value.pipeline!.stageList;
    const index = stageList.findIndex((item, index) => {
      if (create.value) {
        return index === stageId;
      }
      return (item as Stage).id == stageId;
    });
    router.replace({
      name: "workspace.issue.detail",
      query: {
        ...router.currentRoute.value.query,
        stage: stageSlug(stageList[index].name, index),
        task: taskSlug,
      },
    });
  };

  const selectTask = (task: Task) => {
    if (!create.value) return;

    const stage = (issue.value as Issue).pipeline?.stageList.find(
      (t) => t.id === task.id
    );
    if (!stage) {
      return;
    }
    const slug = taskSlug(task.name, task.id);
    selectStageOrTask(stage.id, slug);
  };

  const selectedTask = computed((): Task | TaskCreate => {
    const taskSlug = route.query.task as string;
    const { taskList } = selectedStage.value;
    if (taskSlug) {
      const index = indexFromSlug(taskSlug);
      if (index < taskList.length) {
        return taskList[index];
      }
      const id = idFromSlug(taskSlug);
      for (let i = 0; i < taskList.length; i++) {
        const task = taskList[i] as Task;
        if (task.id === id) {
          return task;
        }
      }
    }
    return taskList[0];
  });

  const isTenantMode = computed((): boolean => {
    if (project.value.tenantMode !== "TENANT") return false;
    return (
      issue.value.type === "bb.issue.database.schema.update" ||
      issue.value.type === "bb.issue.database.data.update"
    );
  });

  const isGhostMode = computed((): boolean => {
    if (!isDev()) return false;

    return issue.value.type === "bb.issue.database.schema.update.ghost";
  });

  const taskStatusOfStage = (stage: Stage | StageCreate) => {
    if (create.value) {
      return stage.taskList[0].status;
    }
    const activeTask = activeTaskInStage(stage as Stage);
    return activeTask.status;
  };

  const isValidStage = (stage: Stage | StageCreate) => {
    if (!create.value) {
      return true;
    }

    for (const task of stage.taskList) {
      if (TaskTypeWithStatement.includes(task.type)) {
        if (isEmpty((task as TaskCreate).statement)) {
          return false;
        }
      }
    }
    return true;
  };

  return {
    project,
    isTenantMode,
    isGhostMode,
    createIssue,
    selectedStage,
    selectedTask,
    selectStageOrTask,
    selectTask,
    taskStatusOfStage,
    isValidStage,
  };
};
