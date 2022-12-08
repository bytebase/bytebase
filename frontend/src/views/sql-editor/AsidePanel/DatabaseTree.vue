<template>
  <div
    v-if="sqlEditorStore.connectionTree.state === ConnectionTreeState.LOADED"
    class="databases-tree p-2 space-y-2 h-full"
  >
    <div class="databases-tree--input">
      <n-input
        v-model:value="searchPattern"
        :placeholder="$t('sql-editor.search-databases')"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </n-input>
    </div>
    <div class="databases-tree--tree overflow-y-auto">
      <n-tree
        block-line
        :data="treeData"
        :pattern="searchPattern"
        :selected-keys="selectedKeys"
        :expanded-keys="sqlEditorStore.expandedTreeNodeKeys"
        :render-label="renderLabel"
        :render-prefix="renderPrefix"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :on-load="loadSubTree"
        :on-update:expanded-keys="updateExpandedKeys"
      />
      <n-dropdown
        placement="bottom-start"
        trigger="manual"
        :x="dropdownPosition.x"
        :y="dropdownPosition.y"
        :options="dropdownOptions"
        :show="showDropdown"
        :on-clickoutside="handleClickoutside"
        @select="handleSelect"
      />
    </div>
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
  </div>
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import type { DropdownOption } from "naive-ui";
import { ref, computed, h, nextTick, watch } from "vue";
import { useI18n } from "vue-i18n";
import { stringify } from "qs";
import ConnectedIcon from "~icons/heroicons-outline/lightning-bolt";

import type { ConnectionAtom, DatabaseId, InstanceId } from "@/types";
import { ConnectionTreeState, UNKNOWN_ID } from "@/types";
import {
  useDatabaseStore,
  useInstanceStore,
  useIsLoggedIn,
  useSQLEditorStore,
  useTableStore,
  useTabStore,
} from "@/store";
import {
  emptyConnection,
  getHighlightHTMLByKeyWords,
  isDescendantOf,
  isSameConnection,
  mapConnectionAtom,
} from "@/utils";
import { generateTableItem } from "./utils";
import { scrollIntoViewIfNeeded } from "@/bbkit/BBUtil";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import HeroiconsOutlineDatabase from "~icons/heroicons-outline/database";
import HeroiconsOutlineTable from "~icons/heroicons-outline/table";
import ProtectedEnvironmentIcon from "@/components/Environment/ProtectedEnvironmentIcon.vue";

type Position = {
  x: number;
  y: number;
};

type DropdownOptionWithConnectionAtom = DropdownOption & {
  item: ConnectionAtom;
};

const { t } = useI18n();

const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const sqlEditorStore = useSQLEditorStore();
const tableStore = useTableStore();
const tabStore = useTabStore();
const isLoggedIn = useIsLoggedIn();

const searchPattern = ref();
const showDropdown = ref(false);
const dropdownPosition = ref<Position>({
  x: 0,
  y: 0,
});
const dropdownContext = ref<ConnectionAtom>();
const dropdownOptions = computed((): DropdownOptionWithConnectionAtom[] => {
  if (!dropdownContext.value) {
    return [];
  } else if (dropdownContext.value.type === "table") {
    return [
      {
        key: "alter-table",
        label: t("sql-editor.alter-table"),
        item: dropdownContext.value,
      },
    ];
  } else {
    // Don't show any context menu actions for disabled
    // instances/databases
    if (dropdownContext.value.disabled) {
      return [];
    }
    return [
      {
        key: "open-connection",
        label: t("sql-editor.open-connection"),
        item: dropdownContext.value,
      },
    ];
  }
});

// Highlight the current tab's connection node.
const selectedKeys = computed(() => {
  const { instanceId, databaseId } = tabStore.currentTab.connection;
  if (databaseId !== UNKNOWN_ID) {
    return [`database-${databaseId}`];
  }
  if (instanceId !== UNKNOWN_ID) {
    return [`instance-${instanceId}`];
  }
  return [];
});

