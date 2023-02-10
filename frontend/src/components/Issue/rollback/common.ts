import { computed } from "vue";
import { head } from "lodash-es";

import type {
  Database,
  Issue,
  IssueCreate,
  MigrationContext,
  Task,
  TaskCreate,
  TaskDatabaseDataUpdatePayload,
} from "@/types";
import {
  hasProjectPermission,
  hasWorkspacePermission,
  isTaskCreate,
  isTaskSkipped,
  semverCompare,
} from "@/utils";
import { useIssueLogic } from "../logic";
import { useCurrentUser, useDatabaseStore } from "@/store";

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
        const migrationDetail = head(createContext.detailList);
        if (migrationDetail) {
          migrationDetail.rollbackEnabled = on;
        }
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
