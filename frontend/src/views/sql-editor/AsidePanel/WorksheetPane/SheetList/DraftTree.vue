<template>
  <div class="flex flex-col items-stretch gap-y-1 relative worksheet-tree">
    <NTree
      block-line
      :data="[treeNode]"
      :multiple="false"
      :show-irrelevant-nodes="false"
      :filter="filterNode"
      :pattern="keyword"
      :render-prefix="renderPrefix"
      :render-suffix="renderSuffix"
      :render-label="renderLabel"
      :node-props="nodeProps"
      :selected-keys="selectedKeys"
      :expanded-keys="expandedKeysArray"
      @update:expanded-keys="(keys: string[]) => expandedKeys = new Set(keys)"
    />
  </div>
</template>

<script lang="tsx" setup>
import {
  XIcon,
  FolderCodeIcon,
  FolderMinusIcon,
  FileCodeIcon,
  FolderOpenIcon,
} from "lucide-vue-next";
import { NTree, type TreeOption } from "naive-ui";
import { ref, computed, watch } from "vue";
import { HighlightLabelText } from "@/components/v2";
import {
  useSQLEditorTabStore,
  useCurrentUserV1,
  useTabViewStateStore,
} from "@/store";
import { isDescendantOf, useDynamicLocalStorage } from "@/utils";

interface DraftWorsheetNode extends TreeOption {
  key: string;
  label: string;
  draftId?: string;
  children?: DraftWorsheetNode[];
}

const props = defineProps<{
  keyword: string;
}>();

const tabStore = useSQLEditorTabStore();
const me = useCurrentUserV1();
const { removeViewState } = useTabViewStateStore();
const selectedKeys = ref<string[]>([]);
const expandedKeys = useDynamicLocalStorage<Set<string>>(
  computed(
    () => `bb.sql-editor.worksheet-tree-expand-keys.draft.${me.value.name}`
  ),
  new Set(["/"])
);

// Convert Set to Array once per render cycle instead of spreading in template
const expandedKeysArray = computed(() => Array.from(expandedKeys.value));

const keyForDraft = (tab: { id: string }) => {
  return `bb-worksheet-list-draft-${tab.id}`;
};

// Extract only tree-relevant properties to avoid rebuilding on unrelated changes
// (e.g., statement, status, connection changes should NOT trigger rebuild)
const draftTreeData = computed(() => {
  return tabStore.tabList
    .filter((tab) => !tab.worksheet)
    .map((tab) => ({
      id: tab.id,
      title: tab.title,
      worksheet: tab.worksheet, // Track to detect when draft becomes saved
    }));
});

const treeNode = computed((): DraftWorsheetNode => {
  return {
    isLeaf: false,
    key: "/",
    label: "Draft",
    children: draftTreeData.value.map((draftData) => {
      return {
        isLeaf: true,
        key: keyForDraft({ id: draftData.id }),
        label: draftData.title,
        draftId: draftData.id,
      };
    }),
  };
});

const filterNode = (pattern: string, option: TreeOption) => {
  const node = option as DraftWorsheetNode;
  const keyword = pattern.trim().toLowerCase();
  if (node.key === "/" || !keyword) {
    return true;
  }
  return node.label.toLowerCase().includes(keyword);
};

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as DraftWorsheetNode;
  if (node.draftId) {
    return <FileCodeIcon class="w-4 h-auto text-gray-600" />;
  }

  if (expandedKeys.value.has(node.key)) {
    return <FolderOpenIcon class="w-4 h-auto text-gray-600" />;
  }
  if (node.empty) {
    return <FolderMinusIcon class="w-4 h-auto text-gray-600" />;
  }
  return <FolderCodeIcon class="w-4 h-auto text-gray-600" />;
};

const renderSuffix = ({ option }: { option: TreeOption }) => {
  const node = option as DraftWorsheetNode;
  if (!node.draftId) {
    return null;
  }

  return (
    <XIcon
      class="w-4 h-auto text-gray-600"
      onClick={() => {
        if (node.draftId) {
          const draft = tabStore.tabList.find((t) => t.id === node.draftId);
          if (draft) {
            tabStore.removeTab(draft);
          }
          removeViewState(node.draftId);
        }
      }}
    />
  );
};

const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as DraftWorsheetNode;
  return <HighlightLabelText text={node.label} keyword={props.keyword} />;
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as DraftWorsheetNode;
  return {
    "data-item-key": node.key,
    onClick(e: MouseEvent) {
      if (
        !isDescendantOf(e.target as Element, ".n-tree-node-content__text") &&
        !isDescendantOf(e.target as Element, ".n-tree-node-content__prefix")
      ) {
        return;
      }
      if (node.draftId) {
        tabStore.setCurrentTabId(node.draftId);
      } else {
        if (expandedKeys.value.has(node.key)) {
          expandedKeys.value.delete(node.key);
        } else {
          expandedKeys.value.add(node.key);
        }
      }
    },
  };
};

watch(
  () => ({
    id: tabStore.currentTab?.id,
    worksheet: tabStore.currentTab?.worksheet,
  }),
  ({ id, worksheet }) => {
    selectedKeys.value = [];
    if (!id) {
      return;
    }
    if (worksheet) {
      return;
    }
    const key = keyForDraft({ id });
    selectedKeys.value = [key];
    expandedKeys.value.add("/");
  },
  { immediate: true }
);
</script>

<style lang="postcss" scoped>
.worksheet-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.worksheet-tree :deep(.n-tree-node-content) {
  padding-left: 0 !important;
  padding-right: 1rem !important;
  font-size: 0.875rem;
  line-height: 1.25rem;
  flex: 1;
}
.worksheet-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.worksheet-tree :deep(.n-tree-node-switcher--hide) {
  width: 0.5rem !important;
}
.worksheet-tree :deep(.n-tree-node-content__prefix) {
  flex-shrink: 0;
}
.worksheet-tree :deep(.n-tree-node-content__suffix) {
  flex-shrink: 0;
}
.worksheet-tree :deep(.n-tree-node-content__text) {
  overflow: hidden;
  flex: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
