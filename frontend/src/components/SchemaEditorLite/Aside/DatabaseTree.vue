<template>
  <div
    class="w-full h-full pl-1 pr-2 flex flex-col gap-y-2 relative overflow-y-hidden"
  >
    <div class="w-full sticky top-0 pt-2 h-12 bg-white z-10">
      <NInput
        v-model:value="searchPattern"
        :placeholder="$t('schema-editor.search-database-and-table')"
      >
        <template #prefix>
          <heroicons-outline:search class="w-4 h-auto text-gray-300" />
        </template>
      </NInput>
    </div>
    <div
      ref="treeContainerElRef"
      class="schema-editor-database-tree flex-1 pb-1 text-sm overflow-hidden select-none"
      :data-height="treeContainerHeight"
    >
      <NTree
        v-if="treeContainerHeight > 0"
        ref="treeRef"
        block-line
        virtual-scroll
        :style="{
          height: `${treeContainerHeight}px`,
        }"
        :data="treeDataRef"
        :pattern="searchPattern"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :on-load="loadSubTree"
        :expanded-keys="expandedKeysRef"
        :selected-keys="selectedKeysRef"
        :on-update:expanded-keys="handleExpandedKeysChange"
        :theme-overrides="{ nodeHeight: '28px' }"
      />
      <NDropdown
        trigger="manual"
        placement="bottom-start"
        :show="contextMenu.showDropdown"
        :options="contextMenuOptions"
        :x="contextMenu.clientX"
        :y="contextMenu.clientY"
        to="body"
        @select="handleContextMenuDropdownSelect"
        @clickoutside="handleDropdownClickoutside"
      />
    </div>
  </div>

  <SchemaNameModal
    v-if="state.schemaNameModalContext !== undefined"
    :parent-name="state.schemaNameModalContext.parentName"
    :schema-id="state.schemaNameModalContext.schemaId"
    @close="state.schemaNameModalContext = undefined"
  />

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :parent-name="state.tableNameModalContext.parentName"
    :schema-id="state.tableNameModalContext.schemaId"
    :table-id="state.tableNameModalContext.tableId"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { escape, head, isUndefined, pick } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { computed, onMounted, watch, ref, h, reactive, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import TableIcon from "~icons/heroicons-outline/table-cells";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  generateUniqueTabId,
  useInstanceV1Store,
  useNotificationStore,
  useSchemaEditorV1Store,
} from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { SchemaEditorTabType } from "@/types/v1/schemaEditor";
import { Schema, Table } from "@/types/v1/schemaEditor";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import SchemaNameModal from "../Modals/SchemaNameModal.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { fetchSchemaListByDatabaseName } from "../utils/database";
import { isSchemaChanged } from "../utils/schema";
import { isTableChanged } from "../utils/table";

interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

interface TreeNodeForInstance extends BaseTreeNode {
  type: "instance";
  // Resoure name format.
  instance: string;
}

interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  instance: string;
  // Resoure name format.
  database: string;
}

interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  instance: string;
  database: string;
  schemaId: string;
}

interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  instance: string;
  database: string;
  schemaId: string;
  tableId: string;
}

type TreeNode =
  | TreeNodeForInstance
  | TreeNodeForDatabase
  | TreeNodeForSchema
  | TreeNodeForTable;

interface TreeContextMenu {
  showDropdown: boolean;
  clientX: number;
  clientY: number;
  treeNode?: TreeNode;
}

interface LocalState {
  shouldRelocateTreeNode: boolean;
  schemaNameModalContext?: {
    parentName: string;
    schemaId: string | undefined;
  };
  tableNameModalContext?: {
    parentName: string;
    schemaId: string;
    tableId: string | undefined;
  };
}

const { t } = useI18n();
const schemaEditorV1Store = useSchemaEditorV1Store();
const instanceStore = useInstanceV1Store();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});
const currentTab = computed(() => schemaEditorV1Store.currentTab);

