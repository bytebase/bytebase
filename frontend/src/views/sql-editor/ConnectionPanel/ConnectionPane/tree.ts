import { useDebounceFn } from "@vueuse/core";
import { storeToRefs } from "pinia";
import type { ComputedRef, Ref } from "vue";
import { computed, ref, watch } from "vue";
import {
  buildTreeImpl,
  type DatabaseFilter,
  mapTreeNodeByType,
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useSQLEditorStore,
} from "@/store";
import type {
  StatefulSQLEditorTreeFactor as StatefulFactor,
  SQLEditorTreeNode as TreeNode,
} from "@/types";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  getDefaultPagination,
  isDatabaseV1Queryable,
  storageKeySqlEditorConnExpanded,
  useDynamicLocalStorage,
} from "@/utils";

const defaultEnvironmentFactor: StatefulFactor = {
  factor: "environment",
  disabled: false,
};

export type TreeByEnvironment = {
  tree: ComputedRef<TreeNode[]>;
  expandedState: Ref<{
    initialized: boolean;
    expandedKeys: string[];
  }>;
  buildTree: (showMissingQueryDatabases: boolean) => void;
  prepareDatabases: (filter?: DatabaseFilter) => Promise<void>;
  fetchDatabases: (filter?: DatabaseFilter) => Promise<void>;
  fetchDataState: ComputedRef<{
    loading: boolean;
    nextPageToken?: string;
  }>;
};

export const useSQLEditorTreeByEnvironment = (
  environment: string
): TreeByEnvironment => {
  const databaseStore = useDatabaseV1Store();
  const { project } = storeToRefs(useSQLEditorStore());
  const currentUser = useCurrentUserV1();
  const environmentStore = useEnvironmentV1Store();

  const tree = ref<TreeNode[]>([]);
  const databaseList = ref<Database[]>([]);
  const fetchDataState = ref<{
    loading: boolean;
    nextPageToken?: string;
  }>({ loading: false });

  const expandedState = useDynamicLocalStorage<{
    initialized: boolean;
    expandedKeys: string[];
  }>(
    computed(() =>
      storageKeySqlEditorConnExpanded(environment, currentUser.value.email)
    ),
    {
      initialized: false,
      expandedKeys: [],
    }
  );

  watch(
    () => expandedState.value.expandedKeys,
    () => {
      expandedState.value.initialized = true;
    },
    { deep: true }
  );

  const fetchDatabases = useDebounceFn(async (filter?: DatabaseFilter) => {
    fetchDataState.value.loading = true;
    const pageToken = fetchDataState.value.nextPageToken;

    try {
      const { databases, nextPageToken } = await databaseStore.fetchDatabases({
        parent: project.value,
        pageToken,
        pageSize: getDefaultPagination(),
        filter: {
          ...filter,
          environment,
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
  }, DEBOUNCE_SEARCH_DELAY);

  const prepareDatabases = async (filter?: DatabaseFilter) => {
    fetchDataState.value.nextPageToken = "";
    await fetchDatabases(filter);
  };

  const buildTree = (showMissingQueryDatabases: boolean) => {
    let list = [...databaseList.value];
    if (!showMissingQueryDatabases) {
      list = list.filter((db) => isDatabaseV1Queryable(db));
    }
    tree.value = buildTreeImpl(list, [defaultEnvironmentFactor.factor]);

    if (tree.value.length === 0) {
      const env = environmentStore.getEnvironmentByName(environment);
      const rootNode = mapTreeNodeByType("environment", env, undefined);
      rootNode.children = [];
      tree.value = [rootNode];
    }
  };

  return {
    tree: computed(() => tree.value),
    expandedState,
    buildTree,
    prepareDatabases,
    fetchDatabases,
    fetchDataState: computed(() => fetchDataState.value),
  };
};
