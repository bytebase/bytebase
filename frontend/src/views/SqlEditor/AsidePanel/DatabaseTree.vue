<template>
  <div
    v-if="!connectionContext.isLoadingTree"
    class="databases-tree p-2 space-y-2 h-full"
  >
    <div class="databases-tree--input">
      <NInput
        v-model:value="searchPattern"
        :placeholder="$t('sql-editor.search-databases')"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>
    <div class="databases-tree--tree overflow-y-auto">
      <NTree
        block-line
        :data="treeData"
        :pattern="searchPattern"
        :default-expanded-keys="defaultExpanedKeys"
        :default-selected-keys="defaultSelectedKeys"
        :on-update:selected-keys="handleSelectedKeysChange"
        :render-label="renderLabel"
      />
    </div>
  </div>
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, h } from "vue";
import {
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";
import { cloneDeep, omit, escape } from "lodash-es";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import { TreeOption } from "naive-ui";

import {
  SqlEditorState,
  ConnectionAtom,
  SqlEditorActions,
  ConnectionContext,
  UNKNOWN_ID,
} from "../../../types";
import { connectionSlug, getHighlightHTMLByKeyWords } from "../../../utils";
import InstanceEngineIconVue from "../../../components/InstanceEngineIcon.vue";
import HeroiconsOutlineDatabase from "~icons/heroicons-outline/database.vue";
import HeroiconsOutlineTable from "~icons/heroicons-outline/table.vue";

const store = useStore();
const router = useRouter();

const searchPattern = ref();
const { connectionTree, connectionContext } =
  useNamespacedState<SqlEditorState>("sqlEditor", [
    "connectionTree",
    "connectionContext",
  ]);
const { setConnectionContext } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setConnectionContext"]
);

const treeData = computed(() => {
  const tree = cloneDeep(connectionTree.value);

  // mapping the prefix icons
  return tree.map((instanceItem) => {
    const instance = store.getters["instance/instanceById"](instanceItem.id);

    return {
      ...instanceItem,
      children: instanceItem?.children?.map((databaseItem) => {
        return {
          ...databaseItem,
          children: databaseItem?.children?.map((tableItem) => {
            return {
              ...tableItem,
              prefix: () => h(HeroiconsOutlineTable, { class: "h-4 w-4" }),
            };
          }),
          prefix: () =>
            h(HeroiconsOutlineDatabase, {
              class: "h-4 w-4",
            }),
        };
      }),
      prefix: () => h(InstanceEngineIconVue, { instance }),
    };
  });
});

const defaultExpanedKeys = computed(() => {
  const ctx = connectionContext.value;
  if (ctx.hasSlug) {
    return [`instance-${ctx.instanceId}`, `database-${ctx.databaseId}`];
  } else {
    return [];
  }
});

const defaultSelectedKeys = computed(() => {
  const ctx = connectionContext.value;
  if (ctx.hasSlug) {
    return [`database-${ctx.databaseId}`];
  } else {
    return [];
  }
});

const getFlattenConnectionTree = () => {
  const tree = connectionTree.value;
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

const handleSelectedKeysChange = (
  keys: number[],
  options: Array<ConnectionAtom>
) => {
  const [selectedItem] = options;

  if (selectedItem) {
    let ctx: ConnectionContext = cloneDeep(connectionContext.value);
    const { instanceList, databaseList } = getFlattenConnectionTree();

    const getInstanceNameByInstanceId = (id: number) => {
      const instance = instanceList?.find((item) => item.id === id);
      return instance ? instance.label : "";
    };

    // If selected item is instance node
    if (selectedItem.type === "instance") {
      ctx.instanceId = selectedItem.id;
      ctx.instanceName = selectedItem.label;
      ctx.databaseId = UNKNOWN_ID;
      ctx.databaseName = "";
      ctx.tableId = UNKNOWN_ID;
      ctx.tableName = "";
    }

    // If selected item is database node
    if (selectedItem.type === "database") {
      const instanceId = selectedItem.parentId;
      ctx.instanceId = instanceId;
      ctx.instanceName = getInstanceNameByInstanceId(instanceId);
      ctx.databaseId = selectedItem.id;
      ctx.databaseName = selectedItem.label;
      ctx.tableId = UNKNOWN_ID;
      ctx.tableName = "";
    }

    // If selected item is table node
    if (selectedItem.type === "table") {
      const databaseId = selectedItem.parentId;
      const databaseInfo = databaseList?.find((item) => item.id === databaseId);
      const databaseName = databaseInfo?.label || "";
      const instanceId = databaseInfo?.parentId || UNKNOWN_ID;
      ctx.instanceId = instanceId;
      ctx.instanceName = getInstanceNameByInstanceId(instanceId);
      ctx.databaseId = databaseId;
      ctx.databaseName = databaseName;
      ctx.tableId = selectedItem.id;
      ctx.tableName = selectedItem.label;
    }

    ctx.hasSlug = true;
    setConnectionContext(ctx);

    if (ctx.instanceId !== UNKNOWN_ID && ctx.databaseId !== UNKNOWN_ID) {
      const database = store.getters["database/databaseById"](
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

const renderLabel = ({ option }: { option: TreeOption }) => {
  const renderLabelHTML = searchPattern.value
    ? h("span", {
        innerHTML: getHighlightHTMLByKeyWords(
          escape(option.label),
          escape(searchPattern.value)
        ),
      })
    : escape(option.label);

  return renderLabelHTML;
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
</style>

<style scoped>
.databases-tree--tree {
  height: calc(100% - 40px);
}
</style>