const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});
const treeContainerElRef = ref<HTMLElement>();
const { height: treeContainerHeight } = useElementSize(
  treeContainerElRef,
  undefined,
  {
    box: "content-box",
  }
);
const treeRef = ref<InstanceType<typeof NTree>>();
const searchPattern = ref("");
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
const treeDataRef = ref<TreeNode[]>([]);
const databaseDataLoadedSet = ref<Set<string>>(new Set());

const databaseList = computed(() => schemaEditorV1Store.databaseList);
const schemaList = computed(() =>
  Array.from(schemaEditorV1Store.resourceMap["database"].values())
    .map((item) => item.schemaList)
    .flat()
    .map((schema) => {
      return {
        ...schema,
        tableList: schema.tableList.map((table) => {
          // Don't watch column changes in database tree.
          return pick(table, ["id", "name", "status"]);
        }),
      };
    })
);
const contextMenuOptions = computed(() => {
  const treeNode = contextMenu.treeNode;
  if (isUndefined(treeNode)) {
    return [];
  }
  const instanceEngine = instanceStore.getInstanceByName(
    treeNode.instance
  ).engine;

  if (treeNode.type === "database") {
    const options = [];
    if (instanceEngine === Engine.MYSQL) {
      options.push({
        key: "create-table",
        label: t("schema-editor.actions.create-table"),
      });
    } else if (instanceEngine === Engine.POSTGRES) {
      options.push({
        key: "create-schema",
        label: t("schema-editor.actions.create-schema"),
      });
    }
    return options;
  } else if (treeNode.type === "schema") {
    const options = [];
    if (instanceEngine === Engine.POSTGRES) {
      const schema = schemaEditorV1Store.getSchema(
        treeNode.database,
        treeNode.schemaId
      );
      if (!schema) {
        return [];
      }

      const isDropped = schema.status === "dropped";
      if (isDropped) {
        options.push({
          key: "restore",
          label: t("schema-editor.actions.restore"),
        });
      } else {
        options.push({
          key: "create-table",
          label: t("schema-editor.actions.create-table"),
        });
        options.push({
          key: "rename",
          label: t("schema-editor.actions.rename"),
        });
        options.push({
          key: "drop-schema",
          label: t("schema-editor.actions.drop-schema"),
        });
      }
    }
    return options;
  } else if (treeNode.type === "table") {
    const table = schemaEditorV1Store.getTable(
      treeNode.database,
      treeNode.schemaId,
      treeNode.tableId
    );
    if (!table) {
      return [];
    }

    const isDropped = table.status === "dropped";
    const options = [];
    if (isDropped) {
      options.push({
        key: "restore",
        label: t("schema-editor.actions.restore"),
      });
    } else {
      options.push({
        key: "rename",
        label: t("schema-editor.actions.rename"),
      });
      options.push({
        key: "drop",
        label: t("schema-editor.actions.drop-table"),
      });
    }
    return options;
  }

  return [];
});

onMounted(async () => {
  const treeNodeList: TreeNode[] = [];
  const instanceTreeNodeMap: Map<string, TreeNodeForInstance> = new Map();
  const databaseTreeNodeList: TreeNodeForDatabase[] = [];
  for (const database of databaseList.value) {
    const instance = database.instanceEntity;
    let instanceTreeNode: TreeNodeForInstance;
    if (instanceTreeNodeMap.has(instance.name)) {
      instanceTreeNode = instanceTreeNodeMap.get(
        instance.name
      ) as TreeNodeForInstance;
    } else {
      instanceTreeNode = {
        type: "instance",
        key: `i-${instance.name}`,
        label: instance.title,
        isLeaf: false,
        instance: instance.name,
        children: [],
      };
      instanceTreeNodeMap.set(instance.name, instanceTreeNode);
      treeNodeList.push(instanceTreeNode);
      // Make the instance tree node expanded as default.
      expandedKeysRef.value.push(instanceTreeNode.key);
    }

    const databaseTreeNode: TreeNodeForDatabase = {
      type: "database",
      key: `d-${database.name}`,
      label: database.databaseName,
      isLeaf: false,
      instance: instance.name,
      database: database.name,
    };
    instanceTreeNode.children?.push(databaseTreeNode);
    databaseTreeNodeList.push(databaseTreeNode);
  }
  treeDataRef.value = treeNodeList;
});

