<template>
  <div class="flex flex-col h-full py-1 gap-y-2">
    <div class="flex items-center gap-x-1 px-1 pt-1">
      <NInput
        v-model:value="keyword"
        size="small"
        :disabled="isLoading"
        :placeholder="$t('sheet.search-sheets')"
        :clearable="true"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
      <NButton
        quaternary
        style="--n-padding: 0 5px; --n-height: 28px"
        @click="showPanel = true"
      >
        <template #icon>
          <heroicons:arrow-left-on-rectangle />
        </template>
      </NButton>
    </div>
    <div
      class="flex-1 flex flex-col h-full overflow-y-auto sheet-tree"
      @scroll="dropdown = undefined"
    >
      <NTree
        ref="treeRef"
        block-line
        style="height: 100%; user-select: none"
        :data="treeData"
        :pattern="keyword.toLowerCase().trim()"
        :default-expand-all="true"
        :selected-keys="selectedKeys"
        :show-irrelevant-nodes="false"
        :expand-on-click="true"
        :render-label="renderLabel"
        :render-prefix="renderPrefix"
        :render-suffix="renderSuffix"
        :node-props="nodeProps"
        :virtual-scroll="false"
      />
      <div v-if="isLoading" class="flex flex-col items-center py-8">
        <BBSpin />
      </div>

      <Dropdown
        v-if="dropdown && isSheetItem(dropdown.item)"
        :sheet="dropdown.item.target"
        :view="view"
        :transparent="true"
        :dropdown-props="{
          trigger: 'manual',
          placement: 'bottom-start',
          show: true,
          x: dropdown.x,
          y: dropdown.y,
          onClickoutside: () => (dropdown = undefined),
        }"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { orderBy, escape } from "lodash-es";
import { NButton, NInput, NTree, NEllipsis, TreeOption } from "naive-ui";
import { storeToRefs } from "pinia";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, onMounted, ref, watch, h } from "vue";
import {
  InstanceV1EngineIcon,
  ProjectV1Name,
  DatabaseV1Name,
} from "@/components/v2";
import {
  useProjectV1Store,
  useSheetAndTabStore,
  useDatabaseV1Store,
  useTabStore,
} from "@/store";
import { ComposedProject, ComposedDatabase, DEFAULT_PROJECT_ID } from "@/types";
import { connectionForTab } from "@/utils";
import {
  SheetViewMode,
  openSheet,
  useSheetContextByView,
  Dropdown,
  useSheetContext,
} from "@/views/sql-editor/Sheet";
import UnsavedPrefix from "./UnsavedPrefix.vue";
import {
  DropdownState,
  MergedItem,
  domIDForItem,
  isSheetItem,
  isTabItem,
} from "./common";

interface TreeNode extends TreeOption {
  key: string;
  label: string;
  project?: ComposedProject;
  database?: ComposedDatabase;
  item?: MergedItem;
  children?: TreeNode[];
}

interface TreeNodeMap {
  [key: string]: {
    key: string;
    label: string;
    project?: ComposedProject;
    database?: ComposedDatabase;
    item?: MergedItem;
    children: TreeNodeMap;
  };
}

const props = defineProps<{
  view: SheetViewMode;
}>();

const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();
const tabStore = useTabStore();
const { showPanel } = useSheetContext();
const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);
const keyword = ref("");
const { currentSheet } = storeToRefs(useSheetAndTabStore());
const dropdown = ref<DropdownState>();

const mergedItemList = computed(() => {
  if (isLoading.value) {
    return [];
  }

  const { tabList } = tabStore;
  const mergedList: MergedItem[] = [];

  if (props.view === "my") {
    // Tabs go ahead
    tabList.forEach((tab) => {
      if (!tab.sheetName) {
        mergedList.push({
          type: "TAB",
          target: tab,
        });
      }
    });
  }
  // Sheets follow
  sheetList.value.forEach((sheet) => {
    mergedList.push({
      type: "SHEET",
      target: sheet,
    });
  });

  const sortedList = orderBy(
    mergedList,
    [
      // Untitled sheets go behind
      // They are probably dirty data
      (item) => (item.type === "SHEET" && !item.target.title ? 1 : 0),
      // Alphabetically otherwise
      (item) => (item.type === "TAB" ? item.target.name : item.target.title),
    ],
    ["asc", "asc"]
  );
  return sortedList;
});

const treeData = computed((): TreeNode[] => {
  const map: TreeNodeMap = {};
  for (const item of mergedItemList.value) {
    let database: ComposedDatabase | undefined;
    if (isTabItem(item)) {
      database = connectionForTab(item.target).database;
    } else {
      database = databaseStore.getDatabaseByName(item.target.database);
    }
    const project =
      database?.projectEntity ??
      projectStore.getProjectByUID(String(DEFAULT_PROJECT_ID));
    if (!map[project.name]) {
      map[project.name] = {
        key: project.name,
        label: project.title,
        project,
        children: {},
      };
    }

    if (database) {
      if (!map[project.name].children[database.name]) {
        map[project.name].children[database.name] = {
          key: database.name,
          label: database.databaseName,
          database,
          children: {},
        };
      }
      map[project.name].children[database.name].children[item.target.name] = {
        key: item.target.name,
        label: isTabItem(item) ? item.target.name : item.target.title,
        item,
        children: {},
      };
    } else {
      map[project.name].children[item.target.name] = {
        key: item.target.name,
        label: isTabItem(item) ? item.target.name : item.target.title,
        item,
        children: {},
      };
    }
  }
  return getTreeNodeList(map);
});

