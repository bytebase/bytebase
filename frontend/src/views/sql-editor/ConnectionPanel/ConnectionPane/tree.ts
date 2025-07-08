import { useDebounceFn } from "@vueuse/core";
import { NCheckbox } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, ref, watch, h } from "vue";
import type { ComputedRef, Ref } from "vue";
import { t } from "@/plugins/i18n";
import {
  useSQLEditorStore,
  useCurrentUserV1,
  useDatabaseV1Store,
  type DatabaseFilter,
  buildTreeImpl,
} from "@/store";
import type {
  ComposedDatabase,
  SQLEditorTreeNode as TreeNode,
  StatefulSQLEditorTreeFactor as StatefulFactor,
} from "@/types";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  isDatabaseV1Queryable,
  getDefaultPagination,
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
  buildTree: () => void;
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

  const databaseList = ref<ComposedDatabase[]>([]);
  const fetchDataState = ref<{
    loading: boolean;
    nextPageToken?: string;
  }>({ loading: false });

  const expandedState = useDynamicLocalStorage<{
    initialized: boolean;
    expandedKeys: string[];
  }>(
    computed(
      () =>
        `bb.sql-editor.connection-pane.expanded_${environment}.${currentUser.value.name}`
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

  const hasMissingQueryDatabases = computed(() => {
    return databaseList.value.some((db) => !isDatabaseV1Queryable(db));
  });

  const filteredDatabaseList = computed(() => {
    if (!showMissingQueryDatabases.value) {
      return databaseList.value.filter((db) => isDatabaseV1Queryable(db));
    }
    return databaseList.value;
  });

  const tree = ref<TreeNode[]>([]);
  const showMissingQueryDatabases = ref<boolean>(false);

  const buildTree = () => {
    tree.value = buildTreeImpl(filteredDatabaseList.value, [
      defaultEnvironmentFactor.factor,
    ]);

    if (hasMissingQueryDatabases.value) {
      for (const node of tree.value) {
        node.suffix = () => {
          return h(
            NCheckbox,
            {
              checked: showMissingQueryDatabases.value,
              onUpdateChecked: (checked) =>
                (showMissingQueryDatabases.value = checked),
              onClick: (e: MouseEvent) => {
                e.preventDefault();
                e.stopPropagation();
              },
            },
            h(
              "span",
              { class: "textinfolabel text-sm" },
              t("sql-editor.show-databases-without-query-permission")
            )
          );
        };
      }
    }
  };

  watch(
    () => showMissingQueryDatabases.value,
    () => {
      buildTree();
    }
  );

  return {
    tree: computed(() => tree.value),
    expandedState,
    buildTree,
    prepareDatabases,
    fetchDatabases,
    fetchDataState: computed(() => fetchDataState.value),
  };
};
