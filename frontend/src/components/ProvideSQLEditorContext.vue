<template>
  <teleport to="#sql-editor-debug">
    <li>[ProvideContext]project: {{ editorStore.project }}</li>
    <li>[ProvideContext]strictProject: {{ editorStore.strictProject }}</li>
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
import { ProvideAIContext } from "@/plugins/ai";
import {
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/router/sqlEditor";
import {
  usePolicyV1Store,
  useProjectV1Store,
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
  pushNotification,
} from "@/store";
import type { SQLEditorConnection } from "@/types";
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
  extractProjectResourceName,
  getSheetStatement,
  hasProjectPermissionV2,
  isDatabaseV1Queryable,
  isWorksheetReadableV1,
  projectNameFromSheetSlug,
  suggestedTabTitleForSQLEditorConnection,
  worksheetNameFromSlug,
  extractWorksheetUID,
  extractInstanceResourceName,
  getDefaultPagination,
} from "@/utils";
import {
  extractWorksheetConnection,
  useSheetContext,
} from "@/views/sql-editor/Sheet";
import {
  useSQLEditorContext,
  type AsidePanelTab,
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
const { isFetching: isFetchingWorksheet } = useSheetContext();
const {
  asidePanelTab,
  events: editorEvents,
  maybeSwitchProject,
} = useSQLEditorContext();

const fallbackToFirstProject = async () => {
  const { projects } = await projectStore.fetchProjectList({
    pageSize: getDefaultPagination(),
    filter: {
      excludeDefault: true,
    },
  });
  return head(projects)?.name ?? DEFAULT_PROJECT_NAME;
};

const initProject = async (project: string) => {
  try {
    await projectStore.getOrFetchProjectByName(project);
    return true;
  } catch {
    // nothing
  }
  return false;
};

const initializeProjects = async () => {
  const projectInQuery = route.query.project as string;
  const projectInParams = route.params.project as string;
  let project: string = "";
  let initializeSuccess = false;

  if (typeof projectInQuery === "string" && projectInQuery) {
    // Legacy "?project={project}"
    project = `projects/${projectInQuery}`;
    editorStore.strictProject = true;
  } else if (typeof projectInParams === "string" && projectInParams) {
    // "/sql-editor/projects/{project}"
    project = `projects/${projectInParams}`;
    editorStore.strictProject = "strict" in route.query;
  } else {
    // plain "/sql-editor"
    project = editorStore.storedLastViewedProject;
    editorStore.strictProject = false;
  }

  if (isValidProjectName(project)) {
    initializeSuccess = await initProject(project);
  }
  if (!initializeSuccess) {
    // Maybe the cached project name is valid, but users cannot get it any more (removed, not permission, etc).
    // So we need to fallback to the first accessible project.
    project = await fallbackToFirstProject();
    initializeSuccess = await initProject(project);
  }

  if (initializeSuccess) {
    editorStore.project = project;
    tabStore.maybeInitProject(editorStore.project);
  } else {
    editorStore.project = "";
    editorStore.storedLastViewedProject = "";
  }
};

const handleProjectSwitched = async () => {
  const { project } = editorStore;
  if (project) {
    await projectStore.getOrFetchProjectByName(project, true /* silent */);
  }
};

const connect = (connection: SQLEditorConnection) => {
  tabStore.selectOrAddSimilarNewTab(
    {
      connection,
      worksheet: "",
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    },
    /* beside */ false,
    /* defaultTitle */ suggestedTabTitleForSQLEditorConnection(connection),
    /* ignoreMode */ true
  );
  if (tabStore.currentTab?.mode === "ADMIN") {
    // Don't enter ADMIN mode initially
    tabStore.updateCurrentTab({
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    });
  }
};

const prepareSheetLegacy = async () => {
  const sheetSlug = (route.params.sheetSlug as string) || "";
  if (!sheetSlug) {
    return false;
  }

  const projectName = projectNameFromSheetSlug(sheetSlug);

  try {
    const project = await projectStore.getOrFetchProjectByName(projectName);
    if (!hasProjectPermissionV2(project, "bb.sql.select")) {
      return false;
    }
  } catch {
    // Nothing
  }
  await maybeSwitchProject(projectName);

  const sheetName = worksheetNameFromSlug(sheetSlug);
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.worksheet == sheetName
  );

  isFetchingWorksheet.value = true;
  const sheet = await worksheetStore.getOrFetchWorksheetByName(sheetName);
  isFetchingWorksheet.value = false;

  if (!sheet) {
    if (openingSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      openingSheetTab.worksheet = "";
      openingSheetTab.status = "DIRTY";
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

  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    // and don't touch it
    tabStore.setCurrentTabId(openingSheetTab.id);
    return true;
  }

  // Open the sheet in a new tab otherwise.
  tabStore.addTab({
    connection: extractWorksheetConnection(sheet),
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

  const projectName = `projects/${projectId}`;
  const sheetName = `worksheets/${sheetId}`;

  const project = await projectStore.getOrFetchProjectByName(projectName);
  if (!hasProjectPermissionV2(project, "bb.sql.select")) {
    return false;
  }
  await maybeSwitchProject(projectName);

  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.worksheet == sheetName
  );

  isFetchingWorksheet.value = true;
  const sheet = await worksheetStore.getOrFetchWorksheetByName(sheetName);
  isFetchingWorksheet.value = false;

  if (!sheet) {
    if (openingSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      openingSheetTab.worksheet = "";
      openingSheetTab.status = "DIRTY";
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

  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    // and don't touch it
    tabStore.setCurrentTabId(openingSheetTab.id);
    return true;
  }

  // Open the sheet in a new tab otherwise.
  tabStore.addTab({
    connection: extractWorksheetConnection(sheet),
    worksheet: sheet.name,
    title: sheet.title,
    statement: getSheetStatement(sheet),
    status: "CLEAN",
  });

  return true;
};

const prepareConnectionParams = async () => {
  if (
    ![SQL_EDITOR_INSTANCE_MODULE, SQL_EDITOR_DATABASE_MODULE].includes(
      route.name as string
    )
  ) {
    return false;
  }
  const instanceName = route.params.instance;
  const databaseName = route.params.database;
  if (
    typeof instanceName !== "string" ||
    !instanceName ||
    typeof databaseName !== "string" ||
    !databaseName
  ) {
    return false;
  }

  const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
    `instances/${instanceName}/databases/${databaseName}`
  );
  if (isValidDatabaseName(database.name)) {
    if (!isDatabaseV1Queryable(database)) {
      router.push({
        name: "error.403",
      });
    }

    // connected to db
    await maybeSwitchProject(database.project);
    connect({
      instance: database.instance,
      database: database.name,
    });
    return true;
  }
  return false;
};

const initializeConnectionFromQuery = async () => {
  // Priority:
  // 1. idFromSlug in sheetSlug
  // 2. instanceId and databaseId in connectionSlug
  // 3. database in global filter
  // 4. disconnected

  if (await prepareSheetLegacy()) {
    return true;
  }
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

      if (editorStore.strictProject) {
        // The API is weird
        // `query.strict = null` will generate "&strict" in the query string
        // while `query.strict = ""` will generate "&strict=" in the query string
        // and we prefer the shorter one
        query.strict = null;
      }
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
      if (databaseName) {
        const database =
          await databaseStore.getOrFetchDatabaseByName(databaseName);
        if (isValidDatabaseName(database.name)) {
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
      }
      if (instanceName) {
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
      }
      if (projectName) {
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
  await Promise.all([
    policyStore.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    initializeProjects(),
  ]);
  tabStore.maybeInitProject(editorStore.project);
  editorStore.projectContextReady = true;
  nextTick(() => {
    editorEvents.emit("project-context-ready", {
      project: editorStore.project,
    });
    restoreLastVisitedSidebarTab();
  });

  watch(
    () => editorStore.project,
    async () => {
      editorStore.projectContextReady = false;
      await handleProjectSwitched();
      tabStore.maybeInitProject(editorStore.project);
      editorStore.projectContextReady = true;
      nextTick(() => {
        editorEvents.emit("project-context-ready", {
          project: editorStore.project,
        });
      });
    }
  );

  await initializeConnectionFromQuery();
  syncURLWithConnection();
});
</script>
