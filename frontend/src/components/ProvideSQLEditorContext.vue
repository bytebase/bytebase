<template>
  <teleport to="#sql-editor-debug">
    <li>[ProvideContext]project: {{ editorStore.project }}</li>
    <li>[ProvideContext]strictProject: {{ editorStore.strictProject }}</li>
    <li>
      [ProvideContext]projectContextReady:
      {{ editorStore.projectContextReady }}
    </li>
    <li>
      [ProvideContext]databaseCount: {{ editorStore.databaseList.length }}
    </li>
    <li>
      [ProvideContext]allowViewALLProjects:
      {{ editorStore.allowViewALLProjects }}
    </li>
  </teleport>
  <slot />
</template>

<script lang="ts" setup>
import { head, omit } from "lodash-es";
import { computed, nextTick, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_INSTANCE_MODULE,
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_WORKSHEET_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/router/sqlEditor";
import {
  useInstanceV1Store,
  useProjectV1Store,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
  pushNotification,
  useFilterStore,
} from "@/store";
import type { SQLEditorConnection } from "@/types";
import {
  DEFAULT_PROJECT_V1_NAME,
  DEFAULT_SQL_EDITOR_TAB_MODE,
  UNKNOWN_ID,
  UNKNOWN_USER_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  emptySQLEditorConnection,
  extractProjectResourceName,
  getSheetStatement,
  hasProjectPermissionV2,
  idFromSlug,
  isDatabaseV1Queryable,
  isWorksheetReadableV1,
  projectNameFromSheetSlug,
  suggestedTabTitleForSQLEditorConnection,
  worksheetNameFromSlug,
  extractWorksheetUID,
  extractInstanceResourceName,
} from "@/utils";
import {
  extractWorksheetConnection,
  useSheetContext,
} from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const instanceStore = useInstanceV1Store();
const editorStore = useSQLEditorStore();
const worksheetStore = useWorkSheetStore();
const tabStore = useSQLEditorTabStore();
const { isFetching: isFetchingWorksheet } = useSheetContext();
const { filter } = useFilterStore();
const { events: editorEvents, maybeSwitchProject } = useSQLEditorContext();

const initializeProjects = async () => {
  const initProject = async (project: string) => {
    try {
      await projectStore.getOrFetchProjectByName(project, /* !silent */ false);
      editorStore.project = project;
      return true;
    } catch {
      // nothing
    }
    return false;
  };

  const projectInQuery = route.query.project as string;
  const projectInParams = route.params.project as string;
  if (typeof projectInQuery === "string" && projectInQuery) {
    // Legacy "?project={project}"
    const project = `projects/${projectInQuery}`;
    editorStore.strictProject = true;
    await initProject(project);
  } else if (typeof projectInParams === "string" && projectInParams) {
    // "/sql-editor/projects/{project}"
    const project = `projects/${projectInParams}`;
    editorStore.strictProject = "strict" in route.query;
    await initProject(project);
  } else {
    // plain "/sql-editor"
    const projectList = await projectStore.fetchProjectList(false);
    const lastView = editorStore.storedLastViewedProject;
    if (
      lastView &&
      projectList.findIndex((proj) => proj.name === lastView) >= 0
    ) {
      editorStore.project = lastView;
    } else {
      const projectListWithoutDefaultProject = projectList.filter(
        (proj) => proj.name !== DEFAULT_PROJECT_V1_NAME
      );
      editorStore.project =
        head(projectListWithoutDefaultProject)?.name ??
        head(projectList)?.name ??
        "";
    }
    editorStore.strictProject = false;
  }

  tabStore.maybeInitProject(editorStore.project);
};

const handleProjectSwitched = async () => {
  const { project } = editorStore;
  if (project) {
    await projectStore.getOrFetchProjectByName(project, true /* silent */);
  } else {
    await projectStore.fetchProjectList(false /* !showDeleted */);
  }
  tabStore.maybeInitProject(project);
};

const prepareInstances = async () => {
  const { project } = editorStore;
  if (project) {
    await instanceStore.fetchProjectInstanceList(
      extractProjectResourceName(project)
    );
  } else {
    await instanceStore.fetchInstanceList();
  }
};

const prepareDatabases = async () => {
  // It will also be called when user logout
  if (me.value.name === UNKNOWN_USER_NAME) {
    return;
  }
  const { project } = editorStore;
  const filters = [`instance = "instances/-"`];
  if (project) {
    filters.push(`project = "${project}"`);
  }
  // `databaseList` is the database list accessible by current user.
  // Only accessible instances and databases will be listed in the tree.
  const databaseList = (
    await databaseStore.searchDatabases({
      filter: filters.join(" && "),
      permission: "bb.databases.query",
    })
  ).filter((db) => db.syncState === State.ACTIVE);

  editorStore.databaseList = databaseList;
};

const connect = (connection: SQLEditorConnection) => {
  tabStore.selectOrAddSimilarNewTab(
    {
      connection,
      sheet: "",
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    },
    /* beside */ false,
    /* defaultTitle */ suggestedTabTitleForSQLEditorConnection(connection)
  );
};

const prepareSheetLegacy = async () => {
  const sheetSlug = (route.params.sheetSlug as string) || "";
  if (!sheetSlug) {
    return false;
  }

  const projectName = projectNameFromSheetSlug(sheetSlug);

  try {
    const project = await projectStore.getOrFetchProjectByName(projectName);
    if (!hasProjectPermissionV2(project, me.value, "bb.databases.query")) {
      return false;
    }
  } catch {
    // Nothing
  }
  await maybeSwitchProject(projectName);

  const sheetName = worksheetNameFromSlug(sheetSlug);
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheet == sheetName
  );

  isFetchingWorksheet.value = true;
  const sheet = await worksheetStore.getOrFetchSheetByName(sheetName);
  isFetchingWorksheet.value = false;

  if (!sheet) {
    if (openingSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      openingSheetTab.sheet = "";
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
    sheet: sheet.name,
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
  if (!hasProjectPermissionV2(project, me.value, "bb.databases.query")) {
    return false;
  }
  await maybeSwitchProject(projectName);

  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheet == sheetName
  );

  isFetchingWorksheet.value = true;
  const sheet = await worksheetStore.getOrFetchSheetByName(sheetName);
  isFetchingWorksheet.value = false;

  if (!sheet) {
    if (openingSheetTab) {
      // If a sheet is open in a tab but it returns 404 NOT_FOUND
      // that means the sheet has been deleted somewhere else.
      // We need to turn the sheet to an unsaved tab.
      openingSheetTab.sheet = "";
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
    sheet: sheet.name,
    title: sheet.title,
    statement: getSheetStatement(sheet),
    status: "CLEAN",
  });

  return true;
};

const prepareConnectionSlugLegacy = async () => {
  const connectionSlug = (route.params.connectionSlug as string) || "";
  const [instanceSlug, databaseSlug = ""] = connectionSlug.split("_");
  const instanceId = Number(idFromSlug(instanceSlug));
  const databaseId = Number(idFromSlug(databaseSlug));

  if (Number.isNaN(instanceId) && Number.isNaN(databaseId)) {
    return false;
  }
  if (instanceId === 0 || databaseId === 0) {
    return false;
  }

  if (Number.isNaN(databaseId)) {
    // connected to instance
    const instance = await useInstanceV1Store().getOrFetchInstanceByUID(
      String(instanceId)
    );
    if (instance.uid !== String(UNKNOWN_ID)) {
      connect({
        instance: instance.name,
        database: "",
      });
      return true;
    }
  } else {
    const database = await useDatabaseV1Store().getOrFetchDatabaseByUID(
      String(databaseId)
    );
    if (database.uid !== String(UNKNOWN_ID)) {
      if (!isDatabaseV1Queryable(database, me.value)) {
        router.push({
          name: "error.403",
        });
      }

      // connected to db
      await maybeSwitchProject(database.project);
      connect({
        instance: database.instance,
        database: database.name,
        schema: filter.schema,
        table: filter.table,
      });
      return true;
    }
  }
  return false;
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
  if (typeof instanceName !== "string" || !instanceName) {
    return false;
  }

  if (typeof databaseName !== "string" || !databaseName) {
    // connected to instance
    const instance = await useInstanceV1Store().getOrFetchInstanceByName(
      `instances/${instanceName}`
    );
    if (instance.uid !== String(UNKNOWN_ID)) {
      connect({
        instance: instance.name,
        database: "",
      });
      return true;
    }
  } else {
    const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
      `instances/${instanceName}/databases/${databaseName}`
    );
    if (database.uid !== String(UNKNOWN_ID)) {
      if (!isDatabaseV1Queryable(database, me.value)) {
        router.push({
          name: "error.403",
        });
      }

      // connected to db
      await maybeSwitchProject(database.project);
      connect({
        instance: database.instance,
        database: database.name,
        schema: filter.schema,
        table: filter.table,
      });
      return true;
    }
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

  if (await prepareConnectionSlugLegacy()) {
    return true;
  }
  if (await prepareConnectionParams()) {
    return;
  }

  if (filter.database) {
    const database = await databaseStore.getOrFetchDatabaseByName(
      filter.database,
      /* silent */ true
    );
    if (database.uid !== String(UNKNOWN_ID)) {
      await maybeSwitchProject(database.project);
      connect({
        instance: database.instance,
        database: database.name,
        schema: filter.schema,
        table: filter.table,
      });
      return;
    }
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
      () => tabStore.currentTab?.sheet,
      () => connection.value?.instance,
      () => connection.value?.database,
      () => connection.value?.schema,
      () => connection.value?.table,
    ],
    ([projectName, sheetName, instanceName, databaseName, schema, table]) => {
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
        const sheet = worksheetStore.getSheetByName(sheetName);
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
            tab.sheet = "";
            tab.status = "DIRTY";
          }
        }
      }
      if (databaseName) {
        const database = databaseStore.getDatabaseByName(databaseName);
        if (database.uid !== String(UNKNOWN_ID)) {
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
        const instance = instanceStore.getInstanceByName(instanceName);
        if (instance.uid !== String(UNKNOWN_ID)) {
          if (table) {
            query.table = table;
            query.schema = schema ?? "";
          }
          router.replace({
            name: SQL_EDITOR_INSTANCE_MODULE,
            params: {
              project: extractProjectResourceName(editorStore.project),
              instance: extractInstanceResourceName(instance.name),
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

onMounted(async () => {
  editorStore.projectContextReady = false;
  await initializeProjects();
  await prepareInstances();
  await prepareDatabases();
  tabStore.maybeInitProject(editorStore.project);
  editorStore.projectContextReady = true;
  nextTick(() => {
    editorEvents.emit("project-context-ready", {
      project: editorStore.project,
    });
  });

  watch(
    () => editorStore.project,
    async () => {
      editorStore.projectContextReady = false;
      await handleProjectSwitched();
      await prepareInstances();
      await prepareDatabases();
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

useEmitteryEventListener;
</script>
