<template>
  <div class="sql-editor-tree gap-y-1 h-full flex flex-col relative">
    <div class="w-full px-4 mt-4">
      <div
        class="textinfolabel mb-2 w-full leading-4 flex flex-col lg:flex-row items-start lg:items-center gap-x-1"
      >
        <div class="flex items-center gap-x-1">
          <FeatureBadge feature="bb.feature.batch-query" />
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
      </div>
    </div>
    <NDivider v-if="tabStore.currentTab" class="!my-3" />
    <div class="flex flex-row gap-x-0.5 px-1 items-center">
      <AdvancedSearch
        v-model:params="state.params"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <GroupingBar :disabled="editorStore.loading" class="shrink-0" />
    </div>
    <div
      v-if="hasMissingQueryDatabases"
      class="flex items-center space-x-2 px-2 py-2"
    >
      <NCheckbox
        :disabled="editorStore.loading"
        v-model:checked="showMissingQueryDatabases"
      >
        <span class="textinfolabel text-sm">
          {{ $t("sql-editor.show-databases-without-query-permission") }}
        </span>
      </NCheckbox>
    </div>
    <div
      ref="treeContainerElRef"
      class="relative sql-editor-tree--tree flex-1 px-1 pb-1 text-sm select-none"
      :data-height="treeContainerHeight"
    >
      <div
        v-if="treeStore.state === 'READY'"
        class="flex flex-col space-y-2 pt-2 pb-4"
      >
        <NTree
          ref="treeRef"
          :block-line="true"
          :data="treeStore.tree"
          :show-irrelevant-nodes="false"
          :selected-keys="selectedKeys"
          :default-expand-all="true"
          :expand-on-click="true"
          :node-props="nodeProps"
          :theme-overrides="{ nodeHeight: '21px' }"
          :render-label="renderLabel"
        />
        <div
          v-if="editorStore.canLoadMore"
          class="w-full flex items-center justify-center"
        >
          <NButton
            quaternary
            :size="'small'"
            :loading="editorStore.loading"
            @click="
              () =>
                editorStore
                  .fetchDatabases(filter)
                  .then(() => treeStore.buildTree())
            "
          >
            {{ $t("common.load-more") }}
          </NButton>
        </div>
      </div>
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
    feature="bb.feature.batch-query"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { head, isEqual } from "lodash-es";
import {
  NTag,
  NButton,
  NTree,
  NDropdown,
  NCheckbox,
  NDivider,
  type TreeOption,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { ref, nextTick, watch, h, computed, reactive } from "vue";
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
  useSQLEditorTabStore,
  resolveOpeningDatabaseListFromSQLEditorTabList,
  useSQLEditorTreeStore,
  useSQLEditorStore,
  idForSQLEditorTreeNodeTarget,
  useInstanceResourceByName,
} from "@/store";
import {
  instanceNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import type {
  ComposedDatabase,
  SQLEditorTreeNode,
  CoreSQLEditorTab,
} from "@/types";
import { DEFAULT_SQL_EDITOR_TAB_MODE, isValidDatabaseName } from "@/types";
import { engineFromJSON } from "@/types/proto/v1/common";
import {
  findAncestor,
  isDescendantOf,
  isDatabaseV1Queryable,
  CommonFilterScopeIdList,
  extractProjectResourceName,
  emptySQLEditorConnection,
  tryConnectToCoreSQLEditorTab,
} from "@/utils";
import type { SearchParams } from "@/utils";
import { useSQLEditorContext } from "../../context";
import BatchQueryDatabaseGroupSelector from "./BatchQueryDatabaseGroupSelector.vue";
import DatabaseGroupTag from "./DatabaseGroupTag.vue";
import {
  DatabaseHoverPanel,
  provideHoverStateContext,
} from "./DatabaseHoverPanel";
import GroupingBar from "./GroupingBar";
import { Label } from "./TreeNode";
import { setConnection, useDropdown } from "./actions";

interface LocalState {
  selectedDatabases: Set<string>;
  params: SearchParams;
  showFeatureModal: boolean;
}

const treeStore = useSQLEditorTreeStore();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();

const hasBatchQueryFeature = featureToRef("bb.feature.batch-query");
const hasDatabaseGroupFeature = featureToRef("bb.feature.database-grouping");

const state = reactive<LocalState>({
  selectedDatabases: new Set(),
  params: {
    query: "",
    scopes: [],
  },
  showFeatureModal: false,
});

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
    tabStore.updateCurrentTab({
      batchQueryContext: {
        databases: selectedDatabases,
        databaseGroups:
          tabStore.currentTab?.batchQueryContext?.databaseGroups ?? [],
      },
    });
  }
);

const handleUncheckDatabase = (database: string) => {
  state.selectedDatabases.delete(database);
};

const handleUncheckDatabaseGroup = (databaseGroupName: string) => {
  tabStore.updateCurrentTab({
    batchQueryContext: {
      databases: tabStore.currentTab?.batchQueryContext?.databases ?? [],
      databaseGroups: (
        tabStore.currentTab?.batchQueryContext?.databaseGroups ?? []
      ).filter((name) => name !== databaseGroupName),
    },
  });
};

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList,
  "database-label",
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
const treeRef = ref<InstanceType<typeof NTree>>();
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
    const node = head(treeStore.nodesByTarget("database", database));
    if (!node) return [];
    return [node.key];
  } else if (connection.instance) {
    const { instance } = useInstanceResourceByName(connection.instance);
    const nodes = treeStore.nodesByTarget("instance", instance.value);
    return nodes.map((node) => node.key);
  }
  return [];
};

const connectedDatabases = computed(() =>
  resolveOpeningDatabaseListFromSQLEditorTabList()
);

const { hasMissingQueryDatabases, showMissingQueryDatabases } =
  storeToRefs(treeStore);

const connect = (node: SQLEditorTreeNode) => {
  if (!isDatabaseV1Queryable(node.meta.target as ComposedDatabase)) {
    return;
  }
  setConnection(node, {
    extra: {
      worksheet: tabStore.currentTab?.worksheet ?? "",
      mode: DEFAULT_SQL_EDITOR_TAB_MODE,
    },
    context: editorContext,
  });
  tabStore.updateCurrentTab({
    batchQueryContext: {
      databases: [],
      databaseGroups: [],
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

  return h(Label, {
    node,
    checked: state.selectedDatabases.has(databaseName),
    factors: treeStore.filteredFactorList,
    keyword: state.params.query,
    connected: connectedDatabases.value.has(databaseName),
    connectedDatabases: connectedDatabases.value,
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

watch(
  selectedKeys,
  (keys) => {
    if (keys.length !== 1) return;
    const key = keys[0];
    nextTick(() => {
      treeRef.value?.scrollTo({ key });
    });
  },
  { immediate: true }
);

useEmitteryEventListener(editorEvents, "tree-ready", async () => {
  selectedKeys.value = await getSelectedKeys();
});

const selectedLabels = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "database-label")
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

const selectedEnvironment = computed(() => {
  const environmentId = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (!environmentId) {
    return;
  }
  return `${environmentNamePrefix}${environmentId}`;
});

const selectedEngines = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "engine")
    .map((scope) => engineFromJSON(scope.value));
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  query: state.params.query,
  labels: selectedLabels.value,
  engines: selectedEngines.value,
}));

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

watch(
  [
    () => editorStore.project,
    () => editorStore.projectContextReady,
    () => filter.value,
  ],
  async ([_, ready, filter]) => {
    if (!ready) {
      treeStore.state = "LOADING";
    } else {
      await editorStore.prepareDatabases(filter);
      treeStore.buildTree();
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
