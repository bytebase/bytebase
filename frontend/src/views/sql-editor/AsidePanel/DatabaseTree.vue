<template>
  <div
    v-if="!sqlEditorStore.isLoadingTree"
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
    <div class="databases-tree--tree overflow-y-scroll">
      <n-tree
        block-line
        expand-on-click
        :data="treeData"
        :pattern="searchPattern"
        :default-expanded-keys="defaultExpandedKeys"
        :selected-keys="selectedKeys"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :on-load="loadSubTree"
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
import { DropdownOption } from "naive-ui";
import { ref, computed, h, nextTick, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import { ConnectionAtom, UNKNOWN_ID } from "@/types";
import {
  useDatabaseStore,
  useInstanceStore,
  useSQLEditorStore,
  useTableStore,
  useTabStore,
} from "@/store";
import {
  connectionSlug,
  emptyConnection,
  getHighlightHTMLByKeyWords,
  mapConnectionAtom,
} from "@/utils";
import OpenConnectionIcon from "@/components/SQLEditor/OpenConnectionIcon.vue";
import { generateInstanceNode, generateTableItem } from "./utils";

type Position = {
  x: number;
  y: number;
};

type DropdownOptionWithConnectionAtom = DropdownOption & {
  item: ConnectionAtom;
};

const { t } = useI18n();

const router = useRouter();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const sqlEditorStore = useSQLEditorStore();
const tabStore = useTabStore();

const defaultExpandedKeys = ref<string[]>([]);
const searchPattern = ref();
const showDropdown = ref(false);
const dropdownPosition = ref<Position>({
  x: 0,
  y: 0,
});
const selectedKeys = ref<string[] | number[]>([]);
const dropdownContext = ref<ConnectionAtom>();

onMounted(() => {
  const { instanceId, databaseId } = tabStore.currentTab.connection;
  const keys: string[] = [];
  if (instanceId !== UNKNOWN_ID) {
    keys.push(`instance-${instanceId}`);
  }
  if (databaseId !== UNKNOWN_ID) {
    keys.push(`database-${databaseId}`);
  }
  defaultExpandedKeys.value = keys;
});

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
    return [
      {
        key: "open-connection",
        label: t("sql-editor.open-connection"),
        item: dropdownContext.value,
      },
    ];
  }
});

const generateTreeData = () => {
  return sqlEditorStore.connectionTree.map((item) =>
    generateInstanceNode(item, instanceStore)
  );
};

const treeData = ref<ConnectionAtom[]>([]);

watch(
  () => sqlEditorStore.connectionTree,
  () => {
    treeData.value = generateTreeData();
  },
  { immediate: true }
);

const setConnection = (option: ConnectionAtom) => {
  if (option) {
    const conn = emptyConnection();

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
      conn.instanceId = instanceId;
      conn.databaseId = databaseId;
      conn.tableId = option.id;
    }

    if (tabStore.currentTab.sheetId) {
      // We won't mutate a saved sheet's connection.
      // So we'll set connection in a temp or new tab.
      tabStore.selectOrAddTempTab();
    }
    tabStore.updateCurrentTab({
      connection: conn,
    });

    // TODO(Jim): move the URL sync logic to <ProvideSQLEditorContext>
    if (conn.instanceId !== UNKNOWN_ID && conn.databaseId !== UNKNOWN_ID) {
      const database = useDatabaseStore().getDatabaseById(
        conn.databaseId,
        conn.instanceId
      );
      router.replace({
        name: "sql-editor.detail",
        params: {
          connectionSlug: connectionSlug(database),
        },
      });
    }

    // TODO(Jim): This part is for <TableSchema> only
    // and should be removed after upcoming refactor.
    sqlEditorStore.setConnectionContext({
      option,
    });
  }
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: ConnectionAtom }) => {
  const renderLabelHTML = searchPattern.value
    ? h("span", {
        innerHTML: getHighlightHTMLByKeyWords(
          escape(option.label),
          escape(searchPattern.value)
        ),
        class: "truncate",
      })
    : escape(option.label);

  return renderLabelHTML;
};

// render the suffix icon
const renderSuffix = ({ option }: { option: ConnectionAtom }) => {
  const renderSuffixHTML = h(OpenConnectionIcon, {
    id: "tree-node-suffix",
    class: "n-tree-node-content__suffix-icon",
    onClick: function () {
      setConnection(option);
    },
  });

  return renderSuffixHTML;
};

const loadSubTree = async (item: ConnectionAtom) => {
  const mapper = mapConnectionAtom("table", item.id);
  if (item.type === "database") {
    const tableList = await useTableStore().fetchTableListByDatabaseId(item.id);

    item.children = tableList.map((table) => generateTableItem(mapper(table)));
    return;
  }
};

const gotoAlterSchema = (option: ConnectionAtom) => {
  const databaseId = option.parentId;
  const database = databaseStore.getDatabaseById(databaseId);
  if (database.id === UNKNOWN_ID) {
    return;
  }

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.database.schema.update",
      name: `[${database.name}] Alter schema`,
      project: database.project.id,
      databaseList: databaseId,
      sql: `ALTER TABLE ${option.label}`,
    },
  });
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

const nodeProps = (info: { option: ConnectionAtom }) => {
  const { option } = info;

  return {
    onClick(e: MouseEvent) {
      // TODO(Jim): This part is for <TableSchema> only
      // and should be removed after upcoming refactor.
      const targetEl = e.target as HTMLElement;
      if (option && targetEl.className === "n-tree-node-content__text") {
        sqlEditorStore.setConnectionContext({
          option,
        });
      }
    },
    onContextmenu(e: MouseEvent) {
      e.preventDefault();
      showDropdown.value = false;
      if (option && option.key) {
        dropdownContext.value = option;
        selectedKeys.value = [option.key as string];
      }

      nextTick().then(() => {
        showDropdown.value = true;
        dropdownPosition.value.x = e.clientX;
        dropdownPosition.value.y = e.clientY;
      });
    },
  };
};
</script>

<style>
.n-tree-node-content__prefix {
  @apply shrink-0;
}
.n-tree-node-content__text {
  @apply truncate mr-1;
}
.n-tree-node-content__suffix {
  display: none !important;
}

.n-tree-node:hover .n-tree-node-content__suffix {
  display: block !important;
}
</style>

<style scoped>
.databases-tree--tree {
  height: calc(100% - 40px);
}
</style>
