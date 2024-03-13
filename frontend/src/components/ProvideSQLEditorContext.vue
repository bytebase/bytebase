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
import { head } from "lodash-es";
import { nextTick, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
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
import {
  DEFAULT_PROJECT_V1_NAME,
  DEFAULT_SQL_EDITOR_TAB_MODE,
  SQLEditorConnection,
  UNKNOWN_ID,
  UNKNOWN_USER_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  extractProjectResourceName,
  getSheetStatement,
  hasProjectPermissionV2,
  idFromSlug,
  isDatabaseV1Queryable,
  isWorksheetReadableV1,
  projectNameFromSheetSlug,
  suggestedTabTitleForSQLEditorConnection,
  worksheetNameFromSlug,
} from "@/utils";
import { useSheetContext } from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "@/views/sql-editor/context";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const editorStore = useSQLEditorStore();
const worksheetStore = useWorkSheetStore();
const tabStore = useSQLEditorTabStore();
const { isFetching: isFetchingWorksheet } = useSheetContext();
const { filter } = useFilterStore();
const { events: editorEvents, maybeSwitchProject } = useSQLEditorContext();

const initializeProjects = async () => {
  const projectInQuery = route.query.project as string;
  if (typeof projectInQuery === "string" && projectInQuery) {
    const project = `projects/${projectInQuery}`;
    editorStore.strictProject = true;
    editorStore.project = project;
    await projectStore.getOrFetchProjectByName(project, true /* silent */);
  } else {
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
  const instanceStore = useInstanceV1Store();
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
  const filter = project ? `project == "${project}"` : "";

  // `databaseList` is the database list accessible by current user.
  // Only accessible instances and databases will be listed in the tree.
  const databaseList = (
    await databaseStore.searchOrListDatabases({
      parent: "instances/-",
      filter,
      permission: "bb.databases.query",
    })
  ).filter((db) => db.syncState === State.ACTIVE);

  editorStore.databaseList = databaseList;
};

const prepareSheet = async () => {
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

  // Won't set connection to the worksheet's database
  // since we are considering to unbind worksheets and databases
  // const connection = emptySQLEditorConnection();
  // if (sheet.database) {
  //   try {
  //     const database = await databaseStore.getOrFetchDatabaseByName(
  //       sheet.database,
  //       true /* silent */
  //     );
  //     if (database.uid !== String(UNKNOWN_ID)) {
  //       connection.instance = database.instance;
  //       connection.database = database.name;
  //     }
  //   } catch {
  //     // Skip.
  //   }
  // }
  // Open the sheet in a new tab otherwise.
  tabStore.addTab();

  tabStore.updateCurrentTab({
    sheet: sheet.name,
    title: sheet.title,
    statement: getSheetStatement(sheet),
    status: "CLEAN",
    // connection,
  });

  return true;
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

const prepareConnectionSlug = async () => {
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

  if (await prepareSheet()) {
    return;
  }

  if (await prepareConnectionSlug()) {
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
      });
      return;
    }
  }

  // Keep disconnected otherwise
  // We don't need to `selectOrAddTempTab` here since we already have the
  // default tab.
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

  initializeConnectionFromQuery();
});

useEmitteryEventListener;
</script>
