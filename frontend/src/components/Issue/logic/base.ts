import { computed, Ref } from "vue";
import { isEmpty, isUndefined } from "lodash-es";
import {
  Issue,
  IssueCreate,
  Project,
  Stage,
  StageCreate,
  StageId,
  Task,
  TaskCreate,
  TaskStatus,
} from "@/types";
import { useRoute, useRouter } from "vue-router";
import {
  activeStage,
  activeTaskInStage,
  idFromSlug,
  indexFromSlug,
  issueSlug,
  stageSlug,
  taskSlug,
} from "@/utils";
import { useDatabaseStore, useIssueStore, useProjectStore } from "@/store";
import {
  flattenTaskList,
  sheetIdOfTask,
  statementOfTask,
  TaskTypeWithSheetId,
  TaskTypeWithStatement,
} from "./common";

export const useBaseIssueLogic = (params: {
  create: Ref<boolean>;
  issue: Ref<Issue | IssueCreate>;
}) => {
  const { create, issue } = params;
  const route = useRoute();
  const router = useRouter();
  const issueStore = useIssueStore();
  const projectStore = useProjectStore();
  const databaseStore = useDatabaseStore();

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
    const stageSlug = route.query.stage as string;
    const taskSlug = route.query.task as string;
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
        ...route.query,
        stage: stageSlug(stageList[index].name, index),
        task: taskSlug,
      },
      hash: route.hash,
    });
  };

  const selectTask = (task: Task) => {
    if (create.value) return;

    // Find the stage which the task belongs to
    const stage = (issue.value as Issue).pipeline?.stageList.find(
      (s) => s.taskList.findIndex((t) => t.id === task.id) >= 0
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

  const selectedDatabase = computed(() => {
    if (create.value) {
      const databaseId = (selectedTask.value as TaskCreate).databaseId;
      if (databaseId) {
        return databaseStore.getDatabaseById(databaseId);
      }
      return undefined;
    }
    return (selectedTask.value as Task).database;
  });

  const isGhostMode = computed((): boolean => {
    return issue.value.type === "bb.issue.database.schema.update.ghost";
  });

  const isTenantMode = computed((): boolean => {
    if (project.value.tenantMode !== "TENANT") return false;

    // We support single database migration in tenant mode projects.
    // So a pipeline should be tenant mode when it contains more
    // than one tasks.
    return (
      flattenTaskList(issue.value).filter((task) =>
        TaskTypeWithStatement.includes(task.type)
      ).length > 1
    );
  });

  const isPITRMode = computed((): boolean => {
    const { type } = issue.value;
    return (
      type === "bb.issue.database.restore.pitr" ||
      type === "bb.issue.database.create"
    );
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

  const selectedStatement = computed((): string => {
    const task = selectedTask.value;
    if (create.value) {
      return (task as TaskCreate).statement;
    }

    // Extract statement from different types of payloads
    return statementOfTask(task as Task) || "";
  });

  const allowApplyTaskStateToOthers = computed(() => {
    if (!create.value) {
      return false;
    }
    const taskList = flattenTaskList<TaskCreate>(issue.value);
    // Allowed when more than one tasks need SQL statement or sheet.
    const count = taskList.filter(
      (task) =>
        TaskTypeWithStatement.includes(task.type) ||
        TaskTypeWithSheetId.includes(task.type)
    ).length;

    return count > 1;
  });

  const allowApplyIssueStatusTransition = () => {
    // no extra logic by default
    return true;
  };

  const allowApplyTaskStatusTransition = (task: Task, to: TaskStatus) => {
    if (to === "CANCELED") {
      // All task types are not CANCELable by default.
      // Might be overwritten by other issue logic providers.
      return false;
    }

    // no extra logic by default
    return true;
  };

  const applyTaskStateToOthers = (task: Task) => {
    const taskList = flattenTaskList<TaskCreate>(issue.value);
    const sheetId = sheetIdOfTask(task);
    const statement = statementOfTask(task);

    for (const taskItem of taskList) {
      if (TaskTypeWithStatement.includes(taskItem.type)) {
        if (!isUndefined(sheetId)) {
          taskItem.sheetId = sheetId;
        } else {
          taskItem.statement = statement ?? "";
        }
      }
    }
  };

  return {
    project,
    isTenantMode,
    isGhostMode,
    isPITRMode,
    createIssue,
    selectedStage,
    selectedTask,
    selectedDatabase,
    selectStageOrTask,
    selectTask,
    taskStatusOfStage,
    isValidStage,
    selectedStatement,
    allowApplyTaskStateToOthers,
    applyTaskStateToOthers,
    allowApplyIssueStatusTransition,
    allowApplyTaskStatusTransition,
  };
};
