import { useLocalStorage } from "@vueuse/core";
import { includes } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref, watch } from "vue";
import { isValidProjectName, type ComposedDatabase } from "@/types";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export const useSQLEditorStore = defineStore("sqlEditor", () => {
  const resultRowsLimit = useLocalStorage(
    "bb.sql-editor.result-rows-limit",
    1000
  );
  const redisCommandOption = useLocalStorage<QueryOption_RedisRunCommandsOn>(
    "bb.sql-editor.redis-command-node",
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

  // empty to "ALL" projects for high-privileged users
  const project = ref<string>("");
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
    projectContextReady,
    storedLastViewedProject,
    allowViewALLProjects,
    isShowExecutingHint,
    executingHintDatabase,
    redisCommandOption,
  };
});
