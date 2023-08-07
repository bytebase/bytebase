import { computed } from "vue";

import { ComposedIssue, UNKNOWN_ID } from "@/types";
import {
  extractUserResourceName,
  flattenTaskV1List,
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
  semverCompare,
} from "@/utils";
import { useCurrentUserV1, experimentalFetchIssueByName } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  databaseForTask,
  notifyNotEditableLegacyIssue,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { Task, Task_Status, Task_Type } from "@/types/proto/v1/rollout_service";

const MIN_ROLLBACK_SQL_MYSQL_VERSION = "5.7.0";

export type RollbackUIType =
  | "SWITCH" // Show a simple checkbox to turn on/off rollback
  | "FULL" // Show featured rollback status
  | "NONE"; // Nothing

export const useRollbackContext = () => {
  const currentUserV1 = useCurrentUserV1();
  const context = useIssueContext();
  const { isCreating, issue, selectedTask: task } = context;
  const project = computed(() => issue.value.projectEntity);

  // Decide with type of UI should be displayed.
  const rollbackUIType = computed((): RollbackUIType => {
    if (task.value.type !== Task_Type.DATABASE_DATA_UPDATE) {
      return "NONE";
    }

    const database = databaseForTask(issue.value, task.value);
    const { engine, engineVersion } = database.instanceEntity;
    switch (engine) {
      case Engine.MYSQL:
        if (
          !semverCompare(engineVersion, MIN_ROLLBACK_SQL_MYSQL_VERSION, "gte")
        ) {
          return "NONE";
        }
        break;
      case Engine.ORACLE:
        // We don't have a check for oracle similar to the MySQL version check.
        break;
      default:
        return "NONE";
    }

    if (isCreating.value) {
      return "SWITCH";
    }

    switch (task.value.status) {
      case Task_Status.SKIPPED:
        return "NONE";
      case Task_Status.CANCELED:
        return "NONE";
      case Task_Status.DONE:
        return "FULL";
      default:
        return "SWITCH";
    }
  });

  // Decide whether current user can operate.
  const allowRollback = computed((): boolean => {
    if (rollbackUIType.value === "NONE") {
      return false;
    }

    if (isCreating.value) {
      return true;
    }

    const user = currentUserV1.value;

    if (user.email === extractUserResourceName(issue.value.creator)) {
      // Allowed to the issue creator
      return true;
    }

    if (user.email === extractUserResourceName(issue.value.assignee)) {
      // Allowed to the issue assignee
      return true;
    }

    if (
      hasPermissionInProjectV1(
        project.value.iamPolicy,
        user,
        "bb.permission.project.admin-database"
      )
    ) {
      return true;
    }

    if (
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-issue",
        user.userRole
      )
    ) {
      // Allowed to DBAs and workspace owners
      return true;
    }
    return false;
  });

  const rollbackEnabled = computed((): boolean => {
    if (isCreating.value) {
      // TODO: see if rollback enabled from issue plan
      return task.value.databaseDataUpdate?.rollbackEnabled ?? false;

      // if (isTenantMode.value) {
      //   // In tenant mode, all tasks share a common MigrationDetail
      //   const issueCreate = issue.value as IssueCreate;
      //   const createContext = issueCreate.createContext as MigrationContext;
      //   const migrationDetail = head(createContext.detailList);
      //   return migrationDetail?.rollbackEnabled ?? false;
      // }
      // // In standard mode, every task has a independent TaskCreate with its
      // // own rollbackEnabled field.
      // const taskCreate = task.value as TaskCreate;
      // return taskCreate.rollbackEnabled ?? false;
    } else {
      return task.value.databaseDataUpdate?.rollbackEnabled ?? false;
      // const taskEntity = task.value as Task;
      // const payload = taskEntity.payload as
      //   | TaskDatabaseDataUpdatePayload
      //   | undefined;
      // return payload?.rollbackEnabled ?? false;
    }
  });

  const toggleRollback = async (on: boolean) => {
    if (isCreating.value) {
      // TODO: update issue plan
      const config = task.value.databaseDataUpdate;
      if (config) {
        config.rollbackEnabled = on;
      }
      const spec = specForTask(issue.value, task.value);
      if (spec) {
        const config = spec.changeDatabaseConfig;
        if (config) {
          config.rollbackEnabled = on;
        }
      }

      // if (isTenantMode.value) {
      //   // In tenant mode, all tasks share a common MigrationDetail
      //   const issueCreate = issue.value as IssueCreate;
      //   const createContext = issueCreate.createContext as MigrationContext;
      //   createContext.detailList.forEach((detail) => {
      //     detail.rollbackEnabled = on;
      //   });
      // } else {
      //   // In standard mode, every task has a independent TaskCreate with its
      //   // own rollbackEnabled field.
      //   const taskCreate = task.value as TaskCreate;
      //   taskCreate.rollbackEnabled = on;
      // }
    } else {
      // TODO: patch plan to reconcile rollout/stages/tasks
      const spec = specForTask(issue.value, task.value);
      if (!spec) {
        notifyNotEditableLegacyIssue();
        return;
      }

      const config = task.value.databaseDataUpdate;
      if (config) {
        config.rollbackEnabled = on;
      }

      // // Once the issue has been created, we need to patch the task.
      // const taskEntity = task.value as Task;
      // await patchTask(taskEntity.id, {
      //   rollbackEnabled: on,
      // });

      // const issueEntity = issue.value as Issue;
      // const action = on ? "Enable" : "Disable";
      // try {
      //   await useIssueV1Store().createIssueComment({
      //     issueId: issueEntity.id,
      //     comment: `${action} SQL rollback log for task [${taskEntity.name}].`,
      //     payload: {
      //       issueName: issueEntity.name,
      //     },
      //   });
      // } catch {
      //   // do nothing
      //   // failing to comment to won't be too bad
      // }
    }
  };

  return {
    rollbackUIType,
    allowRollback,
    rollbackEnabled,
    toggleRollback,
  };
};

