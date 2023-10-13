<template>
  <div
    v-if="treeStore.state === 'READY'"
    class="databases-tree p-0.5 gap-y-1 h-full flex flex-col"
  >
    <div class="databases-tree--input pt-1 px-2">
      <NInput
        size="small"
        :value="searchPattern"
        :placeholder="$t('sql-editor.search-databases')"
        :clearable="true"
        @update:value="$emit('update:search-pattern', $event)"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
    </div>
    <div class="databases-tree--tree flex-1 overflow-y-auto select-none">
      <NTree
        ref="treeRef"
        block-line
        :data="treeStore.tree"
        :show-irrelevant-nodes="false"
        :expand-on-click="true"
        :node-props="nodeProps"
        :virtual-scroll="true"
        debugger
        @load="handleLoadSubTree"
      />
    </div>

    <NDropdown
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
  <div v-else class="flex justify-center items-center h-full">
    <BBSpin :title="$t('sql-editor.loading-databases')" />
  </div>
</template>

<script lang="ts" setup>
import { useMounted, useThrottleFn } from "@vueuse/core";
import { NTree, NInput, NDropdown, DropdownOption, TreeOption } from "naive-ui";
import { ref, computed, nextTick, watch } from "vue";
import { useSQLEditorTreeStore } from "@/store/modules/sqlEditorTree";
import type { SQLEditorTreeNode } from "@/types";
import { TabMode } from "@/types";
import { isDescendantOf } from "@/utils";
import { fetchDatabaseSubTree } from "./common-new";

type Position = {
  x: number;
  y: number;
};

type DropdownOptionWithTreeNode = DropdownOption & {
  item: SQLEditorTreeNode;
};

const props = defineProps<{
  searchPattern?: string;
}>();

defineEmits<{
  (event: "update:search-pattern", keyword: string): void;
}>();

// const { t } = useI18n();

const treeStore = useSQLEditorTreeStore();
// const tabStore = useTabStore();
// const isLoggedIn = useIsLoggedIn();
// const currentUserV1 = useCurrentUserV1();
// const sqlEditorStore = useSQLEditorStore();
// const { selectedDatabaseSchemaByDatabaseName, events: editorEvents } =
//   useSQLEditorContext();

const mounted = useMounted();
const treeRef = ref<InstanceType<typeof NTree>>();
const throttledSearchPattern = ref(props.searchPattern);
const showDropdown = ref(false);
const dropdownPosition = ref<Position>({
  x: 0,
  y: 0,
});
const dropdownContext = ref<SQLEditorTreeNode>();

const dropdownOptions = computed((): DropdownOptionWithTreeNode[] => {
  return [];
  // const atom = dropdownContext.value;
  // if (!atom) {
  //   return [];
  // }
  // if (atom.type === "project") {
  //   return [];
  // } else {
  //   // Don't show any context menu actions for disabled
  //   // instances/databases
  //   if (atom.disabled) {
  //     return [];
  //   }

  //   const items: DropdownOptionWithConnectionAtom[] = [];
  //   if (isConnectableAtom(atom)) {
  //     const instance = instanceOfConnectionAtom(atom);
  //     if (instance && instanceV1HasReadonlyMode(instance)) {
  //       items.push({
  //         key: "connect",
  //         label: t("sql-editor.connect"),
  //         item: atom,
  //       });
  //     }
  //     if (allowAdmin.value) {
  //       items.push({
  //         key: "connect-in-admin-mode",
  //         label: t("sql-editor.connect-in-admin-mode"),
  //         item: atom,
  //       });
  //     }
  //   }
  //   if (atom.type === "database" && sqlEditorStore.mode === "BUNDLED") {
  //     const database = databaseStore.getDatabaseByUID(atom.id);
  //     if (instanceV1HasAlterSchema(database.instanceEntity)) {
  //       items.push({
  //         key: "alter-schema",
  //         label: t("database.edit-schema"),
  //         item: atom,
  //       });
  //     }
  //   }
  //   return items;
  // }
});

// Highlight the current tab's connection node.
const selectedKeys = computed(() => {
  return [];
  // const { instanceId, databaseId } = tabStore.currentTab.connection;

  // if (databaseId !== String(UNKNOWN_ID)) {
  //   const db = databaseStore.getDatabaseByUID(databaseId);
  //   const selected = selectedDatabaseSchemaByDatabaseName.value.get(db.name);
  //   if (selected) {
  //     const { schema, table } = selected;
  //     if (schema.name) {
  //       return [
  //         `table-${[db.uid, schema.name, table.name].join(
  //           CONNECTION_TREE_DELIMITER
  //         )}`,
  //       ];
  //     } else {
  //       return [
  //         `table-${[db.uid, table.name].join(CONNECTION_TREE_DELIMITER)}`,
  //       ];
  //     }
  //   }
  //   return [`database-${databaseId}`];
  // } else if (instanceId !== String(UNKNOWN_ID)) {
  //   return [`instance-${instanceId}`];
  // }
  // return [];
});

