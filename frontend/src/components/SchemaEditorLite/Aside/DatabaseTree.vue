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

  <!-- <SchemaNameModal
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
  /> -->
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { escape, head } from "lodash-es";
import { TreeOption, NEllipsis, NInput, NDropdown, NTree } from "naive-ui";
import { computed, onMounted, watch, ref, h, reactive, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import TableIcon from "~icons/heroicons-outline/table-cells";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import EllipsisIcon from "~icons/heroicons-solid/ellipsis-horizontal";
import DatabaseChangeHistoryPanel from "@/components/DatabaseChangeHistoryPanel.vue";
import { databaseForTask } from "@/components/IssueV1";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useInstanceV1Store } from "@/store";
import { ComposedDatabase, ComposedInstance } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { getHighlightHTMLByKeyWords, groupBy, isDescendantOf } from "@/utils";
import SchemaNameModal from "../Modals/SchemaNameModal.vue";
import TableNameModal from "../Modals/TableNameModal.vue";
import { useSchemaEditorContext } from "../context";
import { keyForResource, keyForResourceName } from "../context/common";

interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

interface TreeNodeForInstance extends BaseTreeNode {
  type: "instance";
  instance: ComposedInstance;
  children: TreeNodeForDatabase[];
}

interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  parent: TreeNodeForInstance;
  instance: ComposedInstance;
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
  };
  children: TreeNodeForSchema[];
}

interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  parent: TreeNodeForDatabase;
  instance: ComposedInstance;
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
  children: TreeNodeForTable[];
}

interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  parent: TreeNodeForSchema;
  instance: ComposedInstance;
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  };
  isLeaf: true;
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
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema?: SchemaMetadata;
  };
  tableNameModalContext?: {
    db: ComposedDatabase;
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table?: TableMetadata;
  };
}

const { t } = useI18n();
const {
  targets,
  readonly,
  currentTab,
  addTab,
  markEditStatus,
  removeEditStatus,
  getSchemaStatus,
  getTableStatus,
  upsertTableConfig,
} = useSchemaEditorContext();
// const schemaEditorV1Store = useSchemaEditorV1Store();
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
const treeNodeMap = new Map<string, TreeNode>();

// const databaseList = computed(() => {
//   return targets.value.map((target) => target.database);
// });
// const schemaList = computed(() =>
//   Array.from(schemaEditorV1Store.resourceMap["database"].values())
//     .map((item) => item.schemaList)
//     .flat()
//     .map((schema) => {
//       return {
//         ...schema,
//         tableList: schema.tableList.map((table) => {
//           // Don't watch column changes in database tree.
//           return pick(table, ["id", "name", "status"]);
//         }),
//       };
//     })
// );
const contextMenuOptions = computed(() => {
  // const treeNode = contextMenu.treeNode;
  // if (isUndefined(treeNode)) {
  //   return [];
  // }
  // const instanceEngine = instanceStore.getInstanceByName(
  //   treeNode.instance
  // ).engine;

  // if (treeNode.type === "database") {
  //   const options = [];
  //   if (instanceEngine === Engine.MYSQL) {
  //     options.push({
  //       key: "create-table",
  //       label: t("schema-editor.actions.create-table"),
  //     });
  //   } else if (instanceEngine === Engine.POSTGRES) {
  //     options.push({
  //       key: "create-schema",
  //       label: t("schema-editor.actions.create-schema"),
  //     });
  //   }
  //   return options;
  // } else if (treeNode.type === "schema") {
  //   const options = [];
  //   if (instanceEngine === Engine.POSTGRES) {
  //     const schema = schemaEditorV1Store.getSchema(
  //       treeNode.database,
  //       treeNode.schemaId
  //     );
  //     if (!schema) {
  //       return [];
  //     }

  //     const isDropped = schema.status === "dropped";
  //     if (isDropped) {
  //       options.push({
  //         key: "restore",
  //         label: t("schema-editor.actions.restore"),
  //       });
  //     } else {
  //       options.push({
  //         key: "create-table",
  //         label: t("schema-editor.actions.create-table"),
  //       });
  //       options.push({
  //         key: "rename",
  //         label: t("schema-editor.actions.rename"),
  //       });
  //       options.push({
  //         key: "drop-schema",
  //         label: t("schema-editor.actions.drop-schema"),
  //       });
  //     }
  //   }
  //   return options;
  // } else if (treeNode.type === "table") {
  //   const table = schemaEditorV1Store.getTable(
  //     treeNode.database,
  //     treeNode.schemaId,
  //     treeNode.tableId
  //   );
  //   if (!table) {
  //     return [];
  //   }

  //   const isDropped = table.status === "dropped";
  //   const options = [];
  //   if (isDropped) {
  //     options.push({
  //       key: "restore",
  //       label: t("schema-editor.actions.restore"),
  //     });
  //   } else {
  //     options.push({
  //       key: "rename",
  //       label: t("schema-editor.actions.rename"),
  //     });
  //     options.push({
  //       key: "drop",
  //       label: t("schema-editor.actions.drop-table"),
  //     });
  //   }
  //   return options;
  // }

  return [];
});

