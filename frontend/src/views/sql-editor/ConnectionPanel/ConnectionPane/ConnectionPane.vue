<template>
  <div class="sql-editor-tree h-full relative">
    <div class="w-full px-4 mt-4" v-if="tabStore.supportBatchMode">
      <div
        class="textinfolabel mb-2 w-full leading-4 flex flex-col items-start gap-x-1"
      >
        <div class="flex items-center gap-x-1">
          <FeatureBadge :feature="PlanFeature.FEATURE_BATCH_QUERY" />
          {{
            $t("sql-editor.batch-query.description", {
              database: selectedDatabaseNames.length,
              group:
                tabStore.currentTab?.batchQueryContext.databaseGroups
                  ?.length ?? 0,
              project: project.title,
            })
          }}
        </div>
      </div>
      <div
        class="w-full mt-1 flex flex-row justify-start items-start flex-wrap gap-2"
      >
        <NTag
          v-for="database in selectedDatabaseNames"
          :key="database"
          :closable="true"
          :disabled="state.switchingConnection"
          @close="() => handleToggleDatabase(database, false)"
        >
          <RichDatabaseName
            :database="databaseStore.getDatabaseByName(database)"
          />
        </NTag>
        <template v-if="hasDatabaseGroupFeature">
          <DatabaseGroupTag
            v-for="databaseGroupName in tabStore.currentTab?.batchQueryContext
              ?.databaseGroups ?? []"
            :key="databaseGroupName"
            :disabled="state.switchingConnection"
            :database-group-name="databaseGroupName"
            @uncheck="handleUncheckDatabaseGroup"
          />
        </template>
        <NDivider v-if="tabStore.isInBatchMode" class="my-2!" />
        <div v-if="tabStore.isInBatchMode" class="w-full">
          <div class="textinfolabel flex items-center gap-x-1">
            {{ $t("sql-editor.batch-query.select-data-source.self") }}
            <NTooltip>
              <template #trigger>
                <InfoIcon class="w-4" />
              </template>
              {{ $t("sql-editor.batch-query.select-data-source.tooltip") }}
            </NTooltip>
          </div>
          <NRadioGroup v-model:value="state.batchQueryDataSourceType">
            <NRadio :value="DataSourceType.ADMIN">
              {{ getDataSourceTypeI18n(DataSourceType.ADMIN) }}
            </NRadio>
            <NRadio :value="DataSourceType.READ_ONLY">
              {{ getDataSourceTypeI18n(DataSourceType.READ_ONLY) }}
            </NRadio>
          </NRadioGroup>
        </div>
      </div>
    </div>
    <NDivider v-if="tabStore.currentTab" class="my-3!" />
    <NTabs
      v-model:value="state.selectionMode"
      type="line"
      animated
      size="small"
      class="px-4"
    >
      <NTabPane name="DATABASE" :tab="$t('common.databases')">
        <div class="flex flex-col gap-y-1">
          <AdvancedSearch
            v-model:params="state.params"
            :autofocus="false"
            :placeholder="$t('database.filter-database')"
            :scope-options="scopeOptions"
            :cache-query="false"
          />
          <div
            ref="treeContainerElRef"
            class="relative sql-editor-tree--tree flex-1 pb-1 text-sm select-none"
            :data-height="treeContainerHeight"
          >
            <template v-if="treeStore.state === 'READY'">
              <NCheckbox
                v-if="existEmptyEnvironment"
                v-model:checked="state.showEmptyEnvironment"
              >
                {{ $t("sql-editor.show-empty-environments") }}
              </NCheckbox>
              <div
                v-for="[environment, treeState] of treeByEnvironment.entries()"
                :key="environment"
              >
                <div
                  v-if="
                    !treeIsEmpty(treeState) ||
                    (state.showEmptyEnvironment &&
                      environment !== UNKNOWN_ENVIRONMENT_NAME)
                  "
                  class="flex flex-col gap-y-2 pt-2 pb-2"
                >
                  <NTree
                    :block-line="true"
                    :data="treeState.tree.value"
                    :show-irrelevant-nodes="false"
                    :selected-keys="selectedKeys"
                    :expand-on-click="true"
                    v-model:expanded-keys="treeState.expandedState.value.expandedKeys"
                    :node-props="nodeProps"
                    :theme-overrides="{ nodeHeight: '21px' }"
                    :render-label="renderLabel"
                  />
                  <div
                    v-if="
                      !!treeState.fetchDataState.value.nextPageToken &&
                      treeState.expandedState.value.expandedKeys.includes(
                        environment
                      ) &&
                      (!treeIsEmpty(treeState) ||
                        treeState.showMissingQueryDatabases.value)
                    "
                    class="w-full flex items-center justify-start pl-4"
                  >
                    <NButton
                      quaternary
                      :size="'small'"
                      :loading="treeState.fetchDataState.value.loading"
                      @click="
                        () =>
                          treeState
                            .fetchDatabases(filter)
                            .then(() => treeState.buildTree())
                      "
                    >
                      {{ $t("common.load-more") }}
                    </NButton>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>
      </NTabPane>
      <NTabPane name="DATABASE-GROUP" :tab="$t('common.database-group')" :disabled="!hasDatabaseGroupFeature || !hasBatchQueryFeature">
        <template #tab>
          <NTooltip :disabled="hasBatchQueryFeature && hasDatabaseGroupFeature">
            <template #trigger>
              {{ $t('common.database-group') }}
            </template>
            {{ $t("subscription.contact-to-upgrade") }}
          </NTooltip>
        </template>
        <DatabaseGroupTable
          :database-group-names="selectedDatabaseGroupNames"
          @update:database-group-names="onDatabaseGroupSelectionUpdate"
        />
      </NTabPane>
    </NTabs>

    <NDropdown
      v-if="treeStore.state === 'READY'"
      placement="bottom-start"
      trigger="manual"
      :x="dropdownPosition.x"
      :y="dropdownPosition.y"
      :options="dropdownOptions"
      :show="showDropdown"
      :on-clickoutside="handleDropdownClickoutside"
      @select="handleDropdownSelect"
    />

    <DatabaseHoverPanel :offset-x="10" :offset-y="4" :margin="4" />

    <MaskSpinner v-if="treeStore.state !== 'READY'" class="bg-white/75!">
      <span class="text-control text-sm">{{
        $t("sql-editor.loading-databases")
      }}</span>
    </MaskSpinner>
  </div>

  <FeatureModal
    v-if="state.missingFeature"
    :feature="state.missingFeature"
    :open="!!state.missingFeature"
    @cancel="state.missingFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { InfoIcon } from "lucide-vue-next";
