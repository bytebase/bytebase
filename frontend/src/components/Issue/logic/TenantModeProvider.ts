import { cloneDeep, head } from "lodash-es";
import { computed, defineComponent } from "vue";
import { useRoute } from "vue-router";
import {
  useSheetV1Store,
  useProjectV1Store,
  useTaskStore,
  useSheetStatementByUID,
  useDatabaseV1Store,
  useCurrentUserV1,
} from "@/store";
import { getProjectPathByLegacyProject } from "@/store/modules/v1/common";
import {
  Issue,
  IssueCreate,
  Task,
  TaskCreate,
  TaskDatabaseSchemaUpdatePayload,
  MigrationContext,
  SheetId,
  UNKNOWN_ID,
  MigrationDetail,
} from "@/types";
import {
  Sheet_Visibility,
  Sheet_Source,
  Sheet_Type,
} from "@/types/proto/v1/sheet_service";
import {
  extractUserUID,
  getBacktracePayloadWithIssue,
  hasWorkspacePermissionV1,
  sheetIdOfTask,
} from "@/utils";
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
    const route = useRoute();
    const currentUser = useCurrentUserV1();
    const databaseStore = useDatabaseV1Store();
    const sheetV1Store = useSheetV1Store();
    const projectV1Store = useProjectV1Store();
    const taskStore = useTaskStore();

    const isGroupingIssue = computed(() => {
      return route.query.databaseGroupName;
    });

    const allowEditStatement = computed(() => {
      if (create.value) {
        return true;
      }

      const tasks = flattenTaskList<Task>(issue.value);
      if (!tasks.every((task) => isTaskEditable(task))) {
        return false;
      }

      if (
        hasWorkspacePermissionV1(
          "bb.permission.workspace.manage-issue",
          currentUser.value.userRole
        )
      ) {
        return true;
      }

      const creatorUID = String((issue.value as Issue).creator.id);
      if (creatorUID === extractUserUID(currentUser.value.name)) {
        return true;
      }
      return false;
    });

    // In tenant mode, the entire issue shares only one SQL statement
    const selectedStatement = computed(() => {
      if (create.value) {
        if (isGroupingIssue.value) {
          return selectedTask.value.statement || "";
        }

        const issueCreate = issue.value as IssueCreate;
        const context = issueCreate.createContext as MigrationContext;
        return context.detailList[0].statement;
      } else {
        const issueEntity = issue.value as Issue;
        const task = issueEntity.pipeline!.stageList[0].taskList[0];
        const payload = task.payload as TaskDatabaseSchemaUpdatePayload;
        const selectedTaskSheetId = sheetIdOfTask(selectedTask.value as Task);
        return (
          useSheetStatementByUID(
            String(selectedTaskSheetId || payload.sheetId || UNKNOWN_ID)
          ).value || ""
        );
      }
    });

    const updateStatement = async (newStatement: string) => {
      if (create.value) {
        // For grouping issue, we don't allow to modify statement.
        if (isGroupingIssue.value) {
          return;
        }

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
        const sheet = await sheetV1Store.createSheet(
          getProjectPathByLegacyProject(issueEntity.project),
          {
            title: `Sheet for issue #${issueEntity.id}`,
            content: new TextEncoder().encode(newStatement),
            visibility: Sheet_Visibility.VISIBILITY_PROJECT,
            source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
            type: Sheet_Type.TYPE_SQL,
            payload: JSON.stringify(
              getBacktracePayloadWithIssue(issue.value as Issue)
            ),
          }
        );
        updateSheetId(Number(extractUserUID(sheet.name)));
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
            pipelineId: issueEntity.pipeline!.id,
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
      // For those database group issues, we create issue directly instead of creating sheets.
      if (route.query.databaseGroupName) {
        const taskList = flattenTaskList(issueCreate);
        context.detailList = [];
        for (const task of taskList) {
          const migrationDetail: MigrationDetail = {
            migrationType: detail.migrationType,
            earliestAllowedTs: detail.earliestAllowedTs,
            databaseGroupName: route.query.databaseGroupName as string,
            databaseId: (task as any).databaseId,
            statement: (task as any).statement,
            sheetId: (task as any).sheetId,
          };
          const payload = (task as Task).payload;
          if (payload && (payload as any).schemaGroupName) {
            migrationDetail.schemaGroupName = (payload as any).schemaGroupName;
          }
          context.detailList.push(migrationDetail);
        }
        createIssue(issueCreate);
        return;
      }

      const db = databaseStore.getDatabaseByUID(String(detail.databaseId!));
      if (!detail.sheetId || detail.sheetId === UNKNOWN_ID) {
        const statement = maybeFormatStatementOnSave(detail.statement, db);
        const project = await projectV1Store.getOrFetchProjectByUID(
          `${issueCreate.projectId}`
        );
        const sheet = await sheetV1Store.createSheet(project.name, {
          title: issueCreate.name + " - " + db.name,
          content: new TextEncoder().encode(statement),
          visibility: Sheet_Visibility.VISIBILITY_PROJECT,
          source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
          type: Sheet_Type.TYPE_SQL,
          payload: "{}",
        });
        detail.statement = "";
        detail.sheetId = Number(extractUserUID(sheet.name));
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
