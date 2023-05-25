import { computed, defineComponent } from "vue";
import { cloneDeep, head } from "lodash-es";
import {
  useDatabaseStore,
  useSheetStore,
  useTaskStore,
  useSheetById,
} from "@/store";
import {
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  TaskDatabaseSchemaUpdatePayload,
  MigrationContext,
  SheetId,
  UNKNOWN_ID,
} from "@/types";
import {
  errorAssertion,
  flattenTaskList,
  isTaskEditable,
  maybeFormatStatementOnSave,
  useCommonLogic,
} from "./common";
import { provideIssueLogic, useIssueLogic } from "./index";
import { getBacktracePayloadWithIssue, sheetIdOfTask } from "@/utils";

export default defineComponent({
  name: "TenantModeProvider",
  setup() {
    const { create, issue, selectedTask, createIssue, onStatusChanged } =
      useIssueLogic();
    const databaseStore = useDatabaseStore();
    const sheetStore = useSheetStore();
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
        const selectedTaskSheetId = sheetIdOfTask(selectedTask.value as Task);
        return (
          useSheetById(selectedTaskSheetId || payload.sheetId || UNKNOWN_ID)
            .value?.statement || ""
        );
      }
    });

    const updateStatement = async (newStatement: string) => {
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
        const issueEntity = issue.value as Issue;
        const sheet = await sheetStore.createSheet({
          projectId: issueEntity.project.id,
          name: `Sheet for issue #${issueEntity.id}`,
          statement: newStatement,
          visibility: "PROJECT",
          source: "BYTEBASE_ARTIFACT",
          payload: getBacktracePayloadWithIssue(issue.value as Issue),
        });
        updateSheetId(sheet.id);
      }
    };

    const updateSheetId = (sheetId: SheetId) => {
      if (create.value) {
        // For tenant deploy mode, we apply the sheetId to all stages and all tasks.
        const allTaskList = flattenTaskList<TaskCreate>(issue.value);
        allTaskList.forEach((task) => {
          task.statement = "";
          task.sheetId = sheetId;
        });

        const issueCreate = issue.value as IssueCreate;
        const context = issueCreate.createContext as MigrationContext;
        // We also apply it back to the CreateContext
        context.detailList.forEach((detail) => {
          detail.statement = "";
          detail.sheetId = sheetId;
        });
      } else {
        // Call patchAllTasksInIssue for tenant mode.
        const issueEntity = issue.value as Issue;
        taskStore
          .patchAllTasksInIssue({
            issueId: issueEntity.id,
            pipelineId: issueEntity.pipeline.id,
            taskPatch: {
              sheetId,
            },
          })
          .then(() => {
            onStatusChanged(true);
          });
      }
    };

    // We are never allowed to "apply task state to other stages" in tenant mode.
    const allowApplyTaskStateToOthers = computed(() => false);

    const doCreate = async () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      // for multi-tenancy issue pipeline (M * N)
      // createContext is up-to-date already
      // so we just format the statement if needed
      const context = issueCreate.createContext as MigrationContext;
      const detail = head(context.detailList);
      if (!detail) {
        // throw error
        return;
      }
      const db = databaseStore.getDatabaseById(detail.databaseId!);
      if (!detail.sheetId || detail.sheetId === UNKNOWN_ID) {
        const statement = maybeFormatStatementOnSave(detail.statement, db);
        const sheet = await useSheetStore().createSheet({
          projectId: issueCreate.projectId,
          name: issueCreate.name + " - " + db.name,
          statement: statement,
          visibility: "PROJECT",
          source: "BYTEBASE_ARTIFACT",
          payload: {},
        });
        detail.statement = "";
        detail.sheetId = sheet.id;
      }
      for (const detailItem of context.detailList) {
        detailItem.statement = "";
        detailItem.sheetId = detail.sheetId;
      }

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
