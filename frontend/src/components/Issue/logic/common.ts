import { computed } from "vue";
import { cloneDeep, isUndefined } from "lodash-es";
import { useRoute } from "vue-router";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import {
  useCurrentUser,
  useDatabaseStore,
  useIssueStore,
  useTaskStore,
  useUIStateStore,
} from "@/store";
import {
  Database,
  Issue,
  IssueCreate,
  IssuePatch,
  IssueType,
  SQLDialect,
  Task,
  TaskCreate,
  TaskDatabaseCreatePayload,
  TaskDatabaseDataUpdatePayload,
  TaskDatabaseSchemaUpdatePayload,
  TaskDatabaseSchemaUpdateSDLPayload,
  TaskGeneralPayload,
  TaskId,
  TaskPatch,
  TaskType,
  MigrationDetail,
  MigrationType,
  TaskDatabaseSchemaBaselinePayload,
  DatabaseId,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  SheetId,
} from "@/types";
import { useIssueLogic } from "./index";
import { isDev, isTaskTriggeredByVCS, taskCheckRunSummary } from "@/utils";

export const useCommonLogic = () => {
  const { create, issue, selectedTask, createIssue, onStatusChanged } =
    useIssueLogic();
  const route = useRoute();
  const currentUser = useCurrentUser();
  const databaseStore = useDatabaseStore();
  const issueStore = useIssueStore();
  const taskStore = useTaskStore();

  const patchIssue = (
    issuePatch: IssuePatch,
    postUpdated?: (updatedIssue: Issue) => void
  ) => {
    issueStore
      .patchIssue({
        issueId: (issue.value as Issue).id,
        issuePatch,
      })
      .then((updatedIssue) => {
        // issue/patchIssue already fetches the new issue, so we schedule
        // the next poll in NORMAL_POLL_INTERVAL
        onStatusChanged(false);
        if (postUpdated) {
          postUpdated(updatedIssue);
        }
      });
  };

  const patchTask = (
    taskId: TaskId,
    taskPatch: TaskPatch,
    postUpdated?: (updatedTask: Task) => void
  ) => {
    taskStore
      .patchTask({
        issueId: (issue.value as Issue).id,
        pipelineId: (issue.value as Issue).pipeline.id,
        taskId,
        taskPatch,
      })
      .then((updatedTask) => {
        // For now, the only task/patchTask is to change statement, which will trigger async task check.
        // Thus we use the short poll interval
        onStatusChanged(true);
        if (postUpdated) {
          postUpdated(updatedTask);
        }
      });
  };

  const initialTaskListStatement = () => {
    if (create.value) {
      const taskList = flattenTaskList<TaskCreate>(issue.value).filter((task) =>
        TaskTypeWithStatement.includes(task.type)
      );
      const databaseStatementMap = new Map<DatabaseId, string>();
      // route.query.databaseList is comma-splitted databaseId list
      // e.g. databaseList=7002,7006,7014
      const idListString = route.query.databaseList as string;
      // route.query.sqlList is JSON string of a string array.
      const sqlListString = route.query.sqlList as string;
      if (idListString && sqlListString) {
        const databaseIdList = idListString.split(",");
        const statementList = JSON.parse(sqlListString) as string[];
        for (
          let i = 0;
          i < Math.min(databaseIdList.length, statementList.length);
          i++
        ) {
          databaseStatementMap.set(Number(databaseIdList[i]), statementList[i]);
        }
      }

      for (const [databaseId, statement] of databaseStatementMap) {
        const task = taskList.find((task) => task.databaseId === databaseId);
        if (task) {
          task.statement = statement;
        }
      }
    }
  };

  const allowEditStatement = computed(() => {
    // if creating an issue, it's editable
    if (create.value) {
      return true;
    }

    const issueEntity = issue.value as Issue;

    if (issueEntity.type === "bb.issue.database.restore.pitr") {
      return false;
    }

    if (issueEntity.type === "bb.issue.database.create") {
      // For standard mode projects, we are not allowed to edit the database
      // creation SQL statement.
      if (issueEntity.project.tenantMode !== "TENANT") {
        return false;
      }

      // We allow to edit create database statement for tenant project to give users a
      // chance to edit the dumped schema from its peer databases, because the dumped schema
      // may not be perfectly correct.
      // So we fallthrough to the common checkpoints.
    }

    // if not creating, we are allowed to edit sql statement only when:
    // 1. issue.status is OPEN
    // 2. AND currentUser is the creator
    if (issueEntity.status !== "OPEN") {
      return false;
    }
    if (issueEntity.creator.id !== currentUser.value.id) {
      if (isTaskTriggeredByVCS(selectedTask.value as Task)) {
        // If an issue is triggered by VCS, its creator will be 1 (SYSTEM_BOT_ID)
        // We should "Allow" current user to edit the statement (via VCS).
        return true;
      }
      return false;
    }

    return isTaskEditable(selectedTask.value as Task);
  });

  const updateStatement = (
    newStatement: string,
    postUpdated?: (updatedTask: Task) => void
  ) => {
    if (create.value) {
      const task = selectedTask.value as TaskCreate;
      task.statement = newStatement;
    } else {
      // otherwise, patch the task
      const task = selectedTask.value as Task;
      patchTask(
        task.id,
        {
          statement: maybeFormatStatementOnSave(newStatement, task.database),
          updatedTs: task.updatedTs,
        },
        postUpdated
      );
    }
  };

  const updateSheetId = (sheetId: SheetId | undefined) => {
    if (!create.value) {
      return;
    }

    const task = selectedTask.value as TaskCreate;
    task.sheetId = sheetId;
  };

  const doCreate = () => {
    const issueCreate = cloneDeep(issue.value as IssueCreate);
    // for standard issue pipeline (1 * 1 or M * 1)
    // copy user edited tasks back to issue.createContext
    const taskCreateList = flattenTaskList<TaskCreate>(issueCreate);
    const detailList: MigrationDetail[] = taskCreateList.map((taskCreate) => {
      const db = databaseStore.getDatabaseById(taskCreate.databaseId!);
      const migrationDetail: MigrationDetail = {
        migrationType: getMigrationTypeFromTask(taskCreate),
        databaseId: taskCreate.databaseId,
        statement: maybeFormatStatementOnSave(taskCreate.statement, db),
        sheetId: taskCreate.sheetId,
        earliestAllowedTs: taskCreate.earliestAllowedTs,
      };
      // If task already has sheet id, we do not need to save statement.
      if (!isUndefined(taskCreate.sheetId)) {
        migrationDetail.statement = "";
      }
      return migrationDetail;
    });

    issueCreate.createContext = {
      detailList: detailList,
    };

    createIssue(issueCreate);
  };

  return {
    patchIssue,
    patchTask,
    allowEditStatement,
    initialTaskListStatement,
    updateStatement,
    updateSheetId,
    doCreate,
  };
};