const treeData = computed(() => sqlEditorStore.connectionTree.data);

const setConnection = (option: ConnectionAtom) => {
  if (option) {
    const conn = emptyConnection();
    const connect = () => {
      if (isSameConnection(tabStore.currentTab.connection, conn)) {
        // Don't go further if the connection doesn't change.
        return;
      }
      if (tabStore.currentTab.sheetId) {
        // We won't mutate a saved sheet's connection.
        // So we'll set connection in a temp or new tab.
        tabStore.selectOrAddTempTab();
      }
      tabStore.updateCurrentTab({
        connection: conn,
      });
    };

    // If selected item is instance node
    if (option.type === "instance") {
      conn.instanceId = option.id;
    } else if (option.type === "database") {
      // If selected item is database node
      const instanceId = option.parentId;
      conn.instanceId = instanceId;
      conn.databaseId = option.id;
    } else if (option.type === "table") {
      // If selected item is table node
      const databaseId = option.parentId;
      const databaseInfo = databaseStore.getDatabaseById(databaseId);
      const instanceId = databaseInfo.instance.id;
      const tableId = option.id;
      conn.instanceId = instanceId;
      conn.databaseId = databaseId;
      tableStore
        .getOrFetchTableByDatabaseIdAndTableId(databaseId, tableId)
        .then((table) => {
          sqlEditorStore.selectedTable = table;
        });
    }

    connect();
  }
};

// dynamic render the highlight keywords
// and render the selected table node in bold font
const renderLabel = ({ option }: { option: ConnectionAtom }) => {
  const classes = ["truncate"];
  const emphasize = () => {
    classes.push("font-bold");
  };

  if (option.type === "table") {
    if (option.id === sqlEditorStore.selectedTable.id) {
      emphasize();
    }
  }

  return h("span", {
    // render an unique id for every node
    // for auto scroll to the node when tab switches
    id: `tree-node-label-${option.type}-${option.id}`,
    innerHTML: getHighlightHTMLByKeyWords(
      escape(option.label),
      escape(searchPattern.value)
    ),
    class: classes,
  });
};

// Render icons before nodes.
const renderPrefix = ({ option }: { option: ConnectionAtom }) => {
  if (option.type === "instance") {
    const instanceId = option.id;
    const instance = instanceStore.getInstanceById(instanceId);
    const children = [
      h(InstanceEngineIcon, {
        instance,
      }),
      h(ProtectedEnvironmentIcon, {
        environment: instance.environment,
        class: "w-4 h-4 text-inherit",
      }),
      h(
        "span",
        {
          class: ["text-sm", !option.disabled && "text-gray-500"],
        },
        `(${instance.environment.name})`
      ),
    ];

    return h("span", { class: "flex items-center gap-x-1" }, children);
  } else if (option.type === "database") {
    return h(HeroiconsOutlineDatabase, {
      class: "w-4 h-4",
    });
  } else if (option.type === "table") {
    return h(HeroiconsOutlineTable, {
      class: "w-4 h-4",
    });
  }
  return null;
};

// Render a 'connected' icon in the right of the node
// if it matches the current tab's connection
const renderSuffix = ({ option }: { option: ConnectionAtom }) => {
  const renderConnectedIcon = () => {
    return h(ConnectedIcon, {
      class: "w-4 h-4",
    });
  };
  const { instanceId, databaseId } = tabStore.currentTab.connection;
  if (option.type === "database") {
    if (option.id === databaseId) {
      return renderConnectedIcon();
    }
  }
  if (option.type === "instance") {
    if (databaseId === UNKNOWN_ID && option.id === instanceId) {
      return renderConnectedIcon();
    }
  }
};

const loadSubTree = async (item: ConnectionAtom): Promise<void> => {
  if (item.type === "database") {
    const tableList = await useTableStore().fetchTableListByDatabaseId(item.id);

    const mapper = mapConnectionAtom("table", item.id);
    item.children = tableList.map((table) => generateTableItem(mapper(table)));
    if (item.children.length === 0) {
      // No tables in the db
      item.isLeaf = true;
      // TODO: this might be a little bit confusing
      // Better add a dummy "no tables" node in the future
    }
  }
};

