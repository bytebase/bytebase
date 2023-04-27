import { computed, defineComponent } from "vue";
import { cloneDeep, isUndefined } from "lodash-es";
import { useDatabaseStore, useTaskStore } from "@/store";
import {
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  TaskDatabaseSchemaUpdatePayload,
  MigrationContext,
  SheetId,
} from "@/types";
import {
  errorAssertion,
  flattenTaskList,
  isTaskEditable,
  maybeFormatStatementOnSave,
  useCommonLogic,
} from "./common";
import { provideIssueLogic, useIssueLogic } from "./index";

export default defineComponent({
  name: "TenantModeProvider",
  setup() {
    const { create, issue, selectedTask, createIssue, onStatusChanged } =
      useIssueLogic();
    const databaseStore = useDatabaseStore();
    const taskStore = useTaskStore();

    const allowEditStatement = computed(() => {
      if (create.value) {
        return true;
      }
      const tasks = flattenTaskList<Task>(issue.value);
      return tasks.every((task) => isTaskEditable(task));
    });

    // In tenant mode, the entire issue shares only one SQL statement
    const selectedStatement = computed(() => {
      if (create.value) {
        const issueCreate = issue.value as IssueCreate;
        const context = issueCreate.createContext as MigrationContext;
        return context.detailList[0].statement;
      } else {
        const issueEntity = issue.value as Issue;
        const task = issueEntity.pipeline.stageList[0].taskList[0];
        const payload = task.payload as TaskDatabaseSchemaUpdatePayload;
        // Return the statement from the selected task, or the statement from the first task.
        return (
          (
            (selectedTask.value as Task)
              .payload as TaskDatabaseSchemaUpdatePayload
          ).statement || payload.statement
        );
      }
    });

    const updateStatement = (
      newStatement: string,
      postUpdated?: (updatedTask: Task) => void
    ) => {
      if (create.value) {
        // For tenant deploy mode, we apply the statement to all stages and all tasks
        const allTaskList = flattenTaskList<TaskCreate>(issue.value);
        allTaskList.forEach((task) => {
          task.statement = newStatement;
        });

        const issueCreate = issue.value as IssueCreate;
        const context = issueCreate.createContext as MigrationContext;
        // We also apply it back to the CreateContext
        context.detailList.forEach(
          (detail) => (detail.statement = newStatement)
        );
      } else {
        // Call patchAllTasksInIssue for tenant mode
        const issueEntity = issue.value as Issue;
        taskStore
          .patchAllTasksInIssue({
            issueId: issueEntity.id,
            pipelineId: issueEntity.pipeline.id,
            taskPatch: {
              statement: newStatement,
            },
          })
          .then(() => {
            onStatusChanged(true);
            if (postUpdated) {
              postUpdated(issueEntity.pipeline.stageList[0].taskList[0]);
            }
          });
      }
    };

    const updateSheetId = (sheetId: SheetId | undefined) => {
      if (!create.value) {
        return;
      }

      // For tenant deploy mode, we apply the sheetId to all stages and all tasks
      const allTaskList = flattenTaskList<TaskCreate>(issue.value);
      allTaskList.forEach((task) => {
        task.sheetId = sheetId;
      });

      const issueCreate = issue.value as IssueCreate;
      const context = issueCreate.createContext as MigrationContext;
      // We also apply it back to the CreateContext
      context.detailList.forEach((detail) => (detail.sheetId = sheetId));
    };

    // We are never allowed to "apply task state to other stages" in tenant mode.
    const allowApplyTaskStateToOthers = computed(() => false);

    const doCreate = () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      // for multi-tenancy issue pipeline (M * N)
      // createContext is up-to-date already
      // so we just format the statement if needed
      const context = issueCreate.createContext as MigrationContext;
      context.detailList.forEach((detail) => {
        const db = databaseStore.getDatabaseById(detail.databaseId!);
        if (!isUndefined(detail.sheetId)) {
          // If task already has sheet id, we do not need to save statement.
          detail.statement = "";
        } else {
          detail.statement = maybeFormatStatementOnSave(detail.statement, db);
        }
      });

      createIssue(issueCreate);
    };

    const logic = {
      ...useCommonLogic(),
      allowEditStatement,
      selectedStatement,
      updateStatement,
      updateSheetId,
      allowApplyTaskStateToOthers,
      applyTaskStateToOthers: errorAssertion,
      doCreate,
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