const upsertExpandedKeys = (key: string) => {
  if (expandedKeysRef.value.includes(key)) return;
  expandedKeysRef.value.push(key);
};
const expandNodeRecursively = (node: TreeNode) => {
  if (node.type === "table") {
    // table nodes are not expandable
    expandNodeRecursively(node.parent);
  }
  if (node.type === "schema") {
    const key = keyForResource(node.database, node.metadata);
    upsertExpandedKeys(key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "database") {
    const key = node.database.name;
    upsertExpandedKeys(key);
    expandNodeRecursively(node.parent);
  }
  if (node.type === "instance") {
    const key = node.instance.name;
    upsertExpandedKeys(key);
  }
};
const buildDatabaseTreeData = () => {
  const groupedByInstance = groupBy(
    targets.value,
    (target) => target.database.instance
  );
  treeNodeMap.clear();
  const treeNodeList = Array.from(groupedByInstance).map(([_, targets]) => {
    const instance = targets[0].database.instanceEntity;
    const instanceNode: TreeNodeForInstance = {
      type: "instance",
      key: instance.name,
      label: instance.title,
      isLeaf: false,
      instance,
      children: [],
    };
    instanceNode.children = targets.map((target) => {
      const { database } = target;
      const databaseNode: TreeNodeForDatabase = {
        type: "database",
        key: database.name,
        parent: instanceNode,
        label: database.databaseName,
        isLeaf: false,
        instance,
        database,
        metadata: {
          database: target.metadata,
        },
        children: [],
      };
      databaseNode.children = target.metadata.schemas.map((schema) => {
        const metadata = {
          database: target.metadata,
          schema,
        };
        const schemaNode: TreeNodeForSchema = {
          type: "schema",
          parent: databaseNode,
          key: keyForResource(database, metadata),
          label: schema.name,
          isLeaf: false,
          instance,
          database,
          metadata,
          children: [],
        };
        schemaNode.children = schema.tables.map((table) => {
          const metadata = {
            database: target.metadata,
            schema,
            table,
          };
          const tableNode: TreeNodeForTable = {
            type: "table",
            parent: schemaNode,
            key: keyForResource(database, metadata),
            label: table.name,
            isLeaf: true,
            instance,
            database,
            metadata,
          };
          treeNodeMap.set(tableNode.key, tableNode);
          return tableNode;
        });
        treeNodeMap.set(schemaNode.key, schemaNode);
        return schemaNode;
      });
      treeNodeMap.set(databaseNode.key, databaseNode);
      return databaseNode;
    });
    treeNodeMap.set(instanceNode.key, instanceNode);
    return instanceNode;
  });

  treeDataRef.value = treeNodeList;

  const firstInstanceNode = head(treeNodeList) as
    | TreeNodeForInstance
    | undefined;
  const firstDatabaseNode = head(firstInstanceNode?.children);
  const firstSchemaNode = head(firstDatabaseNode?.children);
  if (firstSchemaNode) {
    nextTick(() => {
      // Auto expand the first tree node.
      openTabForTreeNode(firstSchemaNode);
    });
  }
};
const tabWatchKey = computed(() => {
  const tab = currentTab.value;
  if (!tab) return undefined;
  if (tab.type === "database") {
    return keyForResourceName(tab.database.name, tab.selectedSchema);
  }
  return keyForResource(tab.database, tab.metadata);
});
// Sync tree expandedKeys and selectedKeys with tabs
watch(tabWatchKey, () => {
  const tab = currentTab.value;
  if (!tab) {
    selectedKeysRef.value = [];
    return;
  }

  if (tab.type === "database") {
    const { database, selectedSchema: schema } = tab;
    if (schema) {
      const key = keyForResourceName(database.name, schema);
      const node = treeNodeMap.get(key);
      if (node) {
        expandNodeRecursively(node);
      }
      selectedKeysRef.value = [key];
    }
  } else if (tab.type === "table") {
    const {
      database,
      metadata: { schema, table },
    } = tab;
    const schemaKey = keyForResource(database, { schema });
    const schemaNode = treeNodeMap.get(schemaKey);
    if (schemaNode) {
      expandNodeRecursively(schemaNode);
    }
    const tableKey = keyForResource(database, { schema, table });
    selectedKeysRef.value = [tableKey];
  }

  if (state.shouldRelocateTreeNode) {
    nextTick(() => {
      treeRef.value?.scrollTo({
        key: selectedKeysRef.value[0],
      });
    });
  }
});

const openTabForTreeNode = (node: TreeNode) => {
  state.shouldRelocateTreeNode = false;

  if (node.type === "table") {
    const { database, metadata } = node;
    expandNodeRecursively(node);
    addTab({
      type: "table",
      database,
      metadata,
    });
  } else if (node.type === "schema") {
    expandNodeRecursively(node);
    const { database, metadata } = node;
    addTab({
      type: "database",
      database,
      metadata: {
        database: metadata.database,
      },
      selectedSchema: metadata.schema.name,
    });
  }

  state.shouldRelocateTreeNode = true;
};

onMounted(async () => {
  buildDatabaseTreeData();
});

// watch(
//   [() => schemaList.value],
//   () => {
//     const databaseTreeNodeList: TreeNodeForDatabase[] = [];
//     for (const treeNode of treeDataRef.value) {
//       if (treeNode.type === "instance") {
//         databaseTreeNodeList.push(
//           ...(treeNode.children as TreeNodeForDatabase[])
//         );
//       }
//     }

//     for (const database of databaseList.value) {
//       if (!databaseDataLoadedSet.value.has(database.name)) {
//         continue;
//       }
//       const databaseTreeNode = databaseTreeNodeList.find(
//         (treeNode) =>
//           treeNode.database === database.name &&
//           databaseDataLoadedSet.value.has(treeNode.database)
//       );
//       if (isUndefined(databaseTreeNode)) {
//         continue;
//       }
//       const instance = database.instanceEntity;
//       const instanceEngine = instance.engine;
//       if (instanceEngine === Engine.MYSQL) {
//         const schemaList: Schema[] =
//           schemaEditorV1Store.resourceMap["database"].get(database.name)
//             ?.schemaList || [];
//         const schema = head(schemaList);
//         if (!schema) {
//           return;
//         }
//         const tableList: Table[] = schema.tableList;
//         if (tableList.length === 0) {
//           databaseTreeNode.isLeaf = true;
//           databaseTreeNode.children = [];
//         } else {
//           databaseTreeNode.isLeaf = false;
//           databaseTreeNode.children = tableList.map((table) => {
//             return {
//               type: "table",
//               key: `t-${database.name}-${table.id}`,
//               label: table.name,
//               children: [],
//               isLeaf: true,
//               instance: instance.name,
//               database: database.name,
//               schemaId: schema.id,
//               tableId: table.id,
//             };
//           });
//         }
//       } else if (instanceEngine === Engine.POSTGRES) {
//         const schemaList: Schema[] =
//           schemaEditorV1Store.resourceMap["database"].get(database.name)
//             ?.schemaList || [];
//         const schemaTreeNodeList: TreeNodeForSchema[] = [];
//         for (const schema of schemaList) {
//           const schemaTreeNode: TreeNodeForSchema = {
//             type: "schema",
//             key: `s-${database.name}-${schema.id}`,
//             label: schema.name,
//             instance: instance.name,
//             database: database.name,
//             schemaId: schema.id,
//             isLeaf: true,
//           };

//           if (schema.tableList.length === 0) {
//             schemaTreeNode.isLeaf = true;
//             schemaTreeNode.children = [];
//           } else {
//             schemaTreeNode.isLeaf = false;
//             schemaTreeNode.children = schema.tableList.map((table) => {
//               return {
//                 type: "table",
//                 key: `t-${database.name}-${table.id}`,
//                 label: table.name,
//                 children: [],
//                 isLeaf: true,
//                 instance: instance.name,
//                 database: database.name,
//                 schemaId: schema.id,
//                 tableId: table.id,
//               };
//             });
//           }
//           schemaTreeNodeList.push(schemaTreeNode);
//         }

//         if (schemaTreeNodeList.length === 0) {
//           databaseTreeNode.isLeaf = true;
//           databaseTreeNode.children = [];
//         } else {
//           databaseTreeNode.isLeaf = false;
//           databaseTreeNode.children = schemaTreeNodeList;
//         }
//       }
//     }
//   },
//   {
//     deep: true,
//   }
// );

// watch(
//   () => currentTab.value,
//   () => {
//     if (!currentTab.value) {
//       selectedKeysRef.value = [];
//       return;
//     }

//     if (currentTab.value.type === SchemaEditorTabType.TabForDatabase) {
//       if (currentTab.value.selectedSchemaId) {
//         const key = `s-${currentTab.value.parentName}-${currentTab.value.selectedSchemaId}`;
//         selectedKeysRef.value = [key];
//       } else {
//         const key = `d-${currentTab.value.parentName}`;
//         selectedKeysRef.value = [key];
//       }
//     } else if (currentTab.value.type === SchemaEditorTabType.TabForTable) {
//       const databaseTreeNodeKey = `d-${currentTab.value.parentName}`;
//       if (!expandedKeysRef.value.includes(databaseTreeNodeKey)) {
//         expandedKeysRef.value.push(databaseTreeNodeKey);
//       }
//       const schemaTreeNodeKey = `s-${currentTab.value.parentName}-${currentTab.value.schemaId}`;
//       if (!expandedKeysRef.value.includes(schemaTreeNodeKey)) {
//         expandedKeysRef.value.push(schemaTreeNodeKey);
//       }
//       const tableTreeNodeKey = `t-${currentTab.value.parentName}-${currentTab.value.tableId}`;
//       selectedKeysRef.value = [tableTreeNodeKey];
//     }

//     if (state.shouldRelocateTreeNode) {
//       nextTick(() => {
//         treeRef.value?.scrollTo({
//           key: selectedKeysRef.value[0],
//         });
//       });
//     }
//   }
// );

// Render prefix icons before label text.
const renderPrefix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;

  if (treeNode.type === "instance") {
    const { instance } = treeNode;
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
  let label = treeNode.label;

  if (treeNode.type === "schema") {
    const { database, metadata } = treeNode;
    additionalClassList.push(getSchemaStatus(database, metadata));

    if (database.instanceEntity.engine !== Engine.POSTGRES) {
      label = t("db.tables");
    }
  } else if (treeNode.type === "table") {
    const { database, metadata } = treeNode;
    additionalClassList.push(getTableStatus(database, metadata));
  }

  return h(
    NEllipsis,
    {
      class: additionalClassList.join(" "),
    },
    () => [
      h("span", {
        innerHTML: getHighlightHTMLByKeyWords(
          escape(label),
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
  const { engine } = treeNode.instance;
  if (treeNode.type === "database") {
    return icon;
  } else if (treeNode.type === "schema") {
    if (engine === Engine.POSTGRES) {
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

// Set event handler to tree nodes.
const nodeProps = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  return {
    async onClick(e: MouseEvent) {
      // // Check if clicked on the content part.
      // // And ignore the fold/unfold arrow.
      // if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
      //   state.shouldRelocateTreeNode = false;
      //   if (treeNode.type === "instance") {
      //     // Toggle instance tree node expanded status.
      //     const index = expandedKeysRef.value.findIndex(
      //       (key) => key === treeNode.key
      //     );
      //     if (index >= 0) {
      //       expandedKeysRef.value.splice(index, 1);
      //     } else {
      //       expandedKeysRef.value.push(treeNode.key);
      //     }
      //   } else if (treeNode.type === "database") {
      //     const index = expandedKeysRef.value.findIndex(
      //       (key) => key === treeNode.key
      //     );
      //     if (index < 0) {
      //       expandedKeysRef.value.push(treeNode.key);
      //     }
      //     schemaEditorV1Store.addTab({
      //       id: generateUniqueTabId(),
      //       type: SchemaEditorTabType.TabForDatabase,
      //       parentName: treeNode.database,
      //       name: treeNode.label,
      //     });
      //   } else if (treeNode.type === "schema") {
      //     const index = expandedKeysRef.value.findIndex(
      //       (key) => key === treeNode.key
      //     );
      //     if (index < 0) {
      //       expandedKeysRef.value.push(treeNode.key);
      //     }
      //   } else if (treeNode.type === "table") {
      //     schemaEditorV1Store.addTab({
      //       id: generateUniqueTabId(),
      //       type: SchemaEditorTabType.TabForTable,
      //       parentName: treeNode.database,
      //       schemaId: treeNode.schemaId,
      //       tableId: treeNode.tableId,
      //       name: treeNode.label,
      //     });
      //   }
      //   nextTick(() => {
      //     if (treeNode.type === "database") {
      //       selectedKeysRef.value = [`d-${treeNode.database}`];
      //     } else if (treeNode.type === "schema") {
      //       selectedKeysRef.value = [
      //         `s-${treeNode.database}-${treeNode.schemaId}`,
      //       ];
      //     } else if (treeNode.type === "table") {
      //       selectedKeysRef.value = [
      //         `t-${treeNode.database}-${treeNode.tableId}`,
      //       ];
      //     }
      //     state.shouldRelocateTreeNode = true;
      //   });
      // } else {
      //   nextTick(() => {
      //     selectedKeysRef.value = [];
      //   });
      // }
    },
    ondblclick() {
      // await loadSubTree(treeNode);
      // nextTick(() => {
      //   const index = expandedKeysRef.value.findIndex(
      //     (key) => key === treeNode.key
      //   );
      //   if (index >= 0) {
      //     expandedKeysRef.value.splice(index, 1);
      //   } else {
      //     expandedKeysRef.value.push(treeNode.key);
      //   }
      // });
    },
    oncontextmenu(e: MouseEvent) {
      handleShowDropdown(e, treeNode);
    },
  };
};

const handleContextMenuDropdownSelect = async (key: string) => {
  // const treeNode = contextMenu.treeNode;
  // if (treeNode?.type === "database") {
  //   if (key === "create-table") {
  //     await loadSubTree(treeNode);
  //     const instanceEngine = instanceStore.getInstanceByName(
  //       treeNode.instance
  //     ).engine;
  //     if (instanceEngine === Engine.MYSQL) {
  //       const schemaList: Schema[] =
  //         schemaEditorV1Store.resourceMap["database"].get(treeNode.database)
  //           ?.schemaList || [];
  //       const schema = head(schemaList);
  //       if (!schema) {
  //         return;
  //       }
  //       state.tableNameModalContext = {
  //         parentName: treeNode.database,
  //         schemaId: schema.id,
  //         tableId: undefined,
  //       };
  //     }
  //   } else if (key === "create-schema") {
  //     state.schemaNameModalContext = {
  //       parentName: treeNode.database,
  //       schemaId: undefined,
  //     };
  //   }
  // } else if (treeNode?.type === "schema") {
  //   if (key === "create-table") {
  //     await loadSubTree(treeNode);
  //     state.tableNameModalContext = {
  //       parentName: treeNode.database,
  //       schemaId: treeNode.schemaId,
  //       tableId: undefined,
  //     };
  //   } else if (key === "rename") {
  //     state.schemaNameModalContext = {
  //       parentName: treeNode.database,
  //       schemaId: treeNode.schemaId,
  //     };
  //   } else if (key === "drop-schema") {
  //     schemaEditorV1Store.dropSchema(treeNode.database, treeNode.schemaId);
  //   } else if (key === "restore") {
  //     schemaEditorV1Store.restoreSchema(treeNode.database, treeNode.schemaId);
  //   }
  // } else if (treeNode?.type === "table") {
  //   if (key === "rename") {
  //     state.tableNameModalContext = {
  //       parentName: treeNode.database,
  //       schemaId: treeNode.schemaId,
  //       tableId: treeNode.tableId,
  //     };
  //   } else if (key === "drop") {
  //     schemaEditorV1Store.dropTable(
  //       treeNode.database,
  //       treeNode.schemaId,
  //       treeNode.tableId
  //     );
  //   } else if (key === "restore") {
  //     schemaEditorV1Store.restoreTable(
  //       treeNode.database,
  //       treeNode.schemaId,
  //       treeNode.tableId
  //     );
  //   }
  // }
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
