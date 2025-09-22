<template>
  <div class="sql-editor-tree gap-y-1 h-full flex flex-col relative">
    <div class="w-full px-4 mt-4" v-if="tabStore.supportBatchMode">
      <div
        class="textinfolabel mb-2 w-full leading-4 flex flex-col items-start gap-x-1"
      >
        <div class="flex items-center gap-x-1">
          <FeatureBadge :feature="PlanFeature.FEATURE_BATCH_QUERY" />
          {{
            $t("sql-editor.batch-query.description", {
              database: state.selectedDatabases.size,
              group:
                tabStore.currentTab?.batchQueryContext?.databaseGroups
                  ?.length ?? 0,
              project: project.title,
            })
          }}
        </div>
        <div class="flex items-center gap-x-1">
          <i18n-t
            v-if="hasDatabaseGroupFeature && tabStore.currentTab"
            keypath="sql-editor.batch-query.select-database-group"
          >
            <template #select-database-group>
              <BatchQueryDatabaseGroupSelector />
            </template>
          </i18n-t>
        </div>
      </div>
      <div
        class="w-full mt-1 flex flex-row justify-start items-start flex-wrap gap-2"
      >
        <NTag
          v-for="database in state.selectedDatabases"
          :key="database"
          :closable="database !== tabStore.currentTab?.connection.database"
          @close="() => handleUncheckDatabase(database)"
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
            :database-group-name="databaseGroupName"
            @uncheck="handleUncheckDatabaseGroup"
          />
        </template>
        <NDivider v-if="tabStore.isInBatchMode" class="!my-2" />
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
    <NDivider v-if="tabStore.currentTab" class="!my-3" />
    <div class="flex flex-row gap-x-0.5 px-1 items-center">
      <AdvancedSearch
        v-model:params="state.params"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
        :override-route-query="false"
      />
    </div>
    <div
      ref="treeContainerElRef"
      class="relative sql-editor-tree--tree flex-1 px-1 pb-1 text-sm select-none"
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
            class="flex flex-col space-y-2 pt-2 pb-2"
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

    <DatabaseHoverPanel :offset-x="4" :offset-y="4" :margin="4" />

    <MaskSpinner v-if="treeStore.state !== 'READY'" class="!bg-white/75">
      <span class="text-control text-sm">{{
        $t("sql-editor.loading-databases")
      }}</span>
    </MaskSpinner>
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_BATCH_QUERY"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { isEqual } from "lodash-es";
import { InfoIcon } from "lucide-vue-next";
import {
  NTag,
  NButton,
  NTree,
  NDropdown,
  NDivider,
  NRadioGroup,
  NRadio,
  NTooltip,
  NCheckbox,
  type TreeOption,
} from "naive-ui";
import { ref, shallowRef, nextTick, watch, h, computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { RichDatabaseName } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  featureToRef,
  batchGetOrFetchDatabases,
  useProjectV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
  useSQLEditorTabStore,
  useCurrentUserV1,
  resolveOpeningDatabaseListFromSQLEditorTabList,
  useSQLEditorTreeStore,
  useSQLEditorStore,
  idForSQLEditorTreeNodeTarget,
  useInstanceResourceByName,
  useEnvironmentV1List,
  type DatabaseFilter,
} from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import type {
  ComposedDatabase,
  SQLEditorTreeNode,
  CoreSQLEditorTab,
  QueryDataSourceType,
} from "@/types";
import {
  DEFAULT_SQL_EDITOR_TAB_MODE,
  isValidDatabaseName,
  getDataSourceTypeI18n,
  isValidProjectName,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  findAncestor,
  isDescendantOf,
  isDatabaseV1Queryable,
  CommonFilterScopeIdList,
  extractProjectResourceName,
  emptySQLEditorConnection,
  tryConnectToCoreSQLEditorTab,
  useDynamicLocalStorage,
} from "@/utils";
import type { SearchParams } from "@/utils";
import { useSQLEditorContext } from "../../context";
import BatchQueryDatabaseGroupSelector from "./BatchQueryDatabaseGroupSelector.vue";
import DatabaseGroupTag from "./DatabaseGroupTag.vue";
import {
  DatabaseHoverPanel,
  provideHoverStateContext,
} from "./DatabaseHoverPanel";
import { Label } from "./TreeNode";
import { setConnection, useDropdown } from "./actions";
import { useSQLEditorTreeByEnvironment, type TreeByEnvironment } from "./tree";

interface LocalState {
  selectedDatabases: Set<string>;
  params: SearchParams;
  showFeatureModal: boolean;
  showEmptyEnvironment: boolean;
  batchQueryDataSourceType?: QueryDataSourceType;
}

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
  selectedDatabases: new Set(),
  params: {
    query: "",
    scopes: [],
  },
  showFeatureModal: false,
  showEmptyEnvironment: false,
  batchQueryDataSourceType: DataSourceType.READ_ONLY,
});

