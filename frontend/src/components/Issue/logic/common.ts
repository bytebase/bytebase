import { cloneDeep, isNaN, isNumber } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { computed } from "vue";
import { useRoute } from "vue-router";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { t } from "@/plugins/i18n";
import { maybeApplyRollbackParams } from "@/plugins/issue/logic/initialize/standard";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useIssueStore,
  useSheetV1Store,
  useTaskStore,
  useUIStateStore,
} from "@/store";
import {
  ComposedDatabase,
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
  UNKNOWN_ID,
  languageOfEngineV1,
  dialectOfEngineV1,
} from "@/types";
import { IssuePayload } from "@/types/proto/store/issue";
import {
  Sheet_Visibility,
  Sheet_Source,
  Sheet_Type,
} from "@/types/proto/v1/sheet_service";
import {
  defer,
  extractSheetUID,
  extractUserUID,
  getBacktracePayloadWithIssue,
  hasWorkspacePermissionV1,
  isDev,
  isTaskTriggeredByVCS,
  taskCheckRunSummary,
} from "@/utils";
import { IssueLogic, useIssueLogic } from "./index";

export const useCommonLogic = () => {
  const {
    create,
    issue,
    project,
    selectedTask,
    createIssue,
    onStatusChanged,
    dialog,
  } = useIssueLogic();
  const route = useRoute();
  const currentUserV1 = useCurrentUserV1();
  const databaseStore = useDatabaseV1Store();
  const issueStore = useIssueStore();
  const taskStore = useTaskStore();
  const sheetV1Store = useSheetV1Store();

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
        pipelineId: (issue.value as Issue).pipeline!.id,
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

  const initialTaskListStatementFromRoute = async () => {
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
        const database = await databaseStore.getOrFetchDatabaseByUID(
          databaseId
        );
        if (task) {
          task.sheetId = sheetId;
          const sheetName = `${database.project}/sheets/${sheetId}`;
          const sheet = await sheetV1Store.getOrFetchSheetByName(sheetName);
          task.statement = new TextDecoder().decode(sheet?.content);
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

    if (!isTaskEditable(selectedTask.value as Task)) {
      return false;
    }

    if (
      String(issueEntity.creator.id) ===
      extractUserUID(currentUserV1.value.name)
    ) {
      return true;
    }

    if (
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-issue",
        currentUserV1.value.userRole
      )
    ) {
      // Workspace OWNER/DBA are always allowed to edit.
      return true;
    }

    if (isTaskTriggeredByVCS(selectedTask.value as Task)) {
      // If an issue is triggered by VCS, its creator will be 1 (SYSTEM_BOT_ID)
      // We should "Allow" current user to edit the statement (via VCS).
      return true;
    }

    return false;
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
      const sheet = await sheetV1Store.createSheet(project.value.name, {
        title: uuidv4(),
        content: new TextEncoder().encode(newStatement),
        visibility: Sheet_Visibility.VISIBILITY_PROJECT,
        source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
        type: Sheet_Type.TYPE_SQL,
        payload: JSON.stringify(
          getBacktracePayloadWithIssue(issue.value as Issue)
        ),
      });

      const patchRequestList = patchingTaskList.map((task) => {
        patchTask(task.id, { sheetId: Number(extractUserUID(sheet.name)) });
      });
      await Promise.allSettled(patchRequestList);
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
      const db = databaseStore.getDatabaseByUID(String(taskCreate.databaseId!));
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
        const sheet = await sheetV1Store.createSheet(db.project, {
          title: issueCreate.name + " - " + db.databaseName,
          content: new TextEncoder().encode(statement),
          visibility: Sheet_Visibility.VISIBILITY_PROJECT,
          source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
          type: Sheet_Type.TYPE_SQL,
          payload: "{}",
        });
        migrationDetail.sheetId = Number(extractSheetUID(sheet.name));
      } else if (taskCreate.sheetId !== UNKNOWN_ID) {
        const sheetName = `${db.project}/sheets/${taskCreate.sheetId}`;
        const sheet = await sheetV1Store.getOrFetchSheetByName(sheetName);
        if (
          new TextDecoder().decode(sheet?.content).length === sheet?.contentSize
        ) {
          await sheetV1Store.patchSheet({
            name: sheetName,
            content: new TextEncoder().encode(statement),
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
  database?: ComposedDatabase
): string => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.issueFormatStatementOnSave) {
    // Don't format if user disabled this feature
    return statement;
  }

  const language = languageOfEngineV1(database?.instanceEntity.engine);
  if (language !== "sql") {
    return statement;
  }
  const dialect = dialectOfEngineV1(database?.instanceEntity.engine);

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

export const isGroupingChangeIssue = (issue: Issue): boolean => {
  const route = useRoute();
  if (!route || !route.query) {
    return false;
  }
  if (route.query.databaseGroupName && route.query.databaseGroupName !== "") {
    return true;
  }
  const groupName = (issue.payload as IssuePayload).grouping?.databaseGroupName;
  if (groupName && groupName !== "") {
    return true;
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
