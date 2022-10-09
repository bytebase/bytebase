<template>
  <div class="flex box-border text-gray-500 text-sm border-b">
    <div class="relative flex flex-nowrap overflow-hidden">
      <Draggable
        id="tab-list"
        ref="tabListRef"
        v-model="tabStore.tabIdList"
        item-key="id"
        animation="300"
        class="tab-list"
        :class="{
          'more-left': scrollState.moreLeft,
          'more-right': scrollState.moreRight,
        }"
        ghost-class="ghost"
        @start="state.dragging = true"
        @end="state.dragging = false"
        @scroll="recalculateScrollState"
      >
        <template
          #item="{ element: id, index }: { element: string, index: number }"
        >
          <TabItem
            :tab="tabStore.getTabById(id)"
            :index="index"
            @select="(tab) => handleSelectTab(tab)"
            @close="(tab) => handleRemoveTab(tab)"
          />
        </template>
      </Draggable>
    </div>

    <button class="px-1" @click="handleAddTab">
      <heroicons-solid:plus class="h-6 w-6 p-1 hover:bg-gray-200 rounded-md" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, nextTick, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useDialog } from "naive-ui";
import Draggable from "vuedraggable";

import type { TabInfo } from "@/types";
import { useTabStore } from "@/store";
import TabItem from "./TabItem";
import { useResizeObserver } from "@vueuse/core";

type LocalState = {
  dragging: boolean;
  hoverTabId: string;
};

const tabStore = useTabStore();

const { t } = useI18n();
const dialog = useDialog();

const state = reactive<LocalState>({
  dragging: false,
  hoverTabId: "",
});

const tabListRef = ref<InstanceType<typeof Draggable>>();

const scrollState = reactive({
  moreLeft: false,
  moreRight: false,
});

const handleSelectTab = async (tab: TabInfo) => {
  tabStore.setCurrentTabId(tab.id);
};

const handleAddTab = () => {
  tabStore.addTab();
  nextTick(recalculateScrollState);
};

const handleRemoveTab = async (tab: TabInfo) => {
  if (!tab.isSaved) {
    const $dialog = dialog.create({
      title: t("sql-editor.hint-tips.confirm-to-close-unsaved-tab"),
      type: "info",
      onPositiveClick() {
        $dialog.destroy();
      },
      async onNegativeClick() {
        await remove();
        $dialog.destroy();
      },
      negativeText: t("common.confirm"),
      positiveText: t("common.cancel"),
      showIcon: false,
    });
  } else {
    await remove();
  }

  async function remove() {
    await tabStore.removeTab(tab);
    const tabsLength = tabStore.tabList.length;

    if (tabsLength > 0) {
      handleSelectTab(tabStore.tabList[tabsLength - 1]);
    }
    nextTick(recalculateScrollState);
  }
};

const tabListElement = computed((): HTMLElement | undefined => {
  const list = tabListRef.value;
  if (!list) return undefined;
  const element = list.$el as HTMLElement;
  return element;
});

useResizeObserver(tabListRef, () => {
  recalculateScrollState();
});

const recalculateScrollState = () => {
  const element = tabListElement.value;
  if (!element) {
    return;
  }

  const { scrollWidth, offsetWidth, scrollLeft } = element;

  if (scrollLeft === 0) {
    scrollState.moreLeft = false;
  } else {
    scrollState.moreLeft = true;
  }

  if (scrollWidth > offsetWidth) {
    // The actual width is wider than the visible width
    const rightBound = scrollLeft + offsetWidth;
    if (rightBound < scrollWidth) {
      scrollState.moreRight = true;
    } else {
      scrollState.moreRight = false;
    }
  }
};

onMounted(() => recalculateScrollState());
</script>

<style scoped lang="postcss">
.ghost {
  @apply opacity-50 bg-white;
}

.tab-list {
  @apply flex flex-nowrap overflow-x-auto w-full hide-scrollbar;
}

.tab-list::before {
  @apply absolute top-0 left-0 w-4 h-full z-50 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.tab-list::after {
  @apply absolute top-0 right-0 w-4 h-full z-50 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.tab-list.more-left::before {
  box-shadow: inset 1rem 0 0.5rem -0.5rem rgb(0 0 0 / 25%);
}
.tab-list.more-right::after {
  box-shadow: inset -1rem 0 0.5rem -0.5rem rgb(0 0 0 / 25%);
}
</style>