const flattenSelectedDatabasesFromGroup = computed(() => {
  const nameMap = new Map<
    string /* database name */,
    string /* group title */
  >();
  for (const groupName of tabStore.currentTab?.batchQueryContext
    ?.databaseGroups ?? []) {
    const group = dbGroupStore.getDBGroupByName(groupName);
    if (!group) {
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
    const databases = tabStore.currentTab.batchQueryContext?.databases ?? [];
    databases.push(tabStore.currentTab.connection.database);
    await batchGetOrFetchDatabases(databases);
    state.selectedDatabases = new Set(databases.filter(isValidDatabaseName));
  },
  {
    immediate: true,
  }
);

watch(
  () => [...state.selectedDatabases],
  (selectedDatabases) => {
    // If the current tab is not connected to any database, we need to connect it to the first selected database.
    if (
      (!tabStore.currentTab ||
        isEqual(emptySQLEditorConnection(), tabStore.currentTab.connection)) &&
      selectedDatabases.length > 0
    ) {
      const database = databaseStore.getDatabaseByName(selectedDatabases[0]);
      const coreTab: CoreSQLEditorTab = {
        connection: {
          ...emptySQLEditorConnection(),
          instance: database.instance,
          database: database.name,
        },
        worksheet: "",
        mode: DEFAULT_SQL_EDITOR_TAB_MODE,
      };
      tryConnectToCoreSQLEditorTab(coreTab);
    }
    tabStore.updateBatchQueryContext({
      databases: selectedDatabases,
    });
  }
);

const handleUncheckDatabase = (database: string) => {
  state.selectedDatabases.delete(database);
};

const handleUncheckDatabaseGroup = (databaseGroupName: string) => {
  tabStore.updateBatchQueryContext({
    databaseGroups: (
      tabStore.currentTab?.batchQueryContext?.databaseGroups ?? []
    ).filter((name) => name !== databaseGroupName),
  });
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

const connect = (node: SQLEditorTreeNode) => {
  if (!isDatabaseV1Queryable(node.meta.target as ComposedDatabase)) {
    return;
  }
  setConnection(node, {
    extra: {
      worksheet: tabStore.currentTab?.worksheet ?? "",
      mode: tabStore.currentTab?.mode ?? DEFAULT_SQL_EDITOR_TAB_MODE,
    },
    context: editorContext,
  });
  if (tabStore.currentTab?.batchQueryContext?.databases.length === 1) {
    tabStore.updateBatchQueryContext({
      databases: [],
    });
  }
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
  if (tabStore.currentTab?.connection.database === databaseName) {
    checkTooltip = t("sql-editor.current-connection");
  } else if (!!checkedByGroup) {
    checkTooltip = t("sql-editor.matched-in-group", { title: checkedByGroup });
  }

  return h(Label, {
    node,
    checked: state.selectedDatabases.has(databaseName) || !!checkedByGroup,
    checkDisabled: !!checkedByGroup,
    checkTooltip,
    keyword: state.params.query,
    connected: connectedDatabases.value.has(databaseName),
    "onUpdate:checked": (checked: boolean) => {
      if (node.meta.type !== "database") {
        return;
      }
      if (checked) {
        if (state.selectedDatabases.size === 0) {
          return connect(node);
        }
        if (!hasBatchQueryFeature.value) {
          state.showFeatureModal = true;
          return;
        }
        state.selectedDatabases.add(databaseName);
      } else {
        state.selectedDatabases.delete(databaseName);
      }
    },
  });
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
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
          hoverPosition.value.x = bounding.right;
          hoverPosition.value.y = bounding.top;
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
  return state.params.scopes
    .filter((scope) => scope.id === "label")
    .map((scope) => scope.value);
});

const selectedInstance = computed(() => {
  const instanceId = state.params.scopes.find(
    (scope) => scope.id === "instance"
  )?.value;
  if (!instanceId) {
    return;
  }
  return `${instanceNamePrefix}${instanceId}`;
});

const selectedEngines = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "engine")
    .map((scope) => {
      // Convert from string engine name to proto-es enum value
      const engineName = scope.value.toUpperCase();
      return (
        Engine[engineName as keyof typeof Engine] ?? Engine.ENGINE_UNSPECIFIED
      );
    });
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
  @apply !pl-0 text-sm;
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
  @apply shrink-0 !mr-1;
}
.sql-editor-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  @apply hidden;
}
.sql-editor-tree :deep(.n-tree-node-content__text) {
  @apply truncate mr-1;
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
