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
    <div class="databases-tree--tree overflow-y-auto">
      <n-tree
        block-line
        :data="treeData"
        :pattern="searchPattern"
        :default-expanded-keys="defaultExpanedKeys"
        :default-selected-keys="defaultSelectedKeys"
        :selected-keys="selectedKeys"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
      />
      <n-dropdown
        trigger="manual"
        placement="bottom-start"
        :show="showDropdown"
        :options="dropdownOptions"
        :x="x"
        :y="y"
        @select="handleSelect"
        @clickoutside="handleClickoutside"
      />
    </div>
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, h } from "vue";
import { cloneDeep, omit, escape } from "lodash-es";
import { useRouter } from "vue-router";
import { TreeOption, DropdownOption } from "naive-ui";

import {
  ConnectionAtom,
  ConnectionContext,
  Database,
  UNKNOWN_ID,
} from "@/types";
import {
  useDatabaseStore,
  useInstanceStore,
  useTabStore,
  useSQLEditorStore,
} from "@/store";
import { connectionSlug, getHighlightHTMLByKeyWords } from "@/utils";
import InstanceEngineIconVue from "@/components/InstanceEngineIcon.vue";
import HeroiconsOutlineDatabase from "~icons/heroicons-outline/database.vue";
import HeroiconsOutlineTable from "~icons/heroicons-outline/table.vue";
import HeroiconsSolidDotsHorizontal from "~icons/heroicons-solid/dots-horizontal.vue";

const router = useRouter();
const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const searchPattern = ref();
const showDropdown = ref(false);
const x = ref(0);
const y = ref(0);
const selectedKeys = ref<string[] | number[]>([]);
const sheetContext = ref<DropdownOption>();

const dropdownOptions = computed(() => {
  if (!sheetContext.value) {
    return [];
  } else if (sheetContext.value.type === "table") {
    return [
      {
        label: "Alter table",
        key: "editor.sheet.alter-table",
        item: sheetContext.value,
      },
    ];
  } else {
    return [
      {
        label: "Open in new tab",
        key: "editor.sheet.new",
        item: sheetContext.value,
      },
      {
        label: "Set as context",
        key: "editor.sheet.set-context",
        item: sheetContext.value,
      },
    ];
  }
});

const treeData = computed(() => {
  const tree = cloneDeep(sqlEditorStore.connectionTree);

  return tree.map((instanceItem) => {
    const instance = instanceStore.getInstanceById(instanceItem.id);

    return {
      ...instanceItem,
      children: instanceItem.children?.map((databaseItem) => {
        return {
          ...databaseItem,
          children: databaseItem.children?.map((tableItem) => {
            return {
              ...tableItem,
              isLeaf: true,
              prefix: () =>
                h(HeroiconsOutlineTable, {
                  class: "h-4 w-4",
                }),
            };
          }),
          prefix: () =>
            h(HeroiconsOutlineDatabase, {
              class: "h-4 w-4",
            }),
        };
      }),
      prefix: () =>
        h(InstanceEngineIconVue, {
          instance,
        }),
    };
  });
});

const defaultExpanedKeys = computed(() => {
  const ctx = sqlEditorStore.connectionContext;
  if (ctx.hasSlug) {
    return [`instance-${ctx.instanceId}`, `database-${ctx.databaseId}`];
  } else {
    return [];
  }
});

const defaultSelectedKeys = computed(() => {
  const ctx = sqlEditorStore.connectionContext;
  if (ctx.hasSlug) {
    return [`database-${ctx.databaseId}`];
  } else {
    return [];
  }
});

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