const getTreeNodeList = (treeNodeMap: TreeNodeMap): TreeNode[] => {
  return Object.values(treeNodeMap).map((item) => {
    const children = getTreeNodeList(item.children);
    return {
      ...item,
      isLeaf: children.length === 0,
      children,
    };
  });
};

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  if (treeNode.project) {
    return h(ProjectV1Name, {
      project: treeNode.project,
      link: false,
    });
  } else if (treeNode.database) {
    return h("span", { class: "flex items-center gap-x-1" }, [
      h(InstanceV1EngineIcon, {
        instance: treeNode.database.instanceEntity,
      }),
      h(
        "span",
        {
          class: "text-gray-500 text-sm",
        },
        `(${treeNode.database.effectiveEnvironmentEntity.title})`
      ),
    ]);
  }
};

const renderSuffix = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  if (!treeNode.item) {
    return null;
  }
  const child = [];
  if (isTabItem(treeNode.item)) {
    child.push(h(UnsavedPrefix));
  } else {
    const tab = tabStore.tabList.find(
      (tab) => tab.sheetName === treeNode.item?.target.name
    );
    if (tab?.isSaved ?? true) {
      child.push(
        h(Dropdown, {
          sheet: treeNode.item.target,
          view: props.view,
          secondary: true,
        })
      );
    } else {
      child.push(h(UnsavedPrefix));
    }
  }
  return h(
    "div",
    {
      class: "mr-2",
      onClick(e: MouseEvent) {
        e.stopImmediatePropagation();
        e.preventDefault();
      },
    },
    child
  );
};

const renderLabel = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  if (treeNode.project) {
    return null;
  }
  if (treeNode.database) {
    return h(DatabaseV1Name, {
      database: treeNode.database,
      link: false,
    });
  }

  return h(
    NEllipsis,
    {
      class: "",
    },
    () => [
      h("span", {
        id: treeNode.item ? domIDForItem(treeNode.item) : null,
        innerHTML: escape(treeNode.label),
      }),
    ]
  );
};

const selectedKeys = computed(() => {
  return [currentSheet.value?.name, tabStore.currentTab.id];
});

const nodeProps = ({ option }: { option: TreeOption }) => {
  const treeNode = option as TreeNode;
  return {
    onClick(e: MouseEvent) {
      if (!treeNode.item) {
        return;
      }
      handleItemClick(treeNode.item, e);
    },
    onContextmenu(e: MouseEvent) {
      if (!treeNode.item) {
        return;
      }
      handleRightClick(treeNode.item, e);
    },
  };
};

const handleItemClick = (item: MergedItem, e: MouseEvent) => {
  if (isTabItem(item)) {
    tabStore.setCurrentTabId(item.target.id);
  } else {
    openSheet(item.target, e.metaKey || e.ctrlKey);
  }
};

const handleRightClick = (item: MergedItem, e: MouseEvent) => {
  if (!isSheetItem(item)) return;
  e.preventDefault();
  dropdown.value = undefined;
  nextTick().then(() => {
    dropdown.value = {
      item,
      x: e.clientX,
      y: e.clientY,
    };
  });
};

const scrollToItem = (item: MergedItem | undefined) => {
  if (!item) return;
  const id = domIDForItem(item);
  const elem = document.getElementById(id);
  if (elem) {
    scrollIntoView(elem, {
      scrollMode: "if-needed",
    });
  }
};

const scrollToCurrentTabOrSheet = () => {
  if (currentSheet.value) {
    scrollToItem({ type: "SHEET", target: currentSheet.value });
  } else {
    const tab = tabStore.currentTab;
    scrollToItem({ type: "TAB", target: tab });
  }
};

watch(
  isInitialized,
  async () => {
    if (!isInitialized.value) {
      await fetchSheetList();
      await nextTick();
      scrollToCurrentTabOrSheet();
    }
  },
  { immediate: true }
);

watch(
  [() => currentSheet.value?.name, () => tabStore.currentTab.id],
  () => {
    scrollToCurrentTabOrSheet();
  },
  { immediate: true }
);

onMounted(() => {
  scrollToCurrentTabOrSheet();
});
</script>

<style lang="postcss" scoped>
.sheet-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.sheet-tree :deep(.n-tree-node-content) {
  @apply !pl-0 text-sm;
}
.sheet-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.sheet-tree :deep(.n-tree-node-indent) {
  width: 0.25rem;
}
.sheet-tree :deep(.n-tree-node-content__prefix) {
  @apply shrink-0 !mr-1;
}
.sheet-tree.project
  :deep(.n-tree-node[data-node-type="project"] .n-tree-node-content__prefix) {
  @apply hidden;
}
.sheet-tree :deep(.n-tree-node-content__text) {
  @apply truncate mr-1;
}
.sheet-tree :deep(.n-tree-node--pending) {
  background-color: transparent !important;
}
.sheet-tree :deep(.n-tree-node--pending:hover) {
  background-color: var(--n-node-color-hover) !important;
}
.sheet-tree :deep(.n-tree-node--selected),
.sheet-tree :deep(.n-tree-node--selected:hover) {
  background-color: var(--n-node-color-active) !important;
  font-weight: 500;
}
</style>
