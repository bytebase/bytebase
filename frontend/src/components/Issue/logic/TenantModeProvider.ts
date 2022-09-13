import { computed, defineComponent } from "vue";
import { cloneDeep } from "lodash-es";
import { useDatabaseStore, useTaskStore } from "@/store";
import {
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  TaskDatabaseSchemaUpdatePayload,
  UpdateSchemaContext,
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
    const { create, issue, createIssue, onStatusChanged } = useIssueLogic();
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
        const context = issueCreate.createContext as UpdateSchemaContext;
        return context.updateSchemaDetailList[0].statement;
      } else {
        const issueEntity = issue.value as Issue;
        const task = issueEntity.pipeline.stageList[0].taskList[0];
        const payload = task.payload as TaskDatabaseSchemaUpdatePayload;
        return payload.statement;
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
        const context = issueCreate.createContext as UpdateSchemaContext;
        // We also apply it back to the CreateContext
        context.updateSchemaDetailList.forEach(
          (detail) => (detail.statement = newStatement)
        );
      } else {
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

    // We are never allowed to "apply statement to other stages" in tenant mode.
    const allowApplyStatementToOtherTasksOtherTasks = computed(() => false);

    const doCreate = () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      // for multi-tenancy issue pipeline (M * N)
      // createContext is up-to-date already
      // so we just format the statement if needed
      const context = issueCreate.createContext as UpdateSchemaContext;
      context.updateSchemaDetailList.forEach((detail) => {
        const db = databaseStore.getDatabaseById(detail.databaseId!);
        detail.statement = maybeFormatStatementOnSave(detail.statement, db);
      });

      createIssue(issueCreate);
    };

    const logic = {
      ...useCommonLogic(),
      allowEditStatement,
      selectedStatement,
      updateStatement,
      allowApplyStatementToOtherTasksOtherTasks,
      applyStatementToOtherTasksOtherTasks: errorAssertion,
      doCreate,
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