watch(
  [() => schemaList.value],
  () => {
    const databaseTreeNodeList: TreeNodeForDatabase[] = [];
    for (const treeNode of treeDataRef.value) {
      if (treeNode.type === "instance") {
        databaseTreeNodeList.push(
          ...(treeNode.children as TreeNodeForDatabase[])
        );
      }
    }

    for (const database of databaseList.value) {
      if (!databaseDataLoadedSet.value.has(database.name)) {
        continue;
      }
      const databaseTreeNode = databaseTreeNodeList.find(
        (treeNode) =>
          treeNode.database === database.name &&
          databaseDataLoadedSet.value.has(treeNode.database)
      );
      if (isUndefined(databaseTreeNode)) {
        continue;
      }
      const instance = database.instanceEntity;
      const instanceEngine = instance.engine;
      if (instanceEngine === Engine.MYSQL) {
        const schemaList: Schema[] =
          schemaEditorV1Store.resourceMap["database"].get(database.name)
            ?.schemaList || [];
        const schema = head(schemaList);
        if (!schema) {
          return;
        }
        const tableList: Table[] = schema.tableList;
        if (tableList.length === 0) {
          databaseTreeNode.isLeaf = true;
          databaseTreeNode.children = [];
        } else {
          databaseTreeNode.isLeaf = false;
          databaseTreeNode.children = tableList.map((table) => {
            return {
              type: "table",
              key: `t-${database.name}-${table.id}`,
              label: table.name,
              children: [],
              isLeaf: true,
              instance: instance.name,
              database: database.name,
              schemaId: schema.id,
              tableId: table.id,
            };
          });
        }
      } else if (instanceEngine === Engine.POSTGRES) {
        const schemaList: Schema[] =
          schemaEditorV1Store.resourceMap["database"].get(database.name)
            ?.schemaList || [];
        const schemaTreeNodeList: TreeNodeForSchema[] = [];
        for (const schema of schemaList) {
          const schemaTreeNode: TreeNodeForSchema = {
            type: "schema",
            key: `s-${database.name}-${schema.id}`,
            label: schema.name,
            instance: instance.name,
            database: database.name,
            schemaId: schema.id,
            isLeaf: true,
          };

          if (schema.tableList.length === 0) {
            schemaTreeNode.isLeaf = true;
            schemaTreeNode.children = [];
          } else {
            schemaTreeNode.isLeaf = false;
            schemaTreeNode.children = schema.tableList.map((table) => {
              return {
                type: "table",
                key: `t-${database.name}-${table.id}`,
                label: table.name,
                children: [],
                isLeaf: true,
                instance: instance.name,
                database: database.name,
                schemaId: schema.id,
                tableId: table.id,
              };
            });
          }
          schemaTreeNodeList.push(schemaTreeNode);
        }

        if (schemaTreeNodeList.length === 0) {
          databaseTreeNode.isLeaf = true;
          databaseTreeNode.children = [];
        } else {
          databaseTreeNode.isLeaf = false;
          databaseTreeNode.children = schemaTreeNodeList;
        }
      }
    }
  },
  {
    deep: true,
  }
);

