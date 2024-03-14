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
        @dismiss="dropdown = undefined"
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
import { useI18n } from "vue-i18n";
import {
  InstanceV1EngineIcon,
  ProjectV1Name,
  DatabaseV1Name,
} from "@/components/v2";
import {
  useProjectV1Store,
  useWorkSheetAndTabStore,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { ComposedProject, ComposedDatabase, UNKNOWN_ID } from "@/types";
import { connectionForSQLEditorTab } from "@/utils";
import {
  SheetViewMode,
  openWorksheetByName,
  useSheetContextByView,
  Dropdown,
  useSheetContext,
} from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import UnsavedPrefix from "./UnsavedPrefix.vue";
import {
  DropdownState,
  MergedItem,
  domIDForItem,
  isSheetItem,
  isTabItem,
  keyOfItem,
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

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();
const tabStore = useSQLEditorTabStore();
const editorContext = useSQLEditorContext();
const worksheetContext = useSheetContext();
const { showPanel } = worksheetContext;
const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);
const keyword = ref("");
const { currentSheet } = storeToRefs(useWorkSheetAndTabStore());
const dropdown = ref<DropdownState>();

const mergedItemList = computed(() => {
  if (isLoading.value) {
    return [];
  }

  const mergedList: MergedItem[] = [];

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
      (item) => item.target.title,
    ],
    ["asc", "asc"]
  );
  return sortedList;
});

const treeData = computed((): TreeNode[] => {
  const map: TreeNodeMap = {};
  for (const item of mergedItemList.value) {
    let database: ComposedDatabase | undefined;
    let project: ComposedProject | undefined;
    if (isTabItem(item)) {
      database = connectionForSQLEditorTab(item.target).database;
      project = database?.projectEntity;
    } else {
      database = databaseStore.getDatabaseByName(item.target.database);
      project = projectStore.getProjectByName(item.target.project);
    }

    project = project ?? projectStore.getProjectByUID(String(UNKNOWN_ID));
    if (!map[project.name]) {
      map[project.name] = {
        key: project.name,
        label: project.title,
        project,
        children: {},
      };
    }

    const key = keyOfItem(item);
    if (database && database.uid !== `${UNKNOWN_ID}`) {
      if (!map[project.name].children[database.name]) {
        map[project.name].children[database.name] = {
          key: database.name,
          label: database.databaseName,
          database,
          children: {},
        };
      }
      map[project.name].children[database.name].children[key] = {
        key,
        label: item.target.title,
        item,
        children: {},
      };
    } else {
      map[project.name].children[key] = {
        key,
        label: item.target.title,
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
    if (treeNode.project.uid === `${UNKNOWN_ID}`) {
      return h("div", {}, t("sheet.unconnected"));
    }
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
  const { item } = treeNode;
  if (isTabItem(item)) {
    child.push(h(UnsavedPrefix));
  } else {
    const tab = tabStore.tabList.find((tab) => tab.sheet === item.target.name);
    if (tab?.status === "CLEAN") {
      child.push(
        h(Dropdown, {
          sheet: item.target,
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
  return [currentSheet.value?.name, tabStore.currentTab?.id].filter(
    (item): item is string => !!item
  );
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
    openWorksheetByName(
      item.target.name,
      editorContext,
      worksheetContext,
      e.metaKey || e.ctrlKey
    );
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
    if (!tab) {
      return;
    }
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
  [() => currentSheet.value?.name, () => tabStore.currentTab?.id],
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