import {
  NButton,
  NCheckbox,
  NDivider,
  NDropdown,
  NRadio,
  NRadioGroup,
  NTabPane,
  NTabs,
  NTag,
  NTooltip,
  NTree,
  type TreeOption,
} from "naive-ui";
import { computed, h, nextTick, reactive, ref, shallowRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { RichDatabaseName } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  type DatabaseFilter,
  featureToRef,
  idForSQLEditorTreeNodeTarget,
  resolveOpeningDatabaseListFromSQLEditorTabList,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1List,
  useInstanceResourceByName,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLEditorTreeStore,
} from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import type {
  BatchQueryContext,
  ComposedDatabase,
  QueryDataSourceType,
  SQLEditorTreeNode,
} from "@/types";
import {
  getDataSourceTypeI18n,
  isValidDatabaseGroupName,
  isValidDatabaseName,
  isValidProjectName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SearchParams } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  findAncestor,
  getConnectionForSQLEditorTab,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  isDatabaseV1Queryable,
  isDescendantOf,
  useDynamicLocalStorage,
} from "@/utils";
import { useSQLEditorContext } from "../../context";
import { setConnection, useDropdown } from "./actions";
import DatabaseGroupTable from "./DatabaseGroupTable.vue";
import DatabaseGroupTag from "./DatabaseGroupTag.vue";
import {
  DatabaseHoverPanel,
  provideHoverStateContext,
} from "./DatabaseHoverPanel";
import { Label } from "./TreeNode";
import { type TreeByEnvironment, useSQLEditorTreeByEnvironment } from "./tree";

