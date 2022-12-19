import { computed, defineComponent } from "vue";
import { cloneDeep } from "lodash-es";
import { provideIssueLogic, useIssueLogic } from "./index";
import {
  flattenTaskList,
  maybeFormatStatementOnSave,
  TaskTypeWithSheetId,
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
  TaskId,
  TaskPatch,
  SheetId,
} from "@/types";
import { useDatabaseStore, useTaskStore } from "@/store";

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
      allowApplyStatementToOtherTasks: baseAllowApplyStatementToOtherTasks,
      applyStatementToOtherTasks: baseApplyStatementToOtherTasks,
      onStatusChanged,
    } = useIssueLogic();
    const databaseStore = useDatabaseStore();
    const taskStore = useTaskStore();

    const patchTask = (
      taskId: TaskId,
      taskPatch: TaskPatch,
      postUpdated?: (updatedTask: Task) => void
    ) => {
      taskStore
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

    const selectedStatement = computed(() => {
      if (isTenantMode.value) {
        // In tenant mode, the entire issue shares only one SQL statement
        if (create.value) {
          const issueCreate = issue.value as IssueCreate;
          const context = issueCreate.createContext as MigrationContext;
          return context.detailList[0].statement;
        } else {
          const issueEntity = issue.value as Issue;
          const task = issueEntity.pipeline.stageList[0].taskList[0];
          const payload =
            task.payload as TaskDatabaseSchemaUpdateGhostSyncPayload;
          return payload.statement;
        }
      } else {
        // In standard pipeline, each ghost-sync task can hold its own
        // statement
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
      }
    });

    const updateStatement = (
      newStatement: string,
      postUpdated?: (updatedTask: Task) => void
    ) => {
      if (isTenantMode.value) {
        if (create.value) {
          // For tenant deploy mode, we apply the statement to all stages and all tasks
          const allTaskList = flattenTaskList<TaskCreate>(issue.value);
          allTaskList.forEach((task) => {
            if (TaskTypeWithStatement.includes(task.type)) {
              task.statement = newStatement;
            }
          });

          const issueCreate = issue.value as IssueCreate;
          const context = issueCreate.createContext as MigrationContext;
          // We also apply it back to the CreateContext
          context.detailList.forEach(
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
      } else {
        if (create.value) {
          const task = selectedTask.value as TaskCreate;
          task.statement = newStatement;
        } else {
          // otherwise, patch the task
          const task = selectedTask.value as Task;
          patchTask(
            task.id,
            {
              statement: maybeFormatStatementOnSave(
                newStatement,
                task.database
              ),
              updatedTs: task.updatedTs,
            },
            postUpdated
          );
        }
      }
    };

    const updateSheetId = (
      sheetId: SheetId | undefined,
      postUpdated?: (updatedTask: Task) => void
    ) => {
      if (isTenantMode.value) {
        if (create.value) {
          // For tenant deploy mode, we apply the statement to all stages and all tasks
          const allTaskList = flattenTaskList<TaskCreate>(issue.value);
          allTaskList.forEach((task) => {
            if (TaskTypeWithSheetId.includes(task.type)) {
              task.sheetId = sheetId;
            }
          });

          const issueCreate = issue.value as IssueCreate;
          const context = issueCreate.createContext as MigrationContext;
          // We also apply it back to the CreateContext
          context.detailList.forEach((detail) => (detail.sheetId = sheetId));
        } else {
          const issueEntity = issue.value as Issue;
          taskStore
            .patchAllTasksInIssue({
              issueId: issueEntity.id,
              pipelineId: issueEntity.pipeline.id,
              taskPatch: {
                sheetId: sheetId,
              },
            })
            .then(() => {
              onStatusChanged(true);
              if (postUpdated) {
                postUpdated(issueEntity.pipeline.stageList[0].taskList[0]);
              }
            });
        }
      } else {
        if (create.value) {
          const task = selectedTask.value as TaskCreate;
          task.sheetId = sheetId;
        } else {
          // otherwise, patch the task
          const task = selectedTask.value as Task;
          patchTask(
            task.id,
            {
              sheetId: sheetId,
              updatedTs: task.updatedTs,
            },
            postUpdated
          );
        }
      }
    };

    const doCreate = () => {
      const issueCreate = cloneDeep(issue.value as IssueCreate);

      if (isTenantMode.value) {
        // for tenant pipeline
        // createContext is up-to-date already
        // so we just format the statement if needed
        const context = issueCreate.createContext as MigrationContext;
        context.detailList.forEach((detail) => {
          const db = databaseStore.getDatabaseById(detail.databaseId!);
          detail.statement = maybeFormatStatementOnSave(detail.statement, db);
        });
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
        syncTaskList.forEach((task) => {
          const { databaseId } = task;
          if (!databaseId) return;
          const detail = detailList.find(
            (detail) => detail.databaseId === databaseId
          );
          if (detail) {
            const db = databaseStore.getDatabaseById(databaseId);
            detail.statement = maybeFormatStatementOnSave(task.statement, db);
            detail.earliestAllowedTs = task.earliestAllowedTs;
          }
        });
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

    const allowApplyStatementToOtherTasks = computed(() => {
      // We are never allowed to "apply statement to other stages" in tenant mode.
      if (isTenantMode.value) return false;
      return baseAllowApplyStatementToOtherTasks.value;
    });
    const applyStatementToOtherTasks = (statement: string) => {
      if (!allowApplyStatementToOtherTasks.value) return;
      return baseApplyStatementToOtherTasks(statement);
    };

    const logic = {
      ...useCommonLogic(),
      selectedStatement,
      doCreate,
      allowApplyTaskStatusTransition,
      allowApplyStatementToOtherTasks,
      applyStatementToOtherTasks,
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
