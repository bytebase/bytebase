<template>
  <div
    v-if="!sqlEditorStore.connectionContext.isLoadingTree"
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
        v-model:expanded-keys="expandedTreeKeys"
        block-line
        :data="treeData"
        :pattern="searchPattern"
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
        :on-clickoutside="handleClickOutside"
        @select="handleSelectDropdownItem"
      />
    </div>
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, omit, escape } from "lodash-es";
import { ref, computed, h, nextTick, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { DropdownOption } from "naive-ui";
import { ConnectionAtom, Database, UNKNOWN_ID } from "@/types";
import {
  useDatabaseStore,
  useInstanceStore,
  useSQLEditorStore,
  useTableStore,
  useTabStore,
} from "@/store";
import {
  connectionSlug,
  getHighlightHTMLByKeyWords,
  mapConnectionAtom,
} from "@/utils";
import OpenConnectionIcon from "@/components/SQLEditor/OpenConnectionIcon.vue";
import { generateInstanceNode, generateTableItem } from "./utils";
import { storeToRefs } from "pinia";

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
const sqlEditorStore = useSQLEditorStore();
const tabStore = useTabStore();

const { expandedTreeKeys } = storeToRefs(sqlEditorStore);
const searchPattern = ref("");
const showDropdown = ref(false);
const dropdownPosition = ref<Position>({
  x: 0,
  y: 0,
});
const selectedKeys = ref<string[] | number[]>([]);
const dropdownContext = ref<ConnectionAtom>();

onMounted(() => {
  const { instanceId, databaseId } = sqlEditorStore.connectionContext;

  if (
    instanceId &&
    instanceId !== UNKNOWN_ID &&
    databaseId &&
    databaseId !== UNKNOWN_ID
  ) {
    const keys = [`instance-${instanceId}`, `database-${databaseId}`];
    sqlEditorStore.addExpandedTreeKeys(keys);
  }
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

const getFlattenConnectionTree = () => {
  const tree = sqlEditorStore.connectionTree;
  if (!tree) {
    return {};
  }

  const instanceList = tree
    .filter((node) => node.type === "instance")
    .map((item) => omit(item, "children"));

  const allDatabaseList = tree.flatMap((node) => {
    if (node.children && node.children.length > 0) {
      return node.children.filter((node) => node.type === "database");
    }
  }) as ConnectionAtom[];

  const databaseList = allDatabaseList.map((item) => omit(item, "children"));

  const tableList = allDatabaseList
    .filter((item) => item.children && item.children.length > 0)
    .flatMap((db: ConnectionAtom) => {
      if (db.children) {
        return db.children.filter((node) => node.type === "table");
      }
    });

  return {
    instanceList,
    databaseList,
    tableList,
  };
};

const setConnectionContext = (option: ConnectionAtom) => {
  if (!option) return;

  if (tabStore.currentTab.sheetId || tabStore.currentTab.isModified) {
    // When
    // 1. The current tab is a saved sheet, its connection can't be changed.
    // 2. The current tab is an unsaved but modified tab.
    // we won't mutate it in-place, we will open a new tab instead.
    tabStore.addTab();
  }

  const ctx = cloneDeep(sqlEditorStore.connectionContext);
  const { databaseList } = getFlattenConnectionTree();

  // If selected item is instance node
  if (option.type === "instance") {
    ctx.instanceId = option.id;
    ctx.databaseId = UNKNOWN_ID;
    ctx.tableId = UNKNOWN_ID;
    ctx.tableName = "";
  } else if (option.type === "database") {
    // If selected item is database node
    const instanceId = option.parentId;
    ctx.instanceId = instanceId;
    ctx.databaseId = option.id;
    ctx.tableId = UNKNOWN_ID;
    ctx.tableName = "";
  } else if (option.type === "table") {
    // If selected item is table node
    const databaseId = option.parentId;
    const databaseInfo = databaseList?.find((item) => item.id === databaseId);
    const instanceId = databaseInfo?.parentId || UNKNOWN_ID;
    ctx.instanceId = instanceId;
    ctx.databaseId = databaseId;
    ctx.tableId = option.id;
    ctx.tableName = option.label;
  }

  sqlEditorStore.setConnectionContext(ctx);

  if (ctx.instanceId !== UNKNOWN_ID && ctx.databaseId !== UNKNOWN_ID) {
    const database = useDatabaseStore().getDatabaseById(
      ctx.databaseId,
      ctx.instanceId
    );
    router.replace({
      name: "sql-editor.detail",
      params: {
        connectionSlug: connectionSlug(database),
      },
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
      setConnectionContext(option);
    },
  });

  return renderSuffixHTML;
};

const loadSubTree = async (item: ConnectionAtom) => {
  const mapper = mapConnectionAtom("table", item.id);
  if (item.type === "database") {
    const tableList = await useTableStore().getOrFetchTableListByDatabaseId(
      item.id
    );

    item.children = tableList.map((table) => generateTableItem(mapper(table)));
    return;
  }
};

const gotoAlterSchema = (option: ConnectionAtom) => {
  const databaseId = option.parentId;
  const projectId = sqlEditorStore.findProjectIdByDatabaseId(databaseId);
  const databaseList =
    sqlEditorStore.connectionInfo.databaseListByProjectId.get(projectId);
  const database = databaseList?.find(
    (database: Database) => database.id === databaseId
  );
  if (!database) {
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
      project: projectId,
      databaseList: databaseId,
      sql: `ALTER TABLE ${option.label}`,
    },
  });
};

const handleSelectDropdownItem = (key: string) => {
  const option = dropdownOptions.value.find((item) => item.key === key);
  if (!option) {
    return;
  }

  if (key === "alter-table") {
    gotoAlterSchema(option.item);
  } else if (key === "open-connection") {
    setConnectionContext(option.item);
  }

  showDropdown.value = false;
};

const handleClickOutside = () => {
  showDropdown.value = false;
};

const nodeProps = (info: { option: ConnectionAtom }) => {
  const { option } = info;

  return {
    onClick() {
      setConnectionContext(option);
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
