<template>
  <teleport to="#sql-editor-debug">
    <li>[ProvideContext]project: {{ editorStore.project }}</li>
    <li>
      [ProvideContext]projectContextReady:
      {{ editorStore.projectContextReady }}
    </li>
    <li>
      [ProvideContext]allowViewALLProjects:
      {{ editorStore.allowViewALLProjects }}
    </li>
  </teleport>

  <Suspense>
    <ProvideAIContext>
      <router-view />
    </ProvideAIContext>
  </Suspense>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { debounce, head, omit } from "lodash-es";
import { computed, nextTick, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import { ProvideAIContext } from "@/plugins/ai";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
} from "@/router/sqlEditor";
import {
  pushNotification,
  useDatabaseV1Store,
  usePolicyV1Store,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import { migrateLegacyCache } from "@/store/modules/sqlEditor/legacy/migration";
import {
  DEFAULT_PROJECT_NAME,
  DEFAULT_SQL_EDITOR_TAB_MODE,
  isValidDatabaseName,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  emptySQLEditorConnection,
  extractInstanceResourceName,
  extractProjectResourceName,
  extractWorksheetConnection,
  extractWorksheetUID,
  getDefaultPagination,
  getSheetStatement,
  isDatabaseV1Queryable,
  isWorksheetReadableV1,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import {
  type AsidePanelTab,
  useSQLEditorContext,
} from "@/views/sql-editor/context";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const editorStore = useSQLEditorStore();
const worksheetStore = useWorkSheetStore();
const tabStore = useSQLEditorTabStore();
const policyStore = usePolicyV1Store();
const {
  asidePanelTab,
  events: editorEvents,
  maybeSwitchProject,
} = useSQLEditorContext();

useRouteChangeGuard(
  computed(() => {
    return (
      tabStore.openTabList.find((tab) => tab.status === "DIRTY") !== undefined
    );
  }),
  `${t("sql-editor.tab.unsaved-worksheet")} ${t("common.leave-without-saving")}`
);

const fallbackToFirstProject = async () => {
  const { projects } = await projectStore.fetchProjectList({
    pageSize: getDefaultPagination(),
    filter: {
      excludeDefault: true,
    },
  });
  return head(projects)?.name ?? DEFAULT_PROJECT_NAME;
};

const initializeProject = async () => {
  const projectInQuery = route.query.project as string;
  const projectInParams = route.params.project as string;
  let project: string = "";
  let initializeSuccess = false;

  if (typeof projectInQuery === "string" && projectInQuery) {
    // Legacy "?project={project}"
    project = `projects/${projectInQuery}`;
  } else if (typeof projectInParams === "string" && projectInParams) {
    // "/sql-editor/projects/{project}"
    project = `projects/${projectInParams}`;
  } else {
    // plain "/sql-editor"
    project = editorStore.storedLastViewedProject;
  }

  initializeSuccess = !!(await maybeSwitchProject(project));
  if (!initializeSuccess) {
    // Maybe the cached project name is valid, but users cannot get it any more (removed, not permission, etc).
    // So we need to fallback to the first accessible project.
    project = await fallbackToFirstProject();
    initializeSuccess = !!(await maybeSwitchProject(project));
  }

  if (!initializeSuccess) {
    // clear last visit project.
    editorStore.setProject("");
  }
  return editorStore.project;
};

const switchWorksheet = async (sheetName: string) => {
  const openedSheetTab = tabStore.getTabByWorksheet(sheetName);

  const sheet = await worksheetStore.getOrFetchWorksheetByName(sheetName);
  if (!sheet) {
    if (openedSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      tabStore.updateTab(openedSheetTab.id, {
        worksheet: "",
        status: "DIRTY",
      });
    }
    return false;
  }
  if (!isWorksheetReadableV1(sheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return false;
  }

  const connection = await extractWorksheetConnection(sheet);
  tabStore.addTab({
    id: openedSheetTab?.id,
    connection,
    worksheet: sheet.name,
    title: sheet.title,
    statement: getSheetStatement(sheet),
    status: "CLEAN",
  });
  return true;
};

const prepareSheet = async () => {
  const projectId = route.params.project;
  const sheetId = route.params.sheet;
  if (typeof projectId !== "string" || !projectId) {
    return false;
  }
  if (typeof sheetId !== "string" || !sheetId) {
    return false;
  }

  await maybeSwitchProject(`projects/${projectId}`);
  return await switchWorksheet(`worksheets/${sheetId}`);
};

const prepareConnectionParams = async () => {
  if (
    ![SQL_EDITOR_INSTANCE_MODULE, SQL_EDITOR_DATABASE_MODULE].includes(
      route.name as string
    )
  ) {
    return false;
  }

  const databaseName = `instances/${route.params.instance}/databases/${route.params.database}`;
  if (!isValidDatabaseName(databaseName)) {
    return false;
  }

  const database =
    await useDatabaseV1Store().getOrFetchDatabaseByName(databaseName);
  await maybeSwitchProject(database.project);
  if (!isDatabaseV1Queryable(database)) {
    const tabs = tabStore.openTabList.filter(
      (tab) => tab.connection.database === database.name
    );
    for (const tab of tabs) {
      tabStore.closeTab(tab.id);
    }
    return false;
  }
  // connected to db
  const connection = {
    instance: database.instance,
    database: database.name,
  };
  tabStore.addTab({
    connection,
    mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    title: suggestedTabTitleForSQLEditorConnection(connection),
  });
  return true;
};

const initializeConnectionFromQuery = async () => {
  // Priority:
  // 1. idFromSlug in sheetSlug
  // 2. instanceId and databaseId in connectionSlug
  // 3. database in global filter
  // 4. disconnected
  if (await prepareSheet()) {
    return;
  }
  if (await prepareConnectionParams()) {
    return;
  }

  // Keep disconnected otherwise
  // We don't need to `selectOrAddTempTab` here since we already have the
  // default tab.
};

// Keep the URL synced with connection
// 1. /sql-editor/projects/{project}/sheets/{sheet}                            - saved sheets
// 2. /sql-editor/projects/{project}/instances/{instance}/databases/{database} - unsaved tabs
// 3. /sql-editor/projects/{project}                                           - disconnected tabs
const syncURLWithConnection = () => {
  const connection = computed(
    () => tabStore.currentTab?.connection ?? emptySQLEditorConnection()
  );
  watch(
    [
      () => editorStore.project,
      () => tabStore.currentTab?.worksheet,
      () => connection.value?.instance,
      () => connection.value?.database,
      () => connection.value?.schema,
      () => connection.value?.table,
    ],
    async ([
      projectName,
      sheetName,
      instanceName,
      databaseName,
      schema,
      table,
    ]) => {
      const query = omit(
        route.query,
        "filter",
        "project",
        "schema",
        "database"
      );

      if (sheetName) {
        const sheet = worksheetStore.getWorksheetByName(sheetName);
        if (sheet) {
          router.replace({
            name: SQL_EDITOR_WORKSHEET_MODULE,
            params: {
              project: extractProjectResourceName(sheet.project),
              sheet: extractWorksheetUID(sheet.name),
            },
            query,
          });
          return;
        } else {
          const tab = tabStore.currentTab;
          if (tab) {
            tab.worksheet = "";
            tab.status = "DIRTY";
          }
        }
      }
      if (isValidDatabaseName(databaseName)) {
        const database =
          await databaseStore.getOrFetchDatabaseByName(databaseName);
        if (!isDatabaseV1Queryable(database)) {
          return router.replace({
            name: SQL_EDITOR_PROJECT_MODULE,
            params: {
              project: extractProjectResourceName(database.project),
            },
            query,
          });
        }
        if (schema) {
          query.schema = schema;
        }
        if (table) {
          query.table = table;
          query.schema = schema ?? "";
        }
        router.replace({
          name: SQL_EDITOR_DATABASE_MODULE,
          params: {
            project: extractProjectResourceName(database.project),
            instance: extractInstanceResourceName(database.instance),
            database: database.databaseName,
          },
          query,
        });
        return;
      }
      if (isValidInstanceName(instanceName)) {
        if (table) {
          query.table = table;
          query.schema = schema ?? "";
        }
        router.replace({
          name: SQL_EDITOR_INSTANCE_MODULE,
          params: {
            project: extractProjectResourceName(editorStore.project),
            instance: extractInstanceResourceName(instanceName),
          },
          query,
        });
        return;
      }
      if (isValidProjectName(projectName)) {
        router.replace({
          name: SQL_EDITOR_PROJECT_MODULE,
          params: {
            project: extractProjectResourceName(projectName),
          },
          query,
        });
        return;
      }
      router.replace({
        name: SQL_EDITOR_HOME_MODULE,
      });
    },
    { immediate: true }
  );
};

const restoreLastVisitedSidebarTab = () => {
  const storedLastVisitedSidebarTab = useLocalStorage<AsidePanelTab>(
    "bb.sql-editor.sidebar.last-visited-tab",
    "WORKSHEET"
  );
  asidePanelTab.value = storedLastVisitedSidebarTab.value;

  watch(
    asidePanelTab,
    debounce((tab: AsidePanelTab) => {
      storedLastVisitedSidebarTab.value = tab;
    }, 100)
  );
};

onMounted(async () => {
  editorStore.projectContextReady = false;
  const [_, project] = await Promise.all([
    policyStore.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    initializeProject(),
  ]);

  await migrateLegacyCache();
  await tabStore.initProject(project);
  await initializeConnectionFromQuery();
  syncURLWithConnection();
});

useEmitteryEventListener(
  editorEvents,
  "project-context-ready",
  ({ project }) => {
    if (!project) {
      return;
    }
    nextTick(() => {
      restoreLastVisitedSidebarTab();
    });
  }
);
</script>
