<template>
  <div class="w-full h-full pl-1 pr-2 relative overflow-y-hidden">
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
    <div ref="treeRef" class="schema-editor-database-tree pb-2 h-auto">
      <NTree
        block-line
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
        :on-update:selected-keys="handleSelectedKeysChange"
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
    :database-id="state.schemaNameModalContext.databaseId"
    :schema-id="state.schemaNameModalContext.schemaId"
    @close="state.schemaNameModalContext = undefined"
  />

  <TableNameModal
    v-if="state.tableNameModalContext !== undefined"
    :database-id="state.tableNameModalContext.databaseId"
    :schema-id="state.tableNameModalContext.schemaId"
    :table-id="state.tableNameModalContext.tableId"
    @close="state.tableNameModalContext = undefined"
  />
</template>

<script lang="ts" setup>
import { escape, head, isUndefined } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { computed, onMounted, watch, ref, h, reactive, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import scrollIntoView from "scroll-into-view-if-needed";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import TableIcon from "~icons/heroicons-outline/table-cells";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";

import { SchemaEditorTabType } from "@/types";
import { Schema, Table } from "@/types/schemaEditor/atomType";
import {
  useSchemaEditorStore,
  generateUniqueTabId,
  useInstanceV1Store,
  useNotificationStore,
} from "@/store";
import { getHighlightHTMLByKeyWords, isDescendantOf } from "@/utils";
import { isSchemaChanged } from "./utils/schema";
import { isTableChanged } from "./utils/table";
import SchemaNameModal from "./Modals/SchemaNameModal.vue";
import TableNameModal from "./Modals/TableNameModal.vue";
import { Engine } from "@/types/proto/v1/common";
import { InstanceV1EngineIcon } from "@/components/v2";

interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

interface TreeNodeForInstance extends BaseTreeNode {
  type: "instance";
  instanceId: string;
}

interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  instanceId: string;
  databaseId: string;
}

interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  instanceId: string;
  databaseId: string;
  schemaId: string;
}

interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  instanceId: string;
  databaseId: string;
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
    databaseId: string;
    schemaId: string | undefined;
  };
  tableNameModalContext?: {
    databaseId: string;
    schemaId: string;
    tableId: string | undefined;
  };
}

const { t } = useI18n();
const editorStore = useSchemaEditorStore();
const instanceStore = useInstanceV1Store();
const state = reactive<LocalState>({
  shouldRelocateTreeNode: false,
});
const contextMenu = reactive<TreeContextMenu>({
  showDropdown: false,
  clientX: 0,
  clientY: 0,
  treeNode: undefined,
});
const treeRef = ref();
const searchPattern = ref("");
const expandedKeysRef = ref<string[]>([]);
const selectedKeysRef = ref<string[]>([]);
const treeDataRef = ref<TreeNode[]>([]);
const databaseDataLoadedSet = ref<Set<string>>(new Set());

