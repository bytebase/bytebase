import { computed, defineComponent } from "vue";
import { cloneDeep } from "lodash-es";
import { provideIssueLogic, useIssueLogic } from "./index";
import { maybeFormatStatementOnSave, useCommonLogic } from "./common";
import {
  IssueCreate,
  Task,
  TaskCreate,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  UpdateSchemaGhostContext,
} from "@/types";
import { useDatabaseStore } from "@/store";

export default defineComponent({
  name: "GhostModeProvider",
  setup() {
    const { create, issue, selectedTask, createIssue } = useIssueLogic();
    const databaseStore = useDatabaseStore();

    // In gh-ost mode, each stage can own its SQL statement
    // But only for task.type === "bb.task.database.schema.update.ghost.sync"
    const selectedStatement = computed(() => {
      const task = selectedTask.value;
      if (task.type === "bb.task.database.schema.update.ghost.sync") {
        if (create.value) {
          return (task as TaskCreate).statement;
        } else {
          const payload = (task as Task)
            .payload as TaskDatabaseSchemaUpdateGhostSyncPayload;
          return payload.statement;
        }
      } else {
        return "";
      }
    });

    const doCreate = () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      // for gh-ost mode, copy user edited tasks back to issue.createContext
      // only the first subtask (bb.task.database.schema.update.ghost.sync) has statement
      const stageList = issueCreate.pipeline!.stageList;
      const createContext =
        issueCreate.createContext as UpdateSchemaGhostContext;
      const detailList = createContext.updateSchemaGhostDetailList;
      stageList.forEach((stage, i) => {
        const detail = detailList[i];
        const syncTask = stage.taskList.find(
          (task) => task.type === "bb.task.database.schema.update.ghost.sync"
        )!;
        const db = databaseStore.getDatabaseById(syncTask.databaseId!);

        detail.databaseId = syncTask.databaseId!;
        detail.databaseName = syncTask.databaseName!;
        detail.statement = maybeFormatStatementOnSave(syncTask.statement, db);
        detail.earliestAllowedTs = syncTask.earliestAllowedTs;
      });

      createIssue(issueCreate);
    };

    const logic = {
      ...useCommonLogic(),
      selectedStatement,
      doCreate,
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