watch(
  () => currentTab.value,
  () => {
    if (!currentTab.value) {
      selectedKeysRef.value = [];
      return;
    }

    if (currentTab.value.type === SchemaEditorTabType.TabForDatabase) {
      if (currentTab.value.selectedSchemaId) {
        const key = `s-${currentTab.value.parentName}-${currentTab.value.selectedSchemaId}`;
        selectedKeysRef.value = [key];
      } else {
        const key = `d-${currentTab.value.parentName}`;
        selectedKeysRef.value = [key];
      }
    } else if (currentTab.value.type === SchemaEditorTabType.TabForTable) {
      const databaseTreeNodeKey = `d-${currentTab.value.parentName}`;
      if (!expandedKeysRef.value.includes(databaseTreeNodeKey)) {
        expandedKeysRef.value.push(databaseTreeNodeKey);
      }
      const schemaTreeNodeKey = `s-${currentTab.value.parentName}-${currentTab.value.schemaId}`;
      if (!expandedKeysRef.value.includes(schemaTreeNodeKey)) {
        expandedKeysRef.value.push(schemaTreeNodeKey);
      }
      const tableTreeNodeKey = `t-${currentTab.value.parentName}-${currentTab.value.tableId}`;
      selectedKeysRef.value = [tableTreeNodeKey];
    }

    if (state.shouldRelocateTreeNode) {
      nextTick(() => {
        treeRef.value?.scrollTo({
          key: selectedKeysRef.value[0],
        });
      });
    }
  }
);

watch(
  () => searchPattern.value,
  () => {
    for (const treeNode of treeDataRef.value) {
      if (treeNode.type === "instance" && treeNode.children) {
        for (const databaseTreeNode of treeNode.children) {
          if (databaseTreeNode.children === undefined) {
            loadSubTree(databaseTreeNode);
          }
        }
      }
    }
  }
);

// Render prefix icons before label text.
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;

  if (treeNode.type === "instance") {
    const instance = instanceStore.getInstanceByName(treeNode.instance);
    const children = [
      h(InstanceV1EngineIcon, {
        instance,
      }),
      h(
        "span",
        {
          class: "text-gray-500 text-sm",
        },
        `(${instance.environmentEntity.title})`
      ),
    ];

    return h("span", { class: "flex items-center gap-x-1" }, children);
  } else if (treeNode.type === "database") {
    return h(DatabaseIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  } else if (treeNode.type === "schema") {
    return h(SchemaIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  } else if (treeNode.type === "table") {
    return h(TableIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  }
  return null;
};

// Render label text.
const renderLabel = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  const additionalClassList: string[] = ["select-none"];

  if (treeNode.type === "schema") {
    const schema = schemaEditorV1Store.resourceMap["database"]
      .get(treeNode.database)
      ?.schemaList.find((schema) => schema.id === treeNode.schemaId);

    if (schema) {
      if (schema.status === "created") {
        additionalClassList.push("text-green-700");
      } else if (schema.status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else {
        if (isSchemaChanged(treeNode.database, treeNode.schemaId)) {
          additionalClassList.push("text-yellow-700");
        }
      }
    }
  } else if (treeNode.type === "table") {
    const table = schemaEditorV1Store.resourceMap["database"]
      .get(treeNode.database)
      ?.schemaList.find((schema) => schema.id === treeNode.schemaId)
      ?.tableList.find((table) => table.id === treeNode.tableId);

    if (table) {
      if (table.status === "created") {
        additionalClassList.push("text-green-700");
      } else if (table.status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else {
        if (
          isTableChanged(treeNode.database, treeNode.schemaId, treeNode.tableId)
        ) {
          additionalClassList.push("text-yellow-700");
        }
      }
    }
  }

  return h(
    NEllipsis,
    {
      class: additionalClassList.join(" "),
    },
    () => [
      h("span", {
        innerHTML: getHighlightHTMLByKeyWords(
          escape(treeNode.label),
          escape(searchPattern.value)
        ),
      }),
    ]
  );
};

// Render a 'menu' icon in the right of the node
const renderSuffix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  const icon = h(EllipsisIcon, {
    class: "w-4 h-auto text-gray-600",
    onClick: (e) => {
      handleShowDropdown(e, treeNode);
    },
  });
  const instanceEngine = instanceStore.getInstanceByName(
    treeNode.instance
  ).engine;
  if (treeNode.type === "database") {
    return icon;
  } else if (treeNode.type === "schema") {
    if (instanceEngine === Engine.POSTGRES) {
      return icon;
    }
  } else if (treeNode.type === "table") {
    return icon;
  }
  return null;
};

const handleShowDropdown = (e: MouseEvent, treeNode: TreeNode) => {
  e.preventDefault();
  e.stopPropagation();
  contextMenu.treeNode = treeNode;
  contextMenu.showDropdown = true;
  contextMenu.clientX = e.clientX;
  contextMenu.clientY = e.clientY;
  selectedKeysRef.value = [treeNode.key];
};

// Dynamic fetching table list when database tree node clicking.
const loadSubTree = async (option: TreeOption) => {
  const treeNode = option as TreeNode;
  if (treeNode.type === "database") {
    const databaseName = treeNode.database;
    if (databaseDataLoadedSet.value.has(databaseName)) {
      return;
    }

    databaseDataLoadedSet.value.add(databaseName);
    try {
      const schemaList = await fetchSchemaListByDatabaseName({
        database: databaseName,
      });
      if (schemaList.length === 0) {
        treeNode.children = [];
        treeNode.isLeaf = true;
      }
    } catch (error) {
      useNotificationStore().pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("schema-editor.message.failed-to-fetch-database-schema"),
      });
    }
  }
};

