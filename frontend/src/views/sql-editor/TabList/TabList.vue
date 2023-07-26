<template>
  <div
    class="flex justify-between items-center box-border text-gray-500 text-sm border-b pr-1"
  >
    <div class="relative flex flex-1 flex-nowrap overflow-hidden">
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
            :data-tab-id="id"
            @select="(tab) => handleSelectTab(tab)"
            @close="(tab, index) => handleRemoveTab(tab, index)"
          />
        </template>
      </Draggable>

      <button class="px-1" @click="handleAddTab">
        <heroicons-solid:plus
          class="h-6 w-6 p-1 hover:bg-gray-200 rounded-md"
        />
      </button>
    </div>

    <NButton size="tiny" quaternary @click="state.showSheetPanel = true">
      {{ $t("sql-editor.sheet.choose-sheet") }}
    </NButton>

    <Drawer v-model:show="state.showSheetPanel">
      <DrawerContent :title="$t('sql-editor.sheet.self')">
        <SheetPanel @close="state.showSheetPanel = false" />
      </DrawerContent>
    </Drawer>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, nextTick, computed, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useDialog, NButton } from "naive-ui";
import Draggable from "vuedraggable";
import scrollIntoView from "scroll-into-view-if-needed";
import { useResizeObserver } from "@vueuse/core";

import type { TabInfo } from "@/types";
import { TabMode } from "@/types";
import { useTabStore } from "@/store";
import { Drawer, DrawerContent } from "@/components/v2";
import TabItem from "./TabItem";
import SheetPanel from "./SheetPanel";

type LocalState = {
  dragging: boolean;
  hoverTabId: string;
  showSheetPanel: boolean;
};

const tabStore = useTabStore();

const { t } = useI18n();
const dialog = useDialog();

const state = reactive<LocalState>({
  dragging: false,
  hoverTabId: "",
  showSheetPanel: false,
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

const handleRemoveTab = async (tab: TabInfo, index: number) => {
  if (tab.mode === TabMode.ReadOnly && !tab.isSaved) {
    const $dialog = dialog.create({
      title: t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.title"),
      content: t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.content"),
      type: "info",
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      onPositiveClick() {
        remove(index);
        $dialog.destroy();
      },
      onNegativeClick() {
        $dialog.destroy();
      },
      positiveText: t("sql-editor.close-sheet"),
      negativeText: t("common.cancel"),
      showIcon: false,
    });
  } else {
    remove(index);
  }

  function remove(index: number) {
    if (tabStore.tabList.length <= 1) return;

    tabStore.removeTab(tab);

    // select a tab near the removed tab.
    const nextIndex = Math.min(index, tabStore.tabList.length - 1);
    const nextTab = tabStore.tabList[nextIndex];
    handleSelectTab(nextTab);

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
  } else {
    scrollState.moreRight = false;
  }
};

watch(
  () => tabStore.currentTabId,
  (id) => {
    requestAnimationFrame(() => {
      // Scroll the selected tab into view if needed.
      const tabElements = Array.from(document.querySelectorAll(".tab-item"));
      const tabElement = tabElements.find(
        (elem) => elem.getAttribute("data-tab-id") === id
      );
      if (tabElement) {
        scrollIntoView(tabElement, {
          scrollMode: "if-needed",
        });
      }
    });
  },
  { immediate: true }
);

onMounted(() => recalculateScrollState());
</script>

<style scoped lang="postcss">
.ghost {
  @apply opacity-50 bg-white;
}

.tab-list {
  @apply flex flex-nowrap overflow-x-auto max-w-full hide-scrollbar;
}
</style>