const setSheetContext = (option: any) => {
  if (option) {
    let ctx = cloneDeep(sqlEditorStore.connectionContext) as ConnectionContext;
    const { instanceList, databaseList } = getFlattenConnectionTree();

    const getInstanceNameByInstanceId = (id: number) => {
      const instance = instanceList?.find((item) => item.id === id);
      return instance ? instance.label : "";
    };
    const getInstanceEngineByInstanceId = (id: number) => {
      const selectedInstance = instanceStore.getInstanceById(id);
      return selectedInstance ? selectedInstance.engine : "MYSQL";
    };

    // If selected item is instance node
    if (option.type === "instance") {
      ctx.instanceId = option.id;
      ctx.instanceName = option.label;
      ctx.databaseId = UNKNOWN_ID;
      ctx.databaseName = "";
      ctx.databaseType = getInstanceEngineByInstanceId(option.id);
      ctx.tableId = UNKNOWN_ID;
      ctx.tableName = "";
    } else if (option.type === "database") {
      // If selected item is database node
      const instanceId = option.parentId;
      ctx.instanceId = instanceId;
      ctx.instanceName = getInstanceNameByInstanceId(instanceId);
      ctx.databaseId = option.id;
      ctx.databaseName = option.label;
      ctx.databaseType = getInstanceEngineByInstanceId(instanceId);
      ctx.tableId = UNKNOWN_ID;
      ctx.tableName = "";
    } else if (option.type === "table") {
      // If selected item is table node
      const databaseId = option.parentId;
      const databaseInfo = databaseList?.find((item) => item.id === databaseId);
      const databaseName = databaseInfo?.label || "";
      const instanceId = databaseInfo?.parentId || UNKNOWN_ID;
      ctx.instanceId = instanceId;
      ctx.instanceName = getInstanceNameByInstanceId(instanceId);
      ctx.databaseId = databaseId;
      ctx.databaseName = databaseName;
      ctx.databaseType = getInstanceEngineByInstanceId(instanceId);
      ctx.tableId = option.id;
      ctx.tableName = option.label;
    }

    ctx.hasSlug = true;
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
  }
};

// dynamic render the highlight keywords
const renderLabel = ({ option }: { option: TreeOption }) => {
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
const renderSuffix = ({ option }: { option: TreeOption }) => {
  const renderSuffixHTML = h(HeroiconsSolidDotsHorizontal, {
    id: "tree-node-suffix",
    class:
      "n-tree-node-content__suffix-icon h-4 w-4 absolute right-0 top-0 hidden",
    onClick: (e: MouseEvent) => {
      sheetContext.value = option;
      showDropdown.value = true;
      x.value = e.clientX;
      y.value = e.clientY;
      e.preventDefault();
    },
  });

  return renderSuffixHTML;
};

const gotoAlterSchema = (option: any) => {
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

  const databaseName = database.name;
  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.database.schema.update",
      name: `[${databaseName}] Alter schema`,
      project: projectId,
      databaseList: databaseId,
      sql: `ALTER TABLE ${option.label}`,
    },
  });
};

const handleSelect = (key: string) => {
  const option = dropdownOptions.value.find(
    (item) => item.key === key
  ) as DropdownOption;

  if (key === "editor.sheet.alter-table") {
    gotoAlterSchema(option.item);
  } else if (key === "editor.sheet.set-context") {
    setSheetContext(option.item);
  } else if (key === "editor.sheet.new") {
    // set the sheet context first
    setSheetContext(option.item);
    // and then create a new tab
    tabStore.addTab();
  }

  showDropdown.value = false;
};

const handleClickoutside = () => {
  showDropdown.value = false;
};

const nodeProps = (info: { option: TreeOption }) => {
  const { option } = info;

  return {
    onClick: (e: MouseEvent) => {
      const targetEl = e.target as HTMLElement;
      if (option && targetEl.className === "n-tree-node-content__text") {
        let ctx = cloneDeep(
          sqlEditorStore.connectionContext
        ) as ConnectionContext;
        ctx.option = option;
        sqlEditorStore.setConnectionContext(ctx);
      }
    },
    onContextmenu(e: MouseEvent) {
      sheetContext.value = option;
      showDropdown.value = true;
      x.value = e.clientX;
      y.value = e.clientY;
      if (option && option.key) {
        selectedKeys.value = [option.key as string];
      }
      e.preventDefault();
    },
  };
};
</script>

<style>
.n-tree
  .n-tree-node.n-tree-node--highlight
  .n-tree-node-content
  .n-tree-node-content__text {
  border-bottom: none;
  border-bottom-color: transparent;
}

.n-tree .n-tree-node-content .n-tree-node-content__text {
  @apply truncate;
}

.n-tree .n-tree-node .n-tree-node-content__suffix {
  @apply w-4 h-4 flex items-center relative ml-1;
}

.n-tree
  .n-tree-node:hover
  .n-tree-node-content__suffix
  .n-tree-node-content__suffix-icon {
  @apply block;
}
</style>

<style scoped>
.databases-tree--tree {
  height: calc(100% - 40px);
}
</style>