// Set event handler to tree nodes.
const nodeProps = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  return {
    async onclick(e: MouseEvent) {
      // Check if clicked on the content part.
      // And ignore the fold/unfold arrow.
      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        state.shouldRelocateTreeNode = false;

        if (treeNode.type === "instance") {
          // Toggle instance tree node expanded status.
          const index = expandedKeysRef.value.findIndex(
            (key) => key === treeNode.key
          );
          if (index >= 0) {
            expandedKeysRef.value.splice(index, 1);
          } else {
            expandedKeysRef.value.push(treeNode.key);
          }
        } else if (treeNode.type === "database") {
          await loadSubTree(treeNode);
          const index = expandedKeysRef.value.findIndex(
            (key) => key === treeNode.key
          );
          if (index < 0) {
            expandedKeysRef.value.push(treeNode.key);
          }
          schemaEditorV1Store.addTab({
            id: generateUniqueTabId(),
            type: SchemaEditorTabType.TabForDatabase,
            parentName: treeNode.database,
            name: treeNode.label,
          });
        } else if (treeNode.type === "schema") {
          const index = expandedKeysRef.value.findIndex(
            (key) => key === treeNode.key
          );
          if (index < 0) {
            expandedKeysRef.value.push(treeNode.key);
          }
        } else if (treeNode.type === "table") {
          schemaEditorV1Store.addTab({
            id: generateUniqueTabId(),
            type: SchemaEditorTabType.TabForTable,
            parentName: treeNode.database,
            schemaId: treeNode.schemaId,
            tableId: treeNode.tableId,
            name: treeNode.label,
          });
        }

        nextTick(() => {
          if (treeNode.type === "database") {
            selectedKeysRef.value = [`d-${treeNode.database}`];
          } else if (treeNode.type === "schema") {
            selectedKeysRef.value = [
              `s-${treeNode.database}-${treeNode.schemaId}`,
            ];
          } else if (treeNode.type === "table") {
            selectedKeysRef.value = [
              `t-${treeNode.database}-${treeNode.tableId}`,
            ];
          }
          state.shouldRelocateTreeNode = true;
        });
      } else {
        nextTick(() => {
          selectedKeysRef.value = [];
        });
      }
    },
    async ondblclick() {
      await loadSubTree(treeNode);
      nextTick(() => {
        const index = expandedKeysRef.value.findIndex(
          (key) => key === treeNode.key
        );
        if (index >= 0) {
          expandedKeysRef.value.splice(index, 1);
        } else {
          expandedKeysRef.value.push(treeNode.key);
        }
      });
    },
    oncontextmenu(e: MouseEvent) {
      handleShowDropdown(e, treeNode);
    },
  };
};