interface LocalState {
  params: SearchParams;
  missingFeature?: PlanFeature;
  showEmptyEnvironment: boolean;
  batchQueryDataSourceType?: QueryDataSourceType;
  selectionMode: "DATABASE" | "DATABASE-GROUP";
  switchingConnection: boolean;
}

const props = defineProps<{
  show: boolean;
}>();

const treeStore = useSQLEditorTreeStore();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const projectStore = useProjectV1Store();
const environmentList = useEnvironmentV1List();
const currentUser = useCurrentUserV1();
const { t } = useI18n();

const expandedState = useDynamicLocalStorage<{
  initialized: boolean;
  expandedKeys: string[];
}>(
  computed(
    () =>
      `bb.sql-editor.connection-pane.expanded-keys.${currentUser.value.name}`
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

const hasBatchQueryFeature = featureToRef(PlanFeature.FEATURE_BATCH_QUERY);
const hasDatabaseGroupFeature = featureToRef(
  PlanFeature.FEATURE_DATABASE_GROUPS
);

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showEmptyEnvironment: false,
  batchQueryDataSourceType: DataSourceType.READ_ONLY,
  selectionMode: "DATABASE",
  switchingConnection: false,
});

const selectedDatabaseGroupNames = computed(
  () => tabStore.currentTab?.batchQueryContext.databaseGroups ?? []
);

const selectedDatabaseNames = computed(() => {
  const currentTab = tabStore.currentTab;
  const databases = currentTab?.batchQueryContext.databases ?? [];
  if (databases.length === 0 && selectedDatabaseGroupNames.value.length === 0) {
    if (currentTab?.connection.database) {
      databases.push(currentTab?.connection.database);
    }
  }
  return databases;
});

watch(
  () => props.show,
  (show) => {
    if (show) {
      if (selectedDatabaseNames.value.length > 0) {
        state.selectionMode = "DATABASE";
      } else if (selectedDatabaseGroupNames.value.length > 0) {
        state.selectionMode = "DATABASE-GROUP";
      } else {
        state.selectionMode = "DATABASE";
      }
    }
  },
  { immediate: true }
);

const onBatchQueryContextChange = async (
  batchQueryContext: BatchQueryContext
) => {
  state.switchingConnection = true;

  try {
    const queryableDatabase = await getQueryableDatabase(batchQueryContext);
    const currentConnection = getConnectionForSQLEditorTab(tabStore.currentTab);

    if (
      !currentConnection.database?.name ||
      currentConnection.database?.name !== queryableDatabase?.name
    ) {
      // switch connection when:
      // - no connection
      //   - no current tab
      //   - current tab doesn't have valid connection
      // - or connection changed
      setConnection({
        database: queryableDatabase,
        mode: "WORKSHEET",
        newTab: false,
        context: editorContext,
        batchQueryContext,
      });
    } else {
      tabStore.updateBatchQueryContext(batchQueryContext);
    }
  } finally {
    state.switchingConnection = false;
  }
};

const getQueryableDatabase = async (batchQueryContext: BatchQueryContext) => {
  for (const databaseName of batchQueryContext.databases) {
    const database = databaseStore.getDatabaseByName(databaseName);
    if (isDatabaseV1Queryable(database)) {
      return database;
    }
  }

  for (const databaseGroupName of batchQueryContext.databaseGroups ?? []) {
    const databaseGroup = dbGroupStore.getDBGroupByName(databaseGroupName);
    await databaseStore.batchGetOrFetchDatabases(
      databaseGroup.matchedDatabases.map((db) => db.name)
    );
    for (const matchedDatabase of databaseGroup.matchedDatabases) {
      const database = databaseStore.getDatabaseByName(matchedDatabase.name);
      if (isDatabaseV1Queryable(database)) {
        return database;
      }
    }
  }
};

