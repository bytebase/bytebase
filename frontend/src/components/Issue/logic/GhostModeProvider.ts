import { computed, defineComponent } from "vue";
import { cloneDeep } from "lodash-es";
import { provideIssueLogic, useIssueLogic } from "./index";
import {
  flattenTaskList,
  maybeFormatStatementOnSave,
  TaskTypeWithStatement,
  useCommonLogic,
} from "./common";
import {
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  TaskStatus,
  MigrationContext,
  SheetId,
  UNKNOWN_ID,
} from "@/types";
import {
  useDatabaseStore,
  useSheetStore,
  useSheetById,
  useTaskStore,
} from "@/store";
import { sheetIdOfTask } from "@/utils";

export default defineComponent({
  name: "GhostModeProvider",
  setup() {
    const {
      create,
      issue,
      selectedTask,
      createIssue,
      isTenantMode,
      allowApplyTaskStatusTransition: baseAllowApplyTaskStatusTransition,
      allowApplyTaskStateToOthers: baseAllowApplyTaskStateToOthers,
      applyTaskStateToOthers: baseApplyTaskStateToOthers,
      onStatusChanged,
    } = useIssueLogic();
    const databaseStore = useDatabaseStore();
    const taskStore = useTaskStore();

    const selectedStatement = computed(() => {
      if (isTenantMode.value) {
        // In tenant mode, the entire issue shares only one SQL statement
        if (create.value) {
          const issueCreate = issue.value as IssueCreate;
          const context = issueCreate.createContext as MigrationContext;
          return (
            useSheetById(context.detailList[0].sheetId).value?.statement || ""
          );
        } else {
          const issueEntity = issue.value as Issue;
          const task = issueEntity.pipeline.stageList[0].taskList[0];
          const payload =
            task.payload as TaskDatabaseSchemaUpdateGhostSyncPayload;
          return useSheetById(payload.sheetId).value?.statement || "";
        }
      } else {
        // In standard pipeline, each ghost-sync task can hold its own
        // statement
        const task = selectedTask.value;
        if (task.type === "bb.task.database.schema.update.ghost.sync") {
          if (create.value) {
            let statement = (task as TaskCreate).statement;
            if ((task as TaskCreate).sheetId !== UNKNOWN_ID) {
              statement =
                useSheetById((task as TaskCreate).sheetId).value?.statement ||
                "";
            }
            return statement;
          } else {
            return (
              useSheetById(sheetIdOfTask(task as Task) || UNKNOWN_ID).value
                ?.statement || ""
            );
          }
        } else {
          return "";
        }
      }
    });

    const updateStatement = async (newStatement: string) => {
      if (isTenantMode.value) {
        if (create.value) {
          const task = selectedTask.value as TaskCreate;
          // For tenant deploy mode, we apply the statement to all stages and all tasks
          const allTaskList = flattenTaskList<TaskCreate>(issue.value);
          allTaskList.forEach((taskItem) => {
            if (TaskTypeWithStatement.includes(taskItem.type)) {
              if (task.sheetId) {
                taskItem.statement = "";
                taskItem.sheetId = task.sheetId;
              } else {
                taskItem.statement = newStatement;
              }
            }
          });

          const issueCreate = issue.value as IssueCreate;
          const context = issueCreate.createContext as MigrationContext;
          // We also apply it back to the CreateContext
          context.detailList.forEach((detail) => {
            if (task.sheetId) {
              detail.statement = "";
              detail.sheetId = task.sheetId;
            } else {
              detail.statement = newStatement;
            }
          });
        } else {
          const sheetId = sheetIdOfTask(selectedTask.value as Task);
          if (sheetId && sheetId !== UNKNOWN_ID) {
            // Call patchAllTasksInIssue for tenant mode
            const issueEntity = issue.value as Issue;
            await taskStore.patchAllTasksInIssue({
              issueId: issueEntity.id,
              pipelineId: issueEntity.pipeline.id,
              taskPatch: {
                sheetId,
              },
            });
            onStatusChanged(true);
          }
        }
      } else {
        if (create.value) {
          const task = selectedTask.value as TaskCreate;
          task.statement = newStatement;
        }
      }
    };

    const updateSheetId = (sheetId: SheetId) => {
      if (isTenantMode.value) {
        // For tenant deploy mode, we apply the sheetId to all stages and all tasks
        const allTaskList = flattenTaskList<TaskCreate>(issue.value);
        allTaskList.forEach((task) => {
          task.statement = "";
          task.sheetId = sheetId;
        });

        const issueCreate = issue.value as IssueCreate;
        const context = issueCreate.createContext as MigrationContext;
        context.detailList.forEach((detail) => {
          detail.statement = "";
          detail.sheetId = sheetId;
        });
      } else {
        const task = selectedTask.value as TaskCreate;
        task.statement = "";
        task.sheetId = sheetId;
      }
    };

    const doCreate = async () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      if (isTenantMode.value) {
        // for tenant pipeline
        // createContext is up-to-date already
        // so we just format the statement if needed
        const context = issueCreate.createContext as MigrationContext;
        for (const detail of context.detailList) {
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
        }
      } else {
        // for standard pipeline, we copy user edited tasks back to
        // issue.createContext
        // For each ghost-sync task, we copy its edited statement back to the
        // createContext.detailList by its databaseId accordingly.
        const createContext = issueCreate.createContext as MigrationContext;
        const syncTaskList = flattenTaskList<TaskCreate>(issueCreate).filter(
          (task) => task.type === "bb.task.database.schema.update.ghost.sync"
        );
        const detailList = createContext.detailList;
        for (const task of syncTaskList) {
          const { databaseId } = task;
          if (!databaseId) return;
          const detail = detailList.find(
            (detail) => detail.databaseId === databaseId
          );
          if (detail) {
            const db = databaseStore.getDatabaseById(databaseId);
            if (!detail.sheetId || detail.sheetId === UNKNOWN_ID) {
              const statement = maybeFormatStatementOnSave(task.statement, db);
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
            detail.earliestAllowedTs = task.earliestAllowedTs;
          }
        }
      }

      createIssue(issueCreate);
    };

    const allowApplyTaskStatusTransition = (
      task: Task,
      to: TaskStatus
    ): boolean => {
      if (
        task.type === "bb.task.database.schema.update.ghost.cutover" &&
        task.status === "FAILED"
      ) {
        if (to === "PENDING" || to === "RUNNING") {
          // RETRYing gh-ost cut-over task is not allowed (yet).
          return false;
        }
      }
      if (
        task.type === "bb.task.database.schema.update.ghost.sync" &&
        to === "CANCELED"
      ) {
        // CANCELing gh-ost sync task is allowed.
        return true;
      }
      return baseAllowApplyTaskStatusTransition(task, to);
    };

    const allowApplyTaskStateToOthers = computed(() => {
      // We are never allowed to "apply task state to other stages" in tenant mode.
      if (isTenantMode.value) return false;
      return baseAllowApplyTaskStateToOthers.value;
    });

    const applyTaskStateToOthers = (task: TaskCreate) => {
      if (!allowApplyTaskStateToOthers.value) return;
      return baseApplyTaskStateToOthers(task);
    };

    const logic = {
      ...useCommonLogic(),
      selectedStatement,
      doCreate,
      allowApplyTaskStatusTransition,
      allowApplyTaskStateToOthers,
      applyTaskStateToOthers,
      updateStatement,
      updateSheetId,
    };
    provideIssueLogic(logic);
    return logic;
  },
  render() {
    return this.$slots.default?.();
  },
});