const handleContextMenuDropdownSelect = async (key: string) => {
  const treeNode = contextMenu.treeNode;
  if (treeNode?.type === "database") {
    if (key === "create-table") {
      await loadSubTree(treeNode);
      const instanceEngine = instanceStore.getInstanceByName(
        treeNode.instance
      ).engine;
      if (instanceEngine === Engine.MYSQL) {
        const schemaList: Schema[] =
          schemaEditorV1Store.resourceMap["database"].get(treeNode.database)
            ?.schemaList || [];
        const schema = head(schemaList);
        if (!schema) {
          return;
        }
        state.tableNameModalContext = {
          parentName: treeNode.database,
          schemaId: schema.id,
          tableId: undefined,
        };
      }
    } else if (key === "create-schema") {
      state.schemaNameModalContext = {
        parentName: treeNode.database,
        schemaId: undefined,
      };
    }
  } else if (treeNode?.type === "schema") {
    if (key === "create-table") {
      await loadSubTree(treeNode);
      state.tableNameModalContext = {
        parentName: treeNode.database,
        schemaId: treeNode.schemaId,
        tableId: undefined,
      };
    } else if (key === "rename") {
      state.schemaNameModalContext = {
        parentName: treeNode.database,
        schemaId: treeNode.schemaId,
      };
    } else if (key === "drop-schema") {
      schemaEditorV1Store.dropSchema(treeNode.database, treeNode.schemaId);
    } else if (key === "restore") {
      schemaEditorV1Store.restoreSchema(treeNode.database, treeNode.schemaId);
    }
  } else if (treeNode?.type === "table") {
    if (key === "rename") {
      state.tableNameModalContext = {
        parentName: treeNode.database,
        schemaId: treeNode.schemaId,
        tableId: treeNode.tableId,
      };
    } else if (key === "drop") {
      schemaEditorV1Store.dropTable(
        treeNode.database,
        treeNode.schemaId,
        treeNode.tableId
      );
    } else if (key === "restore") {
      schemaEditorV1Store.restoreTable(
        treeNode.database,
        treeNode.schemaId,
        treeNode.tableId
      );
    }
  }
  contextMenu.showDropdown = false;
};

const handleDropdownClickoutside = (e: MouseEvent) => {
  if (
    !isDescendantOf(e.target as Element, ".n-tree-node-wrapper") ||
    e.button !== 2
  ) {
    selectedKeysRef.value = [];
    contextMenu.showDropdown = false;
  }
};

const handleExpandedKeysChange = (expandedKeys: string[]) => {
  expandedKeysRef.value = expandedKeys;
};
</script>

<style lang="postcss" scoped>
.schema-editor-database-tree :deep(.n-tree-node-wrapper) {
  @apply !p-0;
}
.schema-editor-database-tree :deep(.n-tree-node-content) {
  @apply !pl-2 text-sm;
}
.schema-editor-database-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.schema-editor-database-tree :deep(.n-tree-node-content__prefix) {
  @apply shrink-0 !mr-1;
}
.schema-editor-database-tree
  :deep(.n-tree-node-wrapper:hover .n-tree-node-content__suffix) {
  @apply !flex;
}
.schema-editor-database-tree
  :deep(.n-tree-node-wrapper
    .n-tree-node--selected
    .n-tree-node-content__suffix) {
  @apply !flex;
}
.schema-editor-database-tree :deep(.n-tree-node-switcher) {
  @apply px-0 !w-4 !h-7;
}
</style>
