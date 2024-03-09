<template>
  <slot />
  <ul
    class="text-xs font-mono max-h-[33vh] max-w-[40vw] overflow-auto fixed bottom-0 right-0 p-2 bg-white/50 border border-gray-400"
  >
    <li>project: {{ sqlEditorStore.project }}</li>
    <li>strictProject: {{ sqlEditorStore.strictProject }}</li>
    <li>
      projectContextReady:
      {{ sqlEditorStore.projectContextReady }}
    </li>
    <li>databaseCount: {{ sqlEditorStore.databaseList.length }}</li>
    <li>allowViewALLProjects: {{ sqlEditorStore.allowViewALLProjects }}</li>
  </ul>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { head } from "lodash-es";
import { NSpin } from "naive-ui";
import { onMounted, computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  SQL_EDITOR_DETAIL_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_SHARE_MODULE,
} from "@/router/sqlEditor";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  usePolicyV1Store,
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
  useSQLEditorStore,
  useTabStore,
  pushNotification,
  useCurrentUserV1,
  useWorkSheetStore,
  useDatabaseV1Store,
  useFilterStore,
  useSQLEditorV2Store,
} from "@/store";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  Connection,
  CoreTabInfo,
  DEFAULT_PROJECT_V1_NAME,
  TabMode,
  UNKNOWN_USER_NAME,
  unknownProject,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import {
  emptyConnection,
  idFromSlug,
  worksheetNameFromSlug,
  projectNameFromSheetSlug,
  worksheetSlugV1,
  connectionV1Slug as makeConnectionV1Slug,
  isWorksheetReadableV1,
  getSuggestedTabNameFromConnection,
  hasProjectPermissionV2,
  extractProjectResourceName,
} from "@/utils";

const { t } = useI18n();
const route = useRoute();

const me = useCurrentUserV1();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const sqlEditorStore = useSQLEditorV2Store();
const worksheetStore = useWorkSheetStore();
const { filter } = useFilterStore();

const initializeProjects = async () => {
  const projectInQuery = route.query.project as string;
  if (typeof projectInQuery === "string" && projectInQuery) {
    const project = `projects/${projectInQuery}`;
    sqlEditorStore.strictProject = true;
    sqlEditorStore.project = project;
    await projectStore.getOrFetchProjectByName(project, true /* silent */);
  } else {
    const projectList = await projectStore.fetchProjectList(false);
    const lastView = sqlEditorStore.storedLastViewedProject;
    if (
      lastView &&
      projectList.findIndex((proj) => proj.name === lastView) >= 0
    ) {
      sqlEditorStore.project = lastView;
    } else {
      const projectListWithoutDefaultProject = projectList.filter(
        (proj) => proj.name !== DEFAULT_PROJECT_V1_NAME
      );
      sqlEditorStore.project =
        head(projectListWithoutDefaultProject)?.name ??
        head(projectList)?.name ??
        "";
    }
    sqlEditorStore.strictProject = false;
  }
};

const switchProject = async () => {
  const { project } = sqlEditorStore;
  if (project) {
    await projectStore.getOrFetchProjectByName(project, true /* silent */);
  } else {
    await projectStore.fetchProjectList(false /* !showDeleted */);
  }
};

const prepareInstances = async () => {
  const instanceStore = useInstanceV1Store();
  const { project } = sqlEditorStore;
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
  const { project } = sqlEditorStore;
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

  sqlEditorStore.databaseList = databaseList;
};

onMounted(async () => {
  sqlEditorStore.projectContextReady = false;
  await initializeProjects();
  await prepareInstances();
  await prepareDatabases();
  sqlEditorStore.projectContextReady = true;

  watch(
    () => sqlEditorStore.project,
    async () => {
      sqlEditorStore.projectContextReady = false;
      await switchProject();
      await prepareInstances();
      await prepareDatabases();
      sqlEditorStore.projectContextReady = true;
    }
  );
});
</script>
