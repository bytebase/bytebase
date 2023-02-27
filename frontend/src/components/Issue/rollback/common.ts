import { computed } from "vue";
import { head } from "lodash-es";

import {
  ActivityCreate,
  Database,
  Issue,
  IssueCreate,
  IssueId,
  MigrationContext,
  Task,
  TaskCreate,
  TaskDatabaseDataUpdatePayload,
  TaskId,
  UNKNOWN_ID,
} from "@/types";
import {
  hasProjectPermission,
  hasWorkspacePermission,
  isTaskCreate,
  isTaskSkipped,
  semverCompare,
} from "@/utils";
import { flattenTaskList, useIssueLogic } from "../logic";
import {
  useActivityStore,
  useCurrentUser,
  useDatabaseStore,
  useIssueStore,
} from "@/store";

const MIN_ROLLBACK_SQL_MYSQL_VERSION = "5.7.0";

export type RollbackUIType =
  | "SWITCH" // Show a simple checkbox to turn on/off rollback
  | "FULL" // Show featured rollback status
  | "NONE"; // Nothing

export const useRollbackLogic = () => {
  const currentUser = useCurrentUser();
  const context = useIssueLogic();
  const {
    create,
    issue,
    isTenantMode,
    selectedTask: task,
    patchTask,
  } = context;

  // Decide with type of UI should be displayed.
  const rollbackUIType = computed((): RollbackUIType => {
    if (issue.value.type !== "bb.issue.database.data.update") {
      return "NONE";
    }
    if (task.value.type !== "bb.task.database.data.update") {
      return "NONE";
    }
    const database = databaseOfTask(task.value);
    const { engine, engineVersion } = database.instance;
    if (engine !== "MYSQL") {
      return "NONE";
    }
    if (!semverCompare(engineVersion, MIN_ROLLBACK_SQL_MYSQL_VERSION, "gte")) {
      return "NONE";
    }
    if (create.value) {
      return "SWITCH";
    }
    const taskEntity = task.value as Task;
    if (taskEntity.status === "DONE") {
      if (isTaskSkipped(taskEntity)) {
        // Rollback is not available for skipped tasks.
        return "NONE";
      }
      return "FULL";
    }
    if (taskEntity.status === "CANCELED") {
      return "NONE";
    }

    return "SWITCH";
  });

  // Decide whether current user can operate.
  const allowRollback = computed((): boolean => {
    if (rollbackUIType.value === "NONE") {
      return false;
    }

    if (create.value) {
      return true;
    }

    const issueEntity = issue.value as Issue;
    const user = currentUser.value;

    if (user.id === issueEntity.creator.id) {
      // Allowed to the issue creator
      return true;
    }

    if (user.id === issueEntity.assignee.id) {
      // Allowed to the issue assignee
      return true;
    }

    const memberInProject = issueEntity.project.memberList.find(
      (member) => member.principal.id === user.id
    );
    if (
      memberInProject?.role &&
      hasProjectPermission(
        "bb.permission.project.admin-database",
        memberInProject.role
      )
    ) {
      // Allowed to the project owner
      return true;
    }

    if (
      hasWorkspacePermission("bb.permission.workspace.manage-issue", user.role)
    ) {
      // Allowed to DBAs and workspace owners
      return true;
    }
    return false;
  });

  const rollbackEnabled = computed((): boolean => {
    if (create.value) {
      if (isTenantMode.value) {
        // In tenant mode, all tasks share a common MigrationDetail
        const issueCreate = issue.value as IssueCreate;
        const createContext = issueCreate.createContext as MigrationContext;
        const migrationDetail = head(createContext.detailList);
        return migrationDetail?.rollbackEnabled ?? false;
      }
      // In standard mode, every task has a independent TaskCreate with its
      // own rollbackEnabled field.
      const taskCreate = task.value as TaskCreate;
      return taskCreate.rollbackEnabled ?? false;
    } else {
      const taskEntity = task.value as Task;
      const payload = taskEntity.payload as
        | TaskDatabaseDataUpdatePayload
        | undefined;
      return payload?.rollbackEnabled ?? false;
    }
  });

  const toggleRollback = (on: boolean) => {
    if (create.value) {
      if (isTenantMode.value) {
        // In tenant mode, all tasks share a common MigrationDetail
        const issueCreate = issue.value as IssueCreate;
        const createContext = issueCreate.createContext as MigrationContext;
        createContext.detailList.forEach((detail) => {
          detail.rollbackEnabled = on;
        });
      } else {
        // In standard mode, every task has a independent TaskCreate with its
        // own rollbackEnabled field.
        const taskCreate = task.value as TaskCreate;
        taskCreate.rollbackEnabled = on;
      }
    } else {
      // Once the issue has been created, we need to patch the task.
      const taskEntity = task.value as Task;
      patchTask(taskEntity.id, {
        rollbackEnabled: on,
      });
    }
  };

  return {
    rollbackUIType,
    allowRollback,
    rollbackEnabled,
    toggleRollback,
  };
};

const databaseOfTask = (task: Task | TaskCreate): Database => {
  if (isTaskCreate(task)) {
    return useDatabaseStore().getDatabaseById((task as TaskCreate).databaseId!);
  }

  return task.database!;
};

export const maybeCreateBackTraceComment = async (newIssue: Issue) => {
  if (newIssue.type !== "bb.issue.database.data.update") return;
  const params = {
    issueId: UNKNOWN_ID as IssueId,
    taskIdList: [] as TaskId[],
  };
  const taskList = flattenTaskList<Task>(newIssue);
  for (let i = 0; i < taskList.length; i++) {
    const task = taskList[i];
    if (task.type !== "bb.task.database.data.update") continue;
    const payload = task.payload as TaskDatabaseDataUpdatePayload;
    if (!payload) continue;
    if (
      payload.rollbackFromIssueId &&
      payload.rollbackFromIssueId !== UNKNOWN_ID &&
      payload.rollbackFromTaskId &&
      payload.rollbackFromTaskId !== UNKNOWN_ID
    ) {
      if (
        params.issueId !== UNKNOWN_ID &&
        payload.rollbackFromIssueId !== params.issueId
      ) {
        console.warn(
          `Different "rollbackFromIssueId" ${params.issueId} and ${payload.rollbackFromIssueId} found.`
        );
        return;
      }
      params.issueId = payload.rollbackFromIssueId;
      params.taskIdList.push(payload.rollbackFromTaskId);
    }
  }
  if (params.issueId === UNKNOWN_ID || params.taskIdList.length === 0) return;
  const fromIssue = await useIssueStore().getOrFetchIssueById(params.issueId);
  const allFromTaskList = flattenTaskList<Task>(fromIssue);
  const fromTaskMap = new Map(allFromTaskList.map((task) => [task.id, task]));
  const fromTaskList: Task[] = [];
  params.taskIdList.forEach((taskId) => {
    const task = fromTaskMap.get(taskId);
    if (task && task.id !== UNKNOWN_ID) {
      fromTaskList.push(task);
    }
  });
  if (fromTaskList.length === 0) return;

  const comment = [
    `Create issue #${newIssue.id}`,
    "to rollback task",
    fromTaskList.map((task) => `[${task.name}]`).join(","),
  ].join(" ");

  const createActivity: ActivityCreate = {
    type: "bb.issue.comment.create",
    containerId: fromIssue.id,
    comment,
  };
  try {
    await useActivityStore().createActivity(createActivity);
  } catch {
    // do nothing
    // failing to comment to won't be too bad
  }
};