// const allowAdmin = computed(() =>
//   hasWorkspacePermissionV1(
//     "bb.permission.workspace.admin-sql-editor",
//     currentUserV1.value.userRole
//   )
// );

// const connect = (target: CoreTabInfo) => {
//   if (isSimilarTab(target, tabStore.currentTab)) {
//     // Don't go further if the connection doesn't change.
//     return;
//   }
//   if (tabStore.currentTab.isFreshNew) {
//     // If the current tab is "fresh new", update its connection directly.
//     tabStore.updateCurrentTab(target);
//   } else {
//     // Otherwise select or add a new tab and set its connection
//     const name = getSuggestedTabNameFromConnection(target.connection);
//     tabStore.selectOrAddSimilarTab(
//       target,
//       /* beside */ false,
//       /* defaultTabName */ name
//     );
//     tabStore.updateCurrentTab(target);
//   }
// };

const setConnection = (
  option: SQLEditorTreeNode,
  extra: { sheetName?: string; mode: TabMode } = {
    sheetName: undefined,
    mode: TabMode.ReadOnly,
  }
) => {
  // if (option) {
  //   if (option.type === "schema" || option.type === "table") {
  //     // Should be handled in maybeSelectTable
  //     return;
  //   }
  //   if (option.type === "project") {
  //     // Not connectable to a project
  //     return;
  //   }
  //   if (option.type === "instance") {
  //     const instance = instanceStore.getInstanceByUID(option.id);
  //     if (!instanceV1AllowsCrossDatabaseQuery(instance)) {
  //       return;
  //     }
  //   }
  //   const target: CoreTabInfo = {
  //     connection: emptyConnection(),
  //     ...extra,
  //   };
  //   const conn = target.connection;
  //   // If selected item is instance node
  //   if (option.type === "instance") {
  //     conn.instanceId = option.id;
  //   } else if (option.type === "database") {
  //     // If selected item is database node
  //     const database = databaseStore.getDatabaseByUID(option.id);
  //     conn.instanceId = database.instanceEntity.uid;
  //     conn.databaseId = database.uid;
  //   }
  //   connect(target);
  // }
};

// dynamic render the highlight keywords
// const renderLabel = ({ option }: { option: TreeOption }) => {
//   const atom = option as any as ConnectionAtom;
//   return h(Label, { atom, keyword: props.searchPattern ?? "" });
// };

// // Render icons before nodes.
// const renderPrefix = ({ option }: { option: TreeOption }) => {
//   const atom = option as any as ConnectionAtom;
//   return h(Prefix, { atom });
// };

const handleSelect = (key: string) => {
  // const option = dropdownOptions.value.find((item) => item.key === key);
  // if (!option) {
  //   return;
  // }
  // if (key === "alter-schema") {
  //   editorEvents.emit("alter-schema", {
  //     databaseUID: option.item.id,
  //     schema: "",
  //     table: "",
  //   });
  // } else if (key === "connect") {
  //   setConnection(option.item);
  // } else if (key === "connect-in-admin-mode") {
  //   setConnection(option.item, { mode: TabMode.Admin });
  // }
  // showDropdown.value = false;
};

const handleClickoutside = () => {
  showDropdown.value = false;
};

// const maybeExpandKey = (key: string) => {
//   const keys = connectionTreeStore.expandedTreeNodeKeys;
//   if (!keys.includes(key)) {
//     keys.push(key);
//   }
// };

