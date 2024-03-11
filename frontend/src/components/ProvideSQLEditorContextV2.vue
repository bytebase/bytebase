<template>
  <teleport to="#sql-editor-debug">
    <li>[ProvideContext]project: {{ sqlEditorStore.project }}</li>
    <li>[ProvideContext]strictProject: {{ sqlEditorStore.strictProject }}</li>
    <li>
      [ProvideContext]projectContextReady:
      {{ sqlEditorStore.projectContextReady }}
    </li>
    <li>
      [ProvideContext]databaseCount: {{ sqlEditorStore.databaseList.length }}
    </li>
    <li>
      [ProvideContext]allowViewALLProjects:
      {{ sqlEditorStore.allowViewALLProjects }}
    </li>
  </teleport>
  <slot />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { onMounted, watch } from "vue";
import { useRoute } from "vue-router";
import {
  useInstanceV1Store,
  useProjectV1Store,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorV2Store,
  useSQLEditorTabStore,
} from "@/store";
import { DEFAULT_PROJECT_V1_NAME, UNKNOWN_USER_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { extractProjectResourceName } from "@/utils";

const route = useRoute();

const me = useCurrentUserV1();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const sqlEditorStore = useSQLEditorV2Store();
const tabStore = useSQLEditorTabStore();

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

  tabStore.maybeInitProject(sqlEditorStore.project);
};

const switchProject = async () => {
  const { project } = sqlEditorStore;
  if (project) {
    await projectStore.getOrFetchProjectByName(project, true /* silent */);
  } else {
    await projectStore.fetchProjectList(false /* !showDeleted */);
  }
  tabStore.maybeInitProject(project);
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