const getMigrationTypeFromTask = (task: Task | TaskCreate) => {
  let migrationType: MigrationType;
  if (task.type === "bb.task.database.schema.baseline") {
    migrationType = "BASELINE";
  } else if (task.type === "bb.task.database.data.update") {
    migrationType = "DATA";
  } else {
    migrationType = "MIGRATE";
  }
  return migrationType;
};

export const TaskTypeWithStatement: TaskType[] = [
  "bb.task.general",
  "bb.task.database.create",
  "bb.task.database.data.update",
  "bb.task.database.schema.baseline",
  "bb.task.database.schema.update",
  "bb.task.database.schema.update-sdl",
  "bb.task.database.schema.update.ghost.sync",
];

// TaskTypeWithSheetId should be a subset of TaskTypeWithStatement.
export const TaskTypeWithSheetId: TaskType[] = [
  "bb.task.database.data.update",
  "bb.task.database.schema.update",
  "bb.task.database.schema.update-sdl",
  "bb.task.database.schema.update.ghost.sync",
];

export const IssueTypeWithStatement: IssueType[] = [
  "bb.issue.database.create",
  "bb.issue.database.schema.update",
  "bb.issue.database.data.update",
  "bb.issue.database.schema.update.ghost",
];

export const flattenTaskList = <T extends Task | TaskCreate>(
  issue: Issue | IssueCreate
) => {
  const taskList = issue.pipeline?.stageList.flatMap(
    (stage) => stage.taskList as T[]
  );
  return taskList || [];
};

export const maybeFormatStatementOnSave = (
  statement: string,
  database?: Database
): string => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.issueFormatStatementOnSave) {
    // Don't format if user disabled this feature
    return statement;
  }

  // Default to use mysql dialect but use postgresql dialect if needed
  let dialect: SQLDialect = "mysql";
  if (database && database.instance.engine === "POSTGRES") {
    dialect = "postgresql";
  }

  const result = formatSQL(statement, dialect);
  if (!result.error) {
    return result.data;
  }

  // Fallback to the input statement if error occurs while formatting
  return statement;
};

// To let us know if we reaches a logic branch unexpectedly.
export const errorAssertion = () => {
  if (isDev()) {
    throw new Error("should never reach here");
  }
};

export const statementOfTask = (task: Task) => {
  switch (task.type) {
    case "bb.task.general":
      return ((task as Task).payload as TaskGeneralPayload).statement || "";
    case "bb.task.database.create":
      return (
        ((task as Task).payload as TaskDatabaseCreatePayload).statement || ""
      );
    case "bb.task.database.schema.baseline":
      return (
        ((task as Task).payload as TaskDatabaseSchemaBaselinePayload)
          .statement || ""
      );
    case "bb.task.database.schema.update":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdatePayload).statement ||
        ""
      );
    case "bb.task.database.schema.update-sdl":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateSDLPayload)
          .statement || ""
      );
    case "bb.task.database.data.update":
      return (
        ((task as Task).payload as TaskDatabaseDataUpdatePayload).statement ||
        ""
      );
    case "bb.task.database.restore":
      return "";
    case "bb.task.database.schema.update.ghost.sync":
    case "bb.task.database.schema.update.ghost.cutover":
      return ""; // should never reach here
  }
};

export const sheetIdOfTask = (task: Task) => {
  switch (task.type) {
    case "bb.task.database.create":
      return (
        ((task as Task).payload as TaskDatabaseCreatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update-sdl":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateSDLPayload)
          .sheetId || undefined
      );
    case "bb.task.database.data.update":
      return (
        ((task as Task).payload as TaskDatabaseDataUpdatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update.ghost.sync":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateGhostSyncPayload)
          .sheetId || undefined
      );
    default:
      return undefined;
  }
};

export const isTaskEditable = (task: Task): boolean => {
  if (task.status === "PENDING_APPROVAL" || task.status === "FAILED") {
    return true;
  }
  if (task.status === "PENDING") {
    // If a task's status is "PENDING", its statement is editable if there
    // are at least ONE ERROR task checks.
    // Since once all its task checks are fulfilled, it might be queued by
    // the scheduler.
    // Editing a queued task's SQL statement is dangerous with kinds of race
    // condition risks.
    const summary = taskCheckRunSummary(task);
    if (summary.errorCount > 0) {
      return true;
    }
  }

  return false;
};
