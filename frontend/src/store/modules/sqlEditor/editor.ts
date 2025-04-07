import { useLocalStorage, useDebounceFn } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref, watch, watchEffect } from "vue";
import { useDatabaseV1Store } from "@/store";
import { type ComposedDatabase, isValidProjectName } from "@/types";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto/v1/sql_service";
import { hasWorkspacePermissionV2, getDefaultPagination } from "@/utils";

export const useSQLEditorStore = defineStore("sqlEditor", () => {
  const databaseStore = useDatabaseV1Store();

  const resultRowsLimit = useLocalStorage(
    "bb.sql-editor.result-rows-limit",
    1000
  );
  const redisCommandOption = useLocalStorage<QueryOption_RedisRunCommandsOn>(
    "bb.sql-editor.redis-command-node",
    QueryOption_RedisRunCommandsOn.SINGLE_NODE
  );

  // empty to "ALL" projects for high-privileged users
  const project = ref<string>("");
  // if `true`, won't show project selector and not allowed to switch to other projects
  const strictProject = ref<boolean>(false);
  // `false` if we are preparing project-scoped resources
  // we should render a skeleton layout with spinner placeholders
  const projectContextReady = ref<boolean>(false);
  const storedLastViewedProject = useLocalStorage<string>(
    "bb.sql-editor.last-viewed-project",
    ""
  );

  const allowViewALLProjects = computed(() => {
    return hasWorkspacePermissionV2("bb.projects.list");
  });

  // `databaseList` is query-able databases scoped by `project`
  const databaseList = ref<ComposedDatabase[]>([]);
  const fetchDataState = ref<{
    nextPageToken?: string;
    loading: boolean;
  }>({
    loading: false,
  });

  const fetchDatabases = useDebounceFn(async (name?: string) => {
    fetchDataState.value.loading = true;
    const pageToken = fetchDataState.value.nextPageToken;

    try {
      const { databases, nextPageToken } = await databaseStore.fetchDatabases({
        parent: project.value,
        pageToken,
        pageSize: getDefaultPagination(),
        filter: {
          query: name,
        },
      });

      if (pageToken) {
        databaseList.value.push(...databases);
      } else {
        databaseList.value = [...databases];
      }
      fetchDataState.value.nextPageToken = nextPageToken;
    } catch {
      databaseList.value = [];
    } finally {
      fetchDataState.value.loading = false;
    }
  }, 500);

  const prepareDatabases = async (name?: string) => {
    fetchDataState.value.nextPageToken = "";
    await fetchDatabases(name);
  };

  watchEffect(async () => {
    if (isValidProjectName(project.value)) {
      await prepareDatabases();
    }
  });

  watch(project, (project) => {
    if (isValidProjectName(project)) {
      storedLastViewedProject.value = project;
    }
  });

  const isShowExecutingHint = ref(false);
  const executingHintDatabase = ref<ComposedDatabase | undefined>();

  return {
    resultRowsLimit,
    project,
    strictProject,
    projectContextReady,
    storedLastViewedProject,
    allowViewALLProjects,
    databaseList,
    isShowExecutingHint,
    executingHintDatabase,
    redisCommandOption,
    prepareDatabases,
    fetchDatabases,
    loading: computed(() => fetchDataState.value.loading),
  };
});