export const maybeCreateBackTraceComments = async (newIssue: ComposedIssue) => {
  const rollbackList = [] as Array<{
    byTask: Task;
    fromIssue: string;
    fromTask: string;
  }>;
  const taskList = flattenTaskV1List(newIssue.rolloutEntity);
  for (let i = 0; i < taskList.length; i++) {
    const byTask = taskList[i];
    if (byTask.type !== Task_Type.DATABASE_DATA_UPDATE) {
      continue;
    }
    const config = byTask.databaseDataUpdate;
    if (!config) {
      continue;
    }
    if (config.rollbackFromIssue && config.rollbackFromTask) {
      rollbackList.push({
        byTask,
        fromIssue: config.rollbackFromIssue,
        fromTask: config.rollbackFromTask,
      });
    }
  }
  if (rollbackList.length === 0) return;

  for (let i = 0; i < rollbackList.length; i++) {
    const {
      // byTask,
      fromIssue: fromIssueName,
      fromTask: fromTaskName,
    } = rollbackList[i];
    const fromIssue = await experimentalFetchIssueByName(fromIssueName);

    if (fromIssue.uid === String(UNKNOWN_ID)) continue;
    const fromTask = flattenTaskV1List(fromIssue.rolloutEntity).find(
      (task) => task.name === fromTaskName
    );
    if (!fromTask || fromTask.uid === String(UNKNOWN_ID)) continue;

    // const comment = [
    //   `Create issue #${newIssue.uid}`,
    //   "to rollback task",
    //   `[${fromTask.title}]`,
    // ].join(" ");
    try {
      // TODO: create comment
      // await useIssueV1Store().createIssueComment({
      //   issueId: fromIssue.uid,
      //   comment,
      //   payload: {
      //     issueName: fromIssue.name,
      //     taskRollbackBy: {
      //       issueId: fromIssue.id,
      //       taskId: fromTask.id,
      //       rollbackByIssueId: newIssue.id,
      //       rollbackByTaskId: byTask.id,
      //     },
      //   },
      // });
    } catch {
      // do nothing
      // failing to comment to won't be too bad
    }
  }
};
