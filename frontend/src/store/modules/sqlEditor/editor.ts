import { useLocalStorage } from "@vueuse/core";
import { includes } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { useProjectV1Store } from "@/store";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  STORAGE_KEY_SQL_EDITOR_LAST_PROJECT,
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
} from "@/utils";

export const useSQLEditorStore = defineStore("sqlEditor", () => {
  const projectStore = useProjectV1Store();

  const resultRowsLimit = useLocalStorage(
    STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
    1000
  );
  const redisCommandOption = useLocalStorage<QueryOption_RedisRunCommandsOn>(
    STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
    QueryOption_RedisRunCommandsOn.SINGLE_NODE,
    {
      // Use a custom merge function to ensure the value is valid.
      mergeDefaults(storageValue, defaults) {
        if (
          !includes(
            [
              QueryOption_RedisRunCommandsOn.SINGLE_NODE,
              QueryOption_RedisRunCommandsOn.ALL_NODES,
            ],
            storageValue
          )
        ) {
          return defaults;
        }
        // Otherwise, return the storage value
        return storageValue;
      },
    }
  );

  // `false` if we are preparing project-scoped resources
  // we should render a skeleton layout with spinner placeholders
  const projectContextReady = ref<boolean>(false);

  const storedLastViewedProject = useLocalStorage<string>(
    STORAGE_KEY_SQL_EDITOR_LAST_PROJECT,
    "",
    { listenToStorageChanges: false }
  );

  const allowViewALLProjects = computed(() => {
    return hasWorkspacePermissionV2("bb.projects.list");
  });

  const setProject = (project: string) => {
    storedLastViewedProject.value = project;
    projectContextReady.value = true;
  };

  const isShowExecutingHint = ref(false);
  const executingHintDatabase = ref<Database | undefined>();

  const allowAdmin = computed(() => {
    const project = projectStore.getProjectByName(
      storedLastViewedProject.value
    );
    return hasProjectPermissionV2(project, "bb.sql.admin");
  });

  return {
    resultRowsLimit,
    project: computed(() => storedLastViewedProject.value),
    setProject,
    projectContextReady,
    storedLastViewedProject,
    allowViewALLProjects,
    isShowExecutingHint,
    executingHintDatabase,
    redisCommandOption,
    allowAdmin,
  };
});
