<template>
  <div
    class="flex justify-between items-center box-border text-gray-500 text-sm border-b pr-2"
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
            @contextmenu.stop.prevent="
              contextMenuRef?.show(tabStore.getTabById(id), index, $event)
            "
          />
        </template>
      </Draggable>

      <button class="px-1" @click="handleAddTab">
        <heroicons-solid:plus
          class="h-6 w-6 p-1 hover:bg-gray-200 rounded-md"
        />
      </button>
    </div>

    <div class="pb-1">
      <NButton size="small" @click="showSheetPanel = true">
        {{ $t("sql-editor.sheet.choose-sheet") }}
      </NButton>
    </div>

    <ContextMenu ref="contextMenuRef" />
  </div>
</template>

<script lang="ts" setup>
import { useResizeObserver } from "@vueuse/core";
import { useDialog, NButton } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { ref, reactive, nextTick, computed, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import Draggable from "vuedraggable";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useTabStore } from "@/store";
import type { TabInfo } from "@/types";
import { TabMode } from "@/types";
import { defer, getSuggestedTabNameFromConnection } from "@/utils";
import { useSheetContext } from "../Sheet";
import ContextMenu from "./ContextMenu.vue";
import TabItem from "./TabItem";
import { provideTabListContext } from "./context";

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
const { showPanel: showSheetPanel, events: sheetEvents } = useSheetContext();
const tabListRef = ref<InstanceType<typeof Draggable>>();
const context = provideTabListContext();
const contextMenuRef = ref<InstanceType<typeof ContextMenu>>();

const scrollState = reactive({
  moreLeft: false,
  moreRight: false,
});

const handleSelectTab = async (tab: TabInfo) => {
  tabStore.setCurrentTabId(tab.id);
};

const handleAddTab = () => {
  const connection = { ...tabStore.currentTab.connection };
  const name = getSuggestedTabNameFromConnection(connection);
  tabStore.addTab({
    name,
    connection,
    // The newly created tab is "clean" so its connection can be changed
    isFreshNew: true,
  });
  nextTick(recalculateScrollState);
  sheetEvents.emit("add-sheet");
};

const handleRemoveTab = async (
  tab: TabInfo,
  index: number,
  focusWhenConfirm = false
) => {
  const _defer = defer<boolean>();
  if (tab.mode === TabMode.ReadOnly && !tab.isSaved) {
    if (focusWhenConfirm) {
      tabStore.setCurrentTabId(tab.id);
    }
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
        _defer.resolve(true);
      },
      onNegativeClick() {
        $dialog.destroy();
        _defer.resolve(false);
      },
      positiveText: t("sql-editor.close-sheet"),
      negativeText: t("common.cancel"),
      showIcon: false,
    });
  } else {
    remove(index);
    _defer.resolve(true);
  }

  function remove(index: number) {
    if (tabStore.tabList.length === 1) {
      // Ensure at least 1 tab
      tabStore.addTab();
    }

    tabStore.removeTab(tab);

    // select a tab near the removed tab.
    const nextIndex = Math.min(index, tabStore.tabList.length - 1);
    const nextTab = tabStore.tabList[nextIndex];
    handleSelectTab(nextTab);

    nextTick(recalculateScrollState);
  }

  return _defer.promise;
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
  contextMenuRef.value?.hide();

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

useEmitteryEventListener(
  context.events,
  "close-tab",
  async ({ tab, index, action }) => {
    const tabList = tabStore.tabList;

    const remove = async (tab: TabInfo, index: number) => {
      await handleRemoveTab(tab, index, true);
      await new Promise((r) => requestAnimationFrame(r));
    };

    if (action === "CLOSE") {
      await remove(tab, index);
      return;
    }
    const max = tabList.length - 1;
    if (action === "CLOSE_OTHERS") {
      for (let i = max; i > index; i--) {
        await remove(tabList[i], i);
      }
      for (let i = index - 1; i >= 0; i--) {
        await remove(tabList[i], i);
      }
      return;
    }
    if (action === "CLOSE_TO_THE_RIGHT") {
      for (let i = max; i > index; i--) {
        await remove(tabList[i], i);
      }
      return;
    }
    if (action === "CLOSE_SAVED") {
      for (let i = max; i >= 0; i--) {
        const tab = tabList[i];
        if (tab.isSaved) {
          await remove(tab, i);
        }
      }
      return;
    }
    if (action === "CLOSE_ALL") {
      for (let i = max; i >= 0; i--) {
        await remove(tabList[i], i);
      }
    }
  }
);
</script>

<style scoped lang="postcss">
.ghost {
  @apply opacity-50 bg-white;
}

.tab-list {
  @apply flex flex-nowrap overflow-x-auto max-w-full hide-scrollbar;
}
</style>