const maybeSelectTable = async (node: SQLEditorTreeNode) => {
  // const parts = atom.id.split(CONNECTION_TREE_DELIMITER);
  // if (parts.length < 2 || parts.length > 3) {
  //   return;
  // }
  // const database = databaseStore.getDatabaseByUID(parts[0]);
  // if (database.uid !== tabStore.currentTab.connection.databaseId) {
  //   const target: CoreTabInfo = {
  //     connection: {
  //       instanceId: database.instanceEntity.uid,
  //       databaseId: database.uid,
  //     },
  //     mode: TabMode.ReadOnly,
  //   };
  //   target.connection.instanceId = database.instanceEntity.uid;
  //   target.connection.databaseId = database.uid;
  //   connect(target);
  //   await nextTick();
  // }
  // const databaseMetadata =
  //   await useDBSchemaV1Store().getOrFetchDatabaseMetadata(database.name);
  // let schemaMetadata: SchemaMetadata | undefined = undefined;
  // if (parts.length === 2) {
  //   // database -> table
  //   schemaMetadata = databaseMetadata.schemas.find((s) => s.name === "");
  // }
  // if (parts.length === 3) {
  //   // database -> schema -> table
  //   const schema = parts[1];
  //   schemaMetadata = databaseMetadata.schemas.find((s) => s.name === schema);
  // }
  // if (!schemaMetadata) {
  //   return;
  // }
  // const table = parts[parts.length - 1];
  // const tableMetadata = schemaMetadata.tables.find((t) => t.name === table);
  // if (!tableMetadata) {
  //   return;
  // }
  // selectedDatabaseSchemaByDatabaseName.value.set(database.name, {
  //   db: database,
  //   database: databaseMetadata,
  //   schema: schemaMetadata,
  //   table: tableMetadata,
  // });
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as any as SQLEditorTreeNode;
  return {
    onClick(e: MouseEvent) {
      if (node.disabled) return;

      if (isDescendantOf(e.target as Element, ".n-tree-node-content")) {
        // Check if clicked on the content part.
        // And ignore the fold/unfold arrow.
        if (node.type === "instance" || node.type === "database") {
          setConnection(node);
        }
        if (node.type === "table") {
          maybeSelectTable(node);
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
    "data-node-type": node.type,
  };
};

const handleLoadSubTree = (option: TreeOption) => {
  const node = option as any as SQLEditorTreeNode;
  const type = node.meta.type;
  if (type === "database") {
    return fetchDatabaseSubTree(node);
  }
  return Promise.resolve();
};

// const updateExpandedKeys = (keys: string[]) => {
//   connectionTreeStore.expandedTreeNodeKeys = keys;
// };

// When switching tabs, scroll the matched node into view if needed.
// const scrollToConnectedNode = (instanceId: string, databaseId: string) => {
//   if (instanceId === String(UNKNOWN_ID) && databaseId === String(UNKNOWN_ID)) {
//     return;
//   }
//   const tree = treeRef.value;
//   if (!tree) {
//     return;
//   }
//   const key =
//     databaseId !== String(UNKNOWN_ID)
//       ? `database-${databaseId}`
//       : `instance-${instanceId}`;

//   nextTick(() => {
//     tree.scrollTo({
//       key,
//     });
//   });
// };

// // Open corresponding tree node when the connection changed.
// watch(
//   [
//     isLoggedIn,
//     () => tabStore.currentTab.connection.instanceId,
//     () => tabStore.currentTab.connection.databaseId,
//     () => connectionTreeStore.tree.state,
//   ],
//   ([isLoggedIn, instanceId, databaseId, treeState]) => {
//     if (!isLoggedIn) {
//       // Don't go further and cleanup the state if we signed out.
//       connectionTreeStore.expandedTreeNodeKeys = [];
//       return;
//     }
//     if (treeState !== ConnectionTreeState.LOADED) {
//       return;
//     }

//     if (instanceId !== String(UNKNOWN_ID)) {
//       maybeExpandKey(`instance-${instanceId}`);
//     }
//     if (databaseId !== String(UNKNOWN_ID)) {
//       const db = databaseStore.getDatabaseByUID(databaseId);
//       const projectId = db.projectEntity.uid;
//       maybeExpandKey(`project-${projectId}`);
//     }

//     scrollToConnectedNode(instanceId, databaseId);
//   },
//   { immediate: true }
// );

watch(
  () => props.searchPattern,
  useThrottleFn(
    (searchPattern: string | undefined) => {
      throttledSearchPattern.value = searchPattern;
    },
    100,
    true /* trailing */,
    true /* leading */
  ),
  {
    immediate: true,
  }
);
</script>

<style lang="postcss">
.databases-tree .n-tree-node-content {
  @apply !pl-0 text-sm;
}
.databases-tree .n-tree-node-wrapper {
  padding: 0;
}
.databases-tree .n-tree-node-indent {
  width: 0.25rem;
}
.databases-tree .n-tree-node-content__prefix {
  @apply shrink-0 !mr-1;
}
.databases-tree.project
  .n-tree-node[data-node-type="project"]
  .n-tree-node-content__prefix {
  @apply hidden;
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