const gotoAlterSchema = (option: ConnectionAtom) => {
  const databaseId = option.parentId;
  const database = databaseStore.getDatabaseById(databaseId);
  if (database.id === UNKNOWN_ID) {
    return;
  }

  const query = {
    template: "bb.issue.database.schema.update",
    name: `[${database.name}] Alter schema`,
    project: database.project.id,
    databaseList: databaseId,
    sql: `ALTER TABLE ${option.label}`,
  };
  const url = `/issue/new?${stringify(query)}`;
  window.open(url, "_blank");
};

const handleSelect = (key: string) => {
  const option = dropdownOptions.value.find((item) => item.key === key);
  if (!option) {
    return;
  }

  if (key === "alter-table") {
    gotoAlterSchema(option.item);
  } else if (key === "open-connection") {
    setConnection(option.item);
  }

  showDropdown.value = false;
};

const handleClickoutside = () => {
  showDropdown.value = false;
};

const nodeProps = ({ option }: { option: ConnectionAtom }) => {
  return {
    onClick(e: MouseEvent) {
      if (option.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        setConnection(option);
      }
    },
    onContextmenu(e: MouseEvent) {
      e.preventDefault();
      showDropdown.value = false;
      if (option && option.key) {
        dropdownContext.value = option;
      }

      nextTick().then(() => {
        showDropdown.value = true;
        dropdownPosition.value.x = e.clientX;
        dropdownPosition.value.y = e.clientY;
      });
    },
  };
};

const updateExpandedKeys = (keys: string[]) => {
  sqlEditorStore.expandedTreeNodeKeys = keys;
};

// When switching tabs, scroll the matched node into view if needed.
const scrollToConnectedNode = (
  instanceId: InstanceId,
  databaseId: DatabaseId
) => {
  if (instanceId === UNKNOWN_ID && databaseId === UNKNOWN_ID) {
    return;
  }
  let id: string;
  if (databaseId === UNKNOWN_ID) {
    id = `tree-node-label-instance-${instanceId}`;
  } else {
    id = `tree-node-label-database-${databaseId}`;
  }
  nextTick(() => {
    const elem = document.getElementById(id);
    if (elem) {
      scrollIntoViewIfNeeded(elem);
    }
  });
};

// Open corresponding tree node when the connection changed.
watch(
  [
    isLoggedIn,
    () => tabStore.currentTab.connection.instanceId,
    () => tabStore.currentTab.connection.databaseId,
  ],
  ([isLoggedIn, instanceId, databaseId]) => {
    if (!isLoggedIn) {
      // Don't go further and cleanup the state if we signed out.
      sqlEditorStore.expandedTreeNodeKeys = [];
      return;
    }

    const maybeExpandKey = (key: string) => {
      const keys = sqlEditorStore.expandedTreeNodeKeys;
      if (!keys.includes(key)) {
        keys.push(key);
      }
    };

    if (instanceId !== UNKNOWN_ID) {
      maybeExpandKey(`instance-${instanceId}`);
    }
    if (databaseId !== UNKNOWN_ID) {
      maybeExpandKey(`database-${databaseId}`);
    }

    scrollToConnectedNode(instanceId, databaseId);
  },
  { immediate: true, flush: "pre" }
);
</script>

<style>
.databases-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.databases-tree .n-tree-node-content__text {
  @apply truncate mr-1;
}
.databases-tree .n-tree-node--pending {
  background-color: transparent !important;
}
.databases-tree .n-tree-node--pending:hover {
  background-color: var(--n-node-color-hover) !important;
}
.databases-tree .n-tree-node--selected,
.databases-tree .n-tree-node--selected:hover {
  background-color: var(--n-node-color-active) !important;
}
</style>

<style scoped>
.databases-tree--tree {
  height: calc(100% - 40px);
}
</style>