const onDatabaseSelectionUpdate = async (databases: string[]) => {
  const batchQueryContext: BatchQueryContext = cloneDeep({
    ...tabStore.currentTab?.batchQueryContext,
    databases,
  });
  await onBatchQueryContextChange(batchQueryContext);
};

const onDatabaseGroupSelectionUpdate = async (databaseGroups: string[]) => {
  if (databaseGroups.length > 0) {
    if (!hasBatchQueryFeature) {
      state.missingFeature = PlanFeature.FEATURE_BATCH_QUERY;
      return;
    }
    if (!hasDatabaseGroupFeature) {
      state.missingFeature = PlanFeature.FEATURE_DATABASE_GROUPS;
      return;
    }
  }

  const batchQueryContext: BatchQueryContext = cloneDeep({
    ...(tabStore.currentTab?.batchQueryContext ?? { databases: [] }),
    databaseGroups,
  });
  await onBatchQueryContextChange(batchQueryContext);
};

const flattenSelectedDatabasesFromGroup = computed(() => {
  const nameMap = new Map<
    string /* database name */,
    string /* group title */
  >();
  for (const groupName of tabStore.currentTab?.batchQueryContext
    ?.databaseGroups ?? []) {
    const group = dbGroupStore.getDBGroupByName(groupName);
    if (!isValidDatabaseGroupName(group.name)) {
      continue;
    }
    for (const db of group.matchedDatabases) {
      nameMap.set(db.name, group.title);
    }
  }
  return nameMap;
});

const treeByEnvironment = shallowRef<
  Map<string /* environment full name */, TreeByEnvironment>
>(new Map());

const existEmptyEnvironment = computed(() => {
  for (const [_, value] of treeByEnvironment.value.entries()) {
    if (treeIsEmpty(value)) {
      return true;
    }
  }
  return false;
});

const treeIsEmpty = (value: TreeByEnvironment) => {
  if (value.tree.value.length === 0) {
    return true;
  }
  const children = value.tree.value[0].children;
  if (!children || children.length === 0) {
    return true;
  }
  return false;
};

watch(
  () => state.batchQueryDataSourceType,
  (dataSourceType) => {
    tabStore.updateBatchQueryContext({
      dataSourceType,
    });
  },
  { immediate: true }
);

watch(
  () => tabStore.currentTab?.id,
  async () => {
    if (!tabStore.currentTab) {
      return;
    }
    const databases = tabStore.currentTab.batchQueryContext.databases ?? [];
    await databaseStore.batchGetOrFetchDatabases(databases);
  },
  {
    immediate: true,
  }
);

const handleToggleDatabase = async (database: string, check: boolean) => {
  let databases: string[] = [...selectedDatabaseNames.value];
  if (check) {
    if (!databases.includes(database)) {
      databases.push(database);
    }
    if (databases.length > 1 && !hasBatchQueryFeature.value) {
      state.missingFeature = PlanFeature.FEATURE_BATCH_QUERY;
      return;
    }
  } else {
    databases = selectedDatabaseNames.value.filter((db) => db !== database);
  }
  await onDatabaseSelectionUpdate(databases);
};

const handleUncheckDatabaseGroup = async (databaseGroupName: string) => {
  const databaseGroups = selectedDatabaseGroupNames.value.filter(
    (name) => name !== databaseGroupName
  );
  await onDatabaseGroupSelectionUpdate(databaseGroups);
};

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList.filter((scope) => scope !== "environment"),
  "label",
  "engine",
]);

const project = computed(() =>
  projectStore.getProjectByName(editorStore.project)
);

const editorContext = useSQLEditorContext();
const { events: editorEvents, showConnectionPanel } = editorContext;
const {
  state: hoverState,
  position: hoverPosition,
  update: updateHoverNode,
} = provideHoverStateContext();
const {
  show: showDropdown,
  context: dropdownContext,
  position: dropdownPosition,
  options: dropdownOptions,
  handleSelect: handleDropdownSelect,
  handleClickoutside: handleDropdownClickoutside,
} = useDropdown();

