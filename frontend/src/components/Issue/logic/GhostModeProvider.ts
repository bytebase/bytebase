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
  useSheetV1Store,
  useSheetStatementByUID,
  useTaskStore,
  useDatabaseV1Store,
} from "@/store";
import { extractSheetUID, sheetIdOfTask } from "@/utils";
import {
  Sheet_Visibility,
  Sheet_Source,
  Sheet_Type,
} from "@/types/proto/v1/sheet_service";

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
    const databaseStore = useDatabaseV1Store();
    const taskStore = useTaskStore();
    const sheetV1Store = useSheetV1Store();

    const selectedStatement = computed(() => {
      if (isTenantMode.value) {
        // In tenant mode, the entire issue shares only one SQL statement
        if (create.value) {
          const issueCreate = issue.value as IssueCreate;
          const context = issueCreate.createContext as MigrationContext;
          return (
            useSheetStatementByUID(String(context.detailList[0].sheetId))
              .value || ""
          );
        } else {
          const issueEntity = issue.value as Issue;
          const task = issueEntity.pipeline!.stageList[0].taskList[0];
          const payload =
            task.payload as TaskDatabaseSchemaUpdateGhostSyncPayload;
          return useSheetStatementByUID(String(payload.sheetId)).value || "";
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
                useSheetStatementByUID(String((task as TaskCreate).sheetId))
                  .value || "";
            }
            return statement;
          } else {
            return (
              useSheetStatementByUID(
                String(sheetIdOfTask(task as Task) || UNKNOWN_ID)
              ).value || ""
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
              pipelineId: issueEntity.pipeline!.id,
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
          const db = databaseStore.getDatabaseByUID(String(detail.databaseId!));
          if (!detail.sheetId || detail.sheetId === UNKNOWN_ID) {
            const statement = maybeFormatStatementOnSave(detail.statement, db);
            const sheet = await sheetV1Store.createSheet(db.project, {
              title: issueCreate.name + " - " + db.databaseName,
              content: new TextEncoder().encode(statement),
              visibility: Sheet_Visibility.VISIBILITY_PROJECT,
              source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
              type: Sheet_Type.TYPE_SQL,
              payload: "{}",
            });
            detail.statement = "";
            detail.sheetId = Number(extractSheetUID(sheet.name));
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
            const db = databaseStore.getDatabaseByUID(String(databaseId));
            if (!detail.sheetId || detail.sheetId === UNKNOWN_ID) {
              const statement = maybeFormatStatementOnSave(task.statement, db);
              const sheet = await sheetV1Store.createSheet(db.project, {
                title: issueCreate.name + " - " + db.name,
                content: new TextEncoder().encode(statement),
                visibility: Sheet_Visibility.VISIBILITY_PROJECT,
                source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
                type: Sheet_Type.TYPE_SQL,
                payload: "{}",
              });
              detail.statement = "";
              detail.sheetId = Number(extractSheetUID(sheet.name));
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
