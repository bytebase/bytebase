import { computed, defineComponent } from "vue";
import { cloneDeep } from "lodash-es";
import { useDatabaseStore } from "@/store";
import {
  Issue,
  IssueCreate,
  TaskCreate,
  TaskDatabaseSchemaUpdatePayload,
  UpdateSchemaContext,
} from "@/types";
import {
  errorAssertion,
  flattenTaskList,
  formatStatementIfNeeded,
  useCommonLogic,
} from "./common";
import { provideIssueLogic, useIssueLogic } from "./index";

export default defineComponent({
  name: "TenantModeProvider",
  setup() {
    const { create, issue, createIssue } = useIssueLogic();
    const databaseStore = useDatabaseStore();

    const allowEditStatement = computed(() => {
      if (create.value) {
        return true;
      }
      // Once a tenant-mode issue created, its statement can never be changed.
      return false;
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

    const updateStatement = (newStatement: string) => {
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
        // But not editable in non-create mode
        errorAssertion();
      }
    };

    // We are never allowed to "apply statement to other stages" in tenant mode.
    const allowApplyStatementToOtherStages = computed(() => false);

    const doCreate = () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      // for multi-tenancy issue pipeline (M * N)
      // createContext is up-to-date already
      // so we just format the statement if needed
      const context = issueCreate.createContext as UpdateSchemaContext;
      context.updateSchemaDetailList.forEach((detail) => {
        const db = databaseStore.getDatabaseById(detail.databaseId!);
        detail.statement = formatStatementIfNeeded(detail.statement, db);
      });

      createIssue(issueCreate);
    };

    const logic = {
      ...useCommonLogic(),
      allowEditStatement,
      selectedStatement,
      updateStatement,
      allowApplyStatementToOtherStages,
      applyStatementToOtherStages: errorAssertion,
      doCreate,
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
