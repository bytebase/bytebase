import { computed } from "vue";
import { cloneDeep, isNaN, isNumber } from "lodash-es";
import { useRoute } from "vue-router";
import { v4 as uuidv4 } from "uuid";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import {
  useCurrentUser,
  useDatabaseStore,
  useIssueStore,
  useSheetStore,
  useTaskStore,
  useUIStateStore,
} from "@/store";
import {
  Database,
  Issue,
  IssueCreate,
  IssuePatch,
  IssueType,
  Task,
  TaskCreate,
  TaskId,
  TaskPatch,
  TaskType,
  MigrationDetail,
  MigrationType,
  SheetId,
  MigrationContext,
  dialectOfEngine,
  languageOfEngine,
  UNKNOWN_ID,
} from "@/types";
import { IssueLogic, useIssueLogic } from "./index";
import {
  defer,
  isDev,
  isTaskTriggeredByVCS,
  taskCheckRunSummary,
} from "@/utils";
import { maybeApplyRollbackParams } from "@/plugins/issue/logic/initialize/standard";
import { t } from "@/plugins/i18n";

export const useCommonLogic = () => {
  const { create, issue, selectedTask, createIssue, onStatusChanged, dialog } =
    useIssueLogic();
  const route = useRoute();
  const currentUser = useCurrentUser();
  const databaseStore = useDatabaseStore();
  const issueStore = useIssueStore();
  const taskStore = useTaskStore();
  const sheetStore = useSheetStore();

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
    return taskStore
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

  const initialTaskListStatementFromRoute = () => {
    if (!create.value) {
      return;
    }

    const taskList = flattenTaskList<TaskCreate>(issue.value).filter((task) =>
      TaskTypeWithStatement.includes(task.type)
    );
    // route.query.databaseList is comma-splitted databaseId list
    // e.g. databaseList=7002,7006,7014
    const idListString = (route.query.databaseList as string) || "";
    const databaseIdList = idListString.split(",");
    if (databaseIdList.length === 0) {
      return;
    }

    // route.query.sheetId is an id of sheet. Mainly using in creating rollback issue.
    const sheetId = Number(route.query.sheetId);
    // route.query.sqlList is JSON string of a string array.
    const sqlListString = (route.query.sqlList as string) || "";
    if (isNumber(sheetId) && !isNaN(sheetId)) {
      for (const databaseId of databaseIdList) {
        const task = taskList.find(
          (task) => task.databaseId === Number(databaseId)
        );
        if (task) {
          task.sheetId = sheetId;
        }
      }
    } else if (sqlListString) {
      const statementList = JSON.parse(sqlListString) as string[];
      for (
        let i = 0;
        i < Math.min(databaseIdList.length, statementList.length);
        i++
      ) {
        const task = taskList.find(
          (task) => task.databaseId === Number(databaseIdList[i])
        );
        if (task) {
          task.statement = statementList[i];
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

  const updateStatement = async (newStatement: string) => {
    if (create.value) {
      const task = selectedTask.value as TaskCreate;
      task.statement = newStatement;
    } else {
      // Ask whether to apply the change to all pending tasks if possible.
      const task = selectedTask.value as Task;
      const issueEntity = issue.value as Issue;
      const patchingTaskList = await getPatchingTaskList(
        issueEntity,
        task,
        dialog
      );
      if (patchingTaskList.length === 0) return;

      // Create a new sheet instead of reusing the old one.
      const sheet = await sheetStore.createSheet({
        projectId: issueEntity.project.id,
        name: uuidv4(),
        statement: newStatement,
        visibility: "PROJECT",
        source: "BYTEBASE_ARTIFACT",
        payload: {},
      });

      const patchRequestList = patchingTaskList.map((task) => {
        patchTask(task.id, { sheetId: sheet.id });
      });
      return Promise.allSettled(patchRequestList);
    }
  };

  const updateSheetId = async (sheetId: SheetId) => {
    if (create.value) {
      const task = selectedTask.value as TaskCreate;
      task.statement = "";
      task.sheetId = sheetId;
    } else {
      const task = selectedTask.value as Task;
      await patchTask(task.id, { sheetId });
    }
  };

  const doCreate = async () => {
    const issueCreate = cloneDeep(issue.value as IssueCreate);
    // for standard issue pipeline (1 * 1 or M * 1)
    // copy user edited tasks back to issue.createContext
    const taskCreateList = flattenTaskList<TaskCreate>(issueCreate);
    const detailList: MigrationDetail[] = [];
    for (const taskCreate of taskCreateList) {
      const db = databaseStore.getDatabaseById(taskCreate.databaseId!);
      const statement = maybeFormatStatementOnSave(taskCreate.statement, db);
      const migrationDetail: MigrationDetail = {
        migrationType: getMigrationTypeFromTask(taskCreate),
        databaseId: taskCreate.databaseId,
        statement: statement,
        sheetId: taskCreate.sheetId,
        earliestAllowedTs: taskCreate.earliestAllowedTs,
        rollbackEnabled: taskCreate.rollbackEnabled,
      };
      // Create a new sheet to save statement.
      if (!taskCreate.sheetId || taskCreate.sheetId === UNKNOWN_ID) {
        const sheet = await useSheetStore().createSheet({
          projectId: issueCreate.projectId,
          name: issueCreate.name + " - " + db.name,
          statement: statement,
          visibility: "PROJECT",
          source: "BYTEBASE_ARTIFACT",
          payload: {},
        });
        migrationDetail.sheetId = sheet.id;
      } else {
        const sheetId = taskCreate.sheetId;
        const sheet = sheetStore.getSheetById(sheetId);
        if (sheet.statement.length === sheet.size) {
          await sheetStore.patchSheetById({
            id: sheetId,
            statement: statement,
          });
        }
      }
      migrationDetail.statement = "";
      detailList.push(migrationDetail);
    }

    const createContext: MigrationContext = {
      detailList,
    };
    maybeApplyRollbackParams(createContext, route);
    issueCreate.createContext = createContext;
    createIssue(issueCreate);
  };

  return {
    patchIssue,
    patchTask,
    allowEditStatement,
    initialTaskListStatementFromRoute,
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

  const language = languageOfEngine(database?.instance.engine);
  if (language !== "sql") {
    return statement;
  }
  const dialect = dialectOfEngine(database?.instance.engine);

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

export const getPatchingTaskList = async (
  issue: Issue,
  task: Task,
  dialog: IssueLogic["dialog"]
) => {
  const patchableTaskList = flattenTaskList<Task>(issue).filter(
    (task) => TaskTypeWithStatement.includes(task.type) && isTaskEditable(task)
  );
  const d = defer<Task[]>();
  if (patchableTaskList.length > 1) {
    dialog.info({
      title: t("task.apply-change-to-all-pending-tasks.title"),
      style: "width: auto",
      negativeText: t("task.apply-change-to-all-pending-tasks.current-only"),
      positiveText: t("task.apply-change-to-all-pending-tasks.self"),
      onPositiveClick: () => {
        d.resolve(patchableTaskList);
      },
      onNegativeClick: () => {
        d.resolve([task]);
      },
      autoFocus: false,
      maskClosable: false,
      closeOnEsc: false,
      onClose: () => {
        d.resolve([]);
      },
    });
  } else {
    d.resolve([task]);
  }

  return d.promise;
};
