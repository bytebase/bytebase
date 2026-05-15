import { useLocalStorage } from "@vueuse/core";
import { includes } from "lodash-es";
import { computed, reactive, ref, watchEffect } from "vue";
import { useProjectV1Store } from "@/store";
import { useQueryDataPolicy } from "@/store/modules/v1/policy";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import {
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  STORAGE_KEY_SQL_EDITOR_LAST_PROJECT,
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
} from "@/utils";

/**
 * Vue-reactive SQL Editor app state. Replaces the Pinia
 * `useSQLEditorStore` (`store/modules/sqlEditor/editor.ts`) — same
 * shape and field semantics, but lives as a module-level lazy
 * singleton instead of a Pinia store. Several fields are derived
 * computeds whose dependencies (project IAM policy, workspace
 * permissions) sit in other Pinia stores, so keeping the Vue
 * reactivity here lets consumers continue to use `useVueState(() =>
 * editorState.X)` without churning that pattern across ~40 files.
 *
 * When SheetTree / other consumers eventually move off `useVueState`,
 * this is the obvious thing to port to zustand (or selectors over the
 * zustand store).
 */
const buildSQLEditorVueState = () => {
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
        return storageValue;
      },
    }
  );

  // `false` while project-scoped resources are loading — consumers
  // render a skeleton layout with spinner placeholders.
  const projectContextReady = ref<boolean>(false);

  const storedLastViewedProject = useLocalStorage<string>(
    STORAGE_KEY_SQL_EDITOR_LAST_PROJECT,
    "",
    { listenToStorageChanges: false }
  );

  const { policy: queryDataPolicy } = useQueryDataPolicy(
    storedLastViewedProject
  );

  watchEffect(() => {
    if (resultRowsLimit.value > queryDataPolicy.value.maximumResultRows) {
      resultRowsLimit.value = queryDataPolicy.value.maximumResultRows;
    }
  });

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

  // Wrap with `reactive()` so consumers see the same auto-unwrap /
  // auto-wrap-on-assign ergonomics they had with the Pinia store. Use
  // Vue's `toRefs()` (not Pinia's `storeToRefs`) to extract individual
  // refs when needed.
  return reactive({
    resultRowsLimit,
    queryDataPolicy,
    project: computed(() => storedLastViewedProject.value),
    setProject,
    projectContextReady,
    storedLastViewedProject,
    allowViewALLProjects,
    isShowExecutingHint,
    executingHintDatabase,
    redisCommandOption,
    allowAdmin,
  });
};

let _state: ReturnType<typeof buildSQLEditorVueState> | undefined;
export const useSQLEditorVueState = () => {
  if (!_state) _state = buildSQLEditorVueState();
  return _state;
};

export type SQLEditorVueState = ReturnType<typeof useSQLEditorVueState>;