const databaseList = computed(() => editorStore.databaseList);
const schemaList = computed(() =>
  Array.from(editorStore.databaseSchemaById.values())
    .map((databaseState) => databaseState.schemaList)
    .flat()
);
const tableList = computed(() =>
  schemaList.value.map((schema) => schema.tableList).flat()
);
const contextMenuOptions = computed(() => {
  const treeNode = contextMenu.treeNode;
  if (isUndefined(treeNode)) {
    return [];
  }
  const instanceEngine = instanceStore.getInstanceByUID(
    treeNode.instanceId
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
      const schema = editorStore.getSchema(
        treeNode.databaseId,
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
    const table = editorStore.databaseSchemaById
      .get(treeNode.databaseId)
      ?.schemaList.find((schema) => schema.id === treeNode.schemaId)
      ?.tableList.find((table) => table.id === treeNode.tableId);
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
    if (instanceTreeNodeMap.has(instance.uid)) {
      instanceTreeNode = instanceTreeNodeMap.get(
        instance.uid
      ) as TreeNodeForInstance;
    } else {
      instanceTreeNode = {
        type: "instance",
        key: `i-${instance.uid}`,
        label: instance.title,
        isLeaf: false,
        instanceId: instance.uid,
        children: [],
      };
      instanceTreeNodeMap.set(instance.uid, instanceTreeNode);
      treeNodeList.push(instanceTreeNode);
      // Make the instance tree node expanded as default.
      expandedKeysRef.value.push(instanceTreeNode.key);
    }

    const databaseTreeNode: TreeNodeForDatabase = {
      type: "database",
      key: `d-${database.uid}`,
      label: database.databaseName,
      isLeaf: false,
      instanceId: instance.uid,
      databaseId: database.uid,
    };
    instanceTreeNode.children?.push(databaseTreeNode);
    databaseTreeNodeList.push(databaseTreeNode);
  }
  treeDataRef.value = treeNodeList;
});

watch(
  [
    () => schemaList.value,
    () => tableList.value,
    () => databaseDataLoadedSet.value,
  ],
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
      if (!databaseDataLoadedSet.value.has(database.uid)) {
        continue;
      }
      const databaseTreeNode = databaseTreeNodeList.find(
        (treeNode) =>
          treeNode.databaseId === database.uid &&
          databaseDataLoadedSet.value.has(treeNode.databaseId)
      );
      if (isUndefined(databaseTreeNode)) {
        continue;
      }
      const instance = database.instanceEntity;
      const instanceEngine = instance.engine;
      if (instanceEngine === Engine.MYSQL) {
        const schemaList: Schema[] =
          editorStore.databaseSchemaById.get(database.uid)?.schemaList || [];
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
              key: `t-${database.uid}-${table.id}`,
              label: table.name,
              children: [],
              isLeaf: true,
              instanceId: instance.uid,
              databaseId: database.uid,
              schemaId: schema.id,
              tableId: table.id,
            };
          });
        }
      } else if (instanceEngine === Engine.POSTGRES) {
        const schemaList: Schema[] =
          editorStore.databaseSchemaById.get(database.uid)?.schemaList || [];
        const schemaTreeNodeList: TreeNodeForSchema[] = [];
        for (const schema of schemaList) {
          const schemaTreeNode: TreeNodeForSchema = {
            type: "schema",
            key: `s-${database.uid}-${schema.id}`,
            label: schema.name,
            instanceId: instance.uid,
            databaseId: database.uid,
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
                key: `t-${database.uid}-${table.id}`,
                label: table.name,
                children: [],
                isLeaf: true,
                instanceId: instance.uid,
                databaseId: database.uid,
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
  () => editorStore.currentTab,
  () => {
    const currentTab = editorStore.currentTab;
    if (!currentTab) {
      selectedKeysRef.value = [];
      return;
    }

    if (currentTab.type === SchemaEditorTabType.TabForDatabase) {
      if (currentTab.selectedSchemaId) {
        const key = `s-${currentTab.databaseId}-${currentTab.selectedSchemaId}`;
        selectedKeysRef.value = [key];
      } else {
        const key = `d-${currentTab.databaseId}`;
        selectedKeysRef.value = [key];
      }
    } else if (currentTab.type === SchemaEditorTabType.TabForTable) {
      const databaseTreeNodeKey = `d-${currentTab.databaseId}`;
      if (!expandedKeysRef.value.includes(databaseTreeNodeKey)) {
        expandedKeysRef.value.push(databaseTreeNodeKey);
      }
      const schemaTreeNodeKey = `s-${currentTab.databaseId}-${currentTab.schemaId}`;
      if (!expandedKeysRef.value.includes(schemaTreeNodeKey)) {
        expandedKeysRef.value.push(schemaTreeNodeKey);
      }
      const tableTreeNodeKey = `t-${currentTab.databaseId}-${currentTab.tableId}`;
      selectedKeysRef.value = [tableTreeNodeKey];
    }

    if (state.shouldRelocateTreeNode) {
      nextTick(() => {
        const element = treeRef.value?.querySelector(".n-tree-node--selected");
        if (element) {
          scrollIntoView(element, {
            scrollMode: "if-needed",
          });
        }
      });
    }
  }
);

watch(searchPattern, () => {
  for (const treeNode of treeDataRef.value) {
    if (treeNode.type === "instance" && treeNode.children) {
      for (const databaseTreeNode of treeNode.children) {
        if (databaseTreeNode.children === undefined) {
          loadSubTree(databaseTreeNode);
        }
      }
    }
  }
});

// Render prefix icons before label text.
const renderPrefix = ({ option: treeNode }: { option: TreeNode }) => {
  if (treeNode.type === "instance") {
    const instance = instanceStore.getInstanceByUID(treeNode.instanceId);
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
const renderLabel = ({ option: treeNode }: { option: TreeNode }) => {
  const additionalClassList: string[] = ["select-none"];

  if (treeNode.type === "schema") {
    const schema = editorStore.databaseSchemaById
      .get(treeNode.databaseId)
      ?.schemaList.find((schema) => schema.id === treeNode.schemaId);

    if (schema) {
      if (schema.status === "created") {
        additionalClassList.push("text-green-700");
      } else if (schema.status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else {
        if (isSchemaChanged(treeNode.databaseId, treeNode.schemaId)) {
          additionalClassList.push("text-yellow-700");
        }
      }
    }
  } else if (treeNode.type === "table") {
    const table = editorStore.databaseSchemaById
      .get(treeNode.databaseId)
      ?.schemaList.find((schema) => schema.id === treeNode.schemaId)
      ?.tableList.find((table) => table.id === treeNode.tableId);

    if (table) {
      if (table.status === "created") {
        additionalClassList.push("text-green-700");
      } else if (table.status === "dropped") {
        additionalClassList.push("text-red-700 line-through");
      } else {
        if (
          isTableChanged(
            treeNode.databaseId,
            treeNode.schemaId,
            treeNode.tableId
          )
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
const renderSuffix = ({ option: treeNode }: { option: TreeNode }) => {
  const icon = h(EllipsisIcon, {
    class: "w-4 h-auto text-gray-600",
    onClick: (e) => {
      handleShowDropdown(e, treeNode);
    },
  });
  const instanceEngine = instanceStore.getInstanceByUID(
    treeNode.instanceId
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
const loadSubTree = async (treeNode: TreeNode) => {
  if (treeNode.type === "database") {
    const databaseId = treeNode.databaseId;
    if (databaseDataLoadedSet.value.has(databaseId)) {
      return;
    }

    databaseDataLoadedSet.value.add(databaseId);
    try {
      const schemaList = await editorStore.fetchSchemaListByDatabaseId(
        databaseId,
        true
      );
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
const nodeProps = ({ option: treeNode }: { option: TreeNode }) => {
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
          editorStore.addTab({
            id: generateUniqueTabId(),
            type: SchemaEditorTabType.TabForDatabase,
            databaseId: treeNode.databaseId,
          });
        } else if (treeNode.type === "schema") {
          const index = expandedKeysRef.value.findIndex(
            (key) => key === treeNode.key
          );
          if (index < 0) {
            expandedKeysRef.value.push(treeNode.key);
          }
          editorStore.addTab({
            id: generateUniqueTabId(),
            type: SchemaEditorTabType.TabForDatabase,
            databaseId: treeNode.databaseId,
            selectedSchemaId: treeNode.schemaId,
          });
        } else if (treeNode.type === "table") {
          editorStore.addTab({
            id: generateUniqueTabId(),
            type: SchemaEditorTabType.TabForTable,
            databaseId: treeNode.databaseId,
            schemaId: treeNode.schemaId,
            tableId: treeNode.tableId,
          });
        }

        nextTick(() => {
          if (treeNode.type === "database") {
            selectedKeysRef.value = [`d-${treeNode.databaseId}`];
          } else if (treeNode.type === "schema") {
            selectedKeysRef.value = [
              `s-${treeNode.databaseId}-${treeNode.schemaId}`,
            ];
          } else if (treeNode.type === "table") {
            selectedKeysRef.value = [
              `t-${treeNode.databaseId}-${treeNode.tableId}`,
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
      const instanceEngine = instanceStore.getInstanceByUID(
        treeNode.instanceId
      ).engine;
      if (instanceEngine === Engine.MYSQL) {
        const schemaList: Schema[] =
          editorStore.databaseSchemaById.get(treeNode.databaseId)?.schemaList ||
          [];
        const schema = head(schemaList);
        if (!schema) {
          return;
        }
        state.tableNameModalContext = {
          databaseId: treeNode.databaseId,
          schemaId: schema.id,
          tableId: undefined,
        };
      }
    } else if (key === "create-schema") {
      state.schemaNameModalContext = {
        databaseId: treeNode.databaseId,
        schemaId: undefined,
      };
    }
  } else if (treeNode?.type === "schema") {
    if (key === "create-table") {
      await loadSubTree(treeNode);
      state.tableNameModalContext = {
        databaseId: treeNode.databaseId,
        schemaId: treeNode.schemaId,
        tableId: undefined,
      };
    } else if (key === "rename") {
      state.schemaNameModalContext = {
        databaseId: treeNode.databaseId,
        schemaId: treeNode.schemaId,
      };
    } else if (key === "drop-schema") {
      editorStore.dropSchema(treeNode.databaseId, treeNode.schemaId);
    } else if (key === "restore") {
      editorStore.restoreSchema(treeNode.databaseId, treeNode.schemaId);
    }
  } else if (treeNode?.type === "table") {
    if (key === "rename") {
      state.tableNameModalContext = {
        databaseId: treeNode.databaseId,
        schemaId: treeNode.schemaId,
        tableId: treeNode.tableId,
      };
    } else if (key === "drop") {
      editorStore.dropTable(
        treeNode.databaseId,
        treeNode.schemaId,
        treeNode.tableId
      );
    } else if (key === "restore") {
      editorStore.restoreTable(
        treeNode.databaseId,
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

const handleSelectedKeysChange = (selectedKeys: string[]) => {
  selectedKeysRef.value = selectedKeys;
};
</script>

<style>
.schema-editor-database-tree .n-tree-node-wrapper {
  @apply !py-px;
}
.schema-editor-database-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.schema-editor-database-tree .n-tree-node-content__text {
  @apply truncate mr-1;
}
.schema-editor-database-tree .n-tree-node-content__suffix {
  @apply rounded-sm !hidden hover:opacity-80;
}
.schema-editor-database-tree
  .n-tree-node-wrapper:hover
  .n-tree-node-content__suffix {
  @apply !flex;
}
.schema-editor-database-tree
  .n-tree-node-wrapper
  .n-tree-node--selected
  .n-tree-node-content__suffix {
  @apply !flex;
}
.schema-editor-database-tree .n-tree-node-switcher {
  @apply px-0 !w-4 !h-7;
}
</style>

<style scoped>
.schema-editor-database-tree {
  @apply overflow-y-auto;
  max-height: calc(100% - 48px);
}
</style>