const treeContainerElRef = ref<HTMLElement>();
const { height: treeContainerHeight } = useElementSize(
  treeContainerElRef,
  undefined,
  {
    box: "content-box",
  }
);
const selectedKeys = ref<string[]>([]);

// Highlight the current tab's connection node.
const getSelectedKeys = async () => {
  const connection = tabStore.currentTab?.connection;
  if (!connection) {
    return [];
  }

  if (connection.database) {
    const database = await databaseStore.getOrFetchDatabaseByName(
      connection.database
    );
    return treeStore.nodeKeysByTarget("database", database);
  } else if (connection.instance) {
    const { instance } = useInstanceResourceByName(connection.instance);
    return treeStore.nodeKeysByTarget("instance", instance.value);
  }
  return [];
};

const connectedDatabases = computed(() =>
  resolveOpeningDatabaseListFromSQLEditorTabList()
);

// The connection panel should:
// - change the connection for the current tab
// - connect to a new tab, this will create a new worksheet
const connect = (node: SQLEditorTreeNode) => {
  if (node.disabled || node.meta.type !== "database") {
    return;
  }
  const database = node.meta.target as ComposedDatabase;
  if (!isDatabaseV1Queryable(database)) {
    return;
  }

  const batchQueryDatabases = [
    ...selectedDatabaseGroupNames.value,
    database.name,
  ].filter(isValidDatabaseName);
  setConnection({
    database,
    mode: tabStore.currentTab?.mode,
    newTab: false,
    context: editorContext,
    batchQueryContext: {
      databases: [...new Set(batchQueryDatabases)],
    },
  });
  showConnectionPanel.value = false;
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as SQLEditorTreeNode;
  let databaseName = "";
  if (node.meta.type === "database") {
    databaseName = (node as SQLEditorTreeNode<"database">).meta.target.name;
  }

  const checkedByGroup =
    flattenSelectedDatabasesFromGroup.value.get(databaseName);
  let checkTooltip = "";
  if (!!checkedByGroup) {
    checkTooltip = t("sql-editor.matched-in-group", { title: checkedByGroup });
  } else if (tabStore.currentTab?.connection.database === databaseName) {
    checkTooltip = t("sql-editor.current-connection");
  }

  return h(Label, {
    node,
    checked:
      selectedDatabaseNames.value.includes(databaseName) || !!checkedByGroup,
    checkDisabled: !!checkedByGroup || state.switchingConnection,
    checkTooltip,
    keyword: state.params.query,
    connected: connectedDatabases.value.has(databaseName),
    "onUpdate:checked": (checked: boolean) => {
      if (node.meta.type !== "database") {
        return;
      }
      if (checked) {
        handleToggleDatabase(databaseName, true);
      } else {
        handleToggleDatabase(databaseName, false);
      }
    },
  });
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as SQLEditorTreeNode;
  return {
    onClick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        const { type } = node.meta;
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        if (type === "database") {
          connect(node);
        }
      }
    },
    onContextmenu(e: MouseEvent) {
      e.preventDefault();
      showDropdown.value = false;
      if (node && node.key) {
        dropdownContext.value = node;
      }

      nextTick().then(() => {
        showDropdown.value = true;
        dropdownPosition.value.x = e.clientX;
        dropdownPosition.value.y = e.clientY;
      });
    },
    onmouseenter(e: MouseEvent) {
      if (node.meta.type === "database") {
        if (hoverState.value) {
          updateHoverNode({ node }, "before", 0 /* overrideDelay */);
        } else {
          updateHoverNode({ node }, "before");
        }
        nextTick().then(() => {
          // Find the node element and put the database panel to the right corner
          // of the node
          const wrapper = findAncestor(e.target as HTMLElement, ".n-tree-node");
          if (!wrapper) {
            updateHoverNode(undefined, "after", 0 /* overrideDelay */);
            return;
          }
          const bounding = wrapper.getBoundingClientRect();
          hoverPosition.value.x = bounding.left;
          hoverPosition.value.y = bounding.bottom;
        });
      }
    },
    onmouseleave() {
      updateHoverNode(undefined, "after");
    },
    // attrs below for trouble-shooting
    "data-node-meta-type": node.meta.type,
    "data-node-meta-id": idForSQLEditorTreeNodeTarget(
      node.meta.type,
      node.meta.target
    ),
    "data-node-key": node.key,
  };
};

