import { computed } from "vue";
import { cloneDeep } from "lodash-es";
import formatSQL from "@/components/MonacoEditor/sqlFormatter";
import { useCurrentUser, useDatabaseStore, useUIStateStore } from "@/store";
import {
  Database,
  Issue,
  IssueCreate,
  IssueType,
  SQLDialect,
  Task,
  TaskCreate,
  TaskDatabaseCreatePayload,
  TaskDatabaseDataUpdatePayload,
  TaskDatabaseSchemaUpdatePayload,
  TaskGeneralPayload,
  TaskType,
  UpdateSchemaDetail,
} from "@/types";
import { useIssueLogic } from "./index";
import { isDev } from "@/utils";

export const useCommonLogic = () => {
  const { create, issue, selectedTask, patchTask, createIssue } =
    useIssueLogic();
  const currentUser = useCurrentUser();
  const databaseStore = useDatabaseStore();

  const allowEditStatement = computed(() => {
    // if creating an issue, it's editable
    if (create.value) {
      return true;
    }

    // if not creating, we are allowed to edit sql statement only when:
    // 1. issue.status is OPEN
    // 2. AND currentUser is the creator
    // 3. AND workflowType is UI
    const issueEntity = issue.value as Issue;
    if (issueEntity.status !== "OPEN") {
      return false;
    }
    if (issueEntity.creator.id !== currentUser.value.id) {
      return false;
    }
    if (issueEntity.project.workflowType !== "UI") {
      return false;
    }

    // check `selectedTask`, expected to be PENDING or PENDING_APPROVAL or FAILED
    const checkTask = (task: Task) => {
      return (
        task.status == "PENDING" ||
        task.status == "PENDING_APPROVAL" ||
        task.status == "FAILED"
      );
    };
    return checkTask(selectedTask.value as Task);
  });

  const selectedStatement = computed((): string => {
    const task = selectedTask.value;
    if (create.value) {
      return (task as TaskCreate).statement;
    }

    // Extract statement from different types of payloads
    return statementOfTask(task as Task);
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
        },
        postUpdated
      );
    }
  };

  const applyStatementToOtherStages = (statement: string) => {
    const taskList = flattenTaskList<TaskCreate>(issue.value);

    for (const task of taskList) {
      if (TaskTypeWithStatement.includes(task.type)) {
        task.statement = statement;
      }
    }
  };

  const allowApplyStatementToOtherStages = computed(() => {
    if (!create.value) {
      return false;
    }

    const taskList = flattenTaskList<TaskCreate>(issue.value);
    const count = taskList.filter((task) =>
      TaskTypeWithStatement.includes(task.type)
    ).length;

    return count > 1;
  });

  const doCreate = () => {
    const issueCreate = cloneDeep(issue.value as IssueCreate);
    // for standard issue pipeline (1 * 1 or M * 1)
    // copy user edited tasks back to issue.createContext
    const taskList = issueCreate.pipeline!.stageList.map(
      (stage) => stage.taskList[0]
    );
    const detailList: UpdateSchemaDetail[] = taskList.map((task) => {
      const db = databaseStore.getDatabaseById(task.databaseId!);
      return {
        databaseId: task.databaseId!,
        databaseName: task.databaseName!,
        statement: maybeFormatStatementOnSave(task.statement, db),
        earliestAllowedTs: task.earliestAllowedTs,
      };
    });
    issueCreate.createContext = {
      migrationType: taskList[0].migrationType!,
      updateSchemaDetailList: detailList,
    };

    createIssue(issueCreate);
  };

  return {
    patchTask,
    allowEditStatement,
    selectedStatement,
    updateStatement,
    allowApplyStatementToOtherStages,
    applyStatementToOtherStages,
    doCreate,
  };
};

export const TaskTypeWithStatement: TaskType[] = [
  "bb.task.general",
  "bb.task.database.create",
  "bb.task.database.schema.update",
  "bb.task.database.schema.update.ghost.sync",
  "bb.task.database.data.update",
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

const statementOfTask = (task: Task) => {
  switch (task.type) {
    case "bb.task.general":
      return ((task as Task).payload as TaskGeneralPayload).statement || "";
    case "bb.task.database.create":
      return (
        ((task as Task).payload as TaskDatabaseCreatePayload).statement || ""
      );
    case "bb.task.database.schema.update":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdatePayload).statement ||
        ""
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
    case "bb.task.database.schema.update.ghost.drop-original-table":
      return ""; // should never reach here
  }
};