useEmitteryEventListener(editorEvents, "tree-ready", async () => {
  selectedKeys.value = await getSelectedKeys();

  for (const [environment, treeState] of treeByEnvironment.value.entries()) {
    if (!treeState.expandedState.value.initialized) {
      // default expand all nodes.
      treeState.expandedState.value.expandedKeys = treeStore.allNodeKeys.filter(
        (key) => key.startsWith(environment)
      );
    }
  }
});

const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const selectedInstance = computed(() => {
  return getValueFromSearchParams(state.params, "instance", instanceNamePrefix);
});

const selectedEngines = computed(() => {
  return getValuesFromSearchParams(state.params, "engine").map(
    (engine) => Engine[engine as keyof typeof Engine]
  );
});

const filter = computed(
  (): DatabaseFilter => ({
    instance: selectedInstance.value,
    query: state.params.query,
    labels: selectedLabels.value,
    engines: selectedEngines.value,
  })
);

watch(
  () => editorStore.project,
  (project) => {
    state.params.scopes = [
      {
        id: "project",
        readonly: true,
        value: extractProjectResourceName(project),
      },
    ];
  },
  { immediate: true }
);

const prepareDatabases = async () => {
  treeByEnvironment.value.clear();
  await Promise.all(
    [...environmentList.value, unknownEnvironment()].map(
      async (environment) => {
        if (!treeByEnvironment.value.has(environment.name)) {
          treeByEnvironment.value.set(
            environment.name,
            useSQLEditorTreeByEnvironment(environment.name)
          );
        }
        await treeByEnvironment.value
          .get(environment.name)
          ?.prepareDatabases(filter.value);
        treeByEnvironment.value.get(environment.name)?.buildTree();
      }
    )
  );
};

watch(
  [
    () => editorStore.project,
    () => editorStore.projectContextReady,
    () => filter.value,
  ],
  async ([project, ready, _]) => {
    if (!isValidProjectName(project)) {
      return;
    }
    if (!ready) {
      treeStore.state = "LOADING";
    } else {
      await prepareDatabases();
      treeStore.state = "READY";
      editorEvents.emit("tree-ready");
    }
  },
  { immediate: true, deep: true }
);
</script>

<style lang="postcss" scoped>
.sql-editor-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.sql-editor-tree :deep(.n-tree-node-content) {
  padding-left: 0 !important;
  font-size: 0.875rem;
  line-height: 1.25rem;
}
.sql-editor-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.sql-editor-tree :deep(.n-tree-node-indent) {
  width: 1rem;
}
.sql-editor-tree :deep(.n-tree-node-switcher--hide) {
  width: 0.5rem !important;
}
.sql-editor-tree :deep(.n-tree-node-content__prefix) {
  flex-shrink: 0;
  margin-right: 0.25rem !important;
}
.sql-editor-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  display: none;
}
.sql-editor-tree :deep(.n-tree-node-content__text) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 0.25rem;
}
.sql-editor-tree :deep(.n-tree-node--pending) {
  background-color: transparent !important;
}
.sql-editor-tree :deep(.n-tree-node--pending:hover) {
  background-color: var(--n-node-color-hover) !important;
}
.sql-editor-tree :deep(.n-tree-node--selected),
.sql-editor-tree :deep(.n-tree-node--selected:hover) {
  background-color: var(--n-node-color-active) !important;
  font-weight: 500;
}
</style>
