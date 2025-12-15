<template>
  <div
    class="bb-sql-editor-tab-list flex justify-between items-center box-border text-gray-500 text-sm border-b pr-2 gap-1"
  >
    <NScrollbar
      ref="scrollbarRef"
      :x-scrollable="true"
      trigger="hover"
      class="scrollbar"
      content-class="flex gap-x-1"
      @scroll="recalculateScrollState"
    >
      <Draggable
        id="tab-list"
        ref="tabListRef"
        v-model="tabStore.openTabList"
        item-key="id"
        animation="300"
        class="relative flex flex-nowrap overflow-hidden h-9 pt-0.5 hide-scrollbar"
        :class="{
          'more-left': scrollState.moreLeft,
          'more-right': scrollState.moreRight,
        }"
        ghost-class="ghost"
      >
        <template
          #item="{ element: tab, index }: { element: SQLEditorTab; index: number }"
        >
          <TabItem
            :tab="tab"
            :index="index"
            :data-tab-id="tab.id"
            @select="(tab) => tabStore.setCurrentTabId(tab.id)"
            @close="(tab) => handleRemoveTab(tab)"
            @contextmenu.stop.prevent="
              contextMenuRef?.show(tab, index, $event)
            "
          />
        </template>
      </Draggable>

      <div
        class="shrink-0 sticky right-0 bg-white flex items-stretch justify-end"
      >
        <button
          class="bg-gray-200/20 hover:bg-accent/10py-1 px-1.5 border-t border-x rounded-t hover:border-accent"
          :disabled="state.loading"
          @click="handleAddTab"
        >
          <PlusIcon class="h-5 w-5" stroke-width="2.5" />
        </button>
      </div>
    </NScrollbar>

    <div class="flex items-center gap-2">
      <BrandingLogoWrapper>
        <ProfileDropdown :link="true" />
      </BrandingLogoWrapper>
    </div>

    <ContextMenu ref="contextMenuRef" />
  </div>
</template>

<script lang="ts" setup>
import { useResizeObserver } from "@vueuse/core";
import { PlusIcon } from "lucide-vue-next";
import { NScrollbar, useDialog } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import Draggable from "vuedraggable";
import ProfileDropdown from "@/components/ProfileDropdown.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { defer, usePreventBackAndForward } from "@/utils";
import { useSQLEditorContext } from "../context";
import BrandingLogoWrapper from "./BrandingLogoWrapper.vue";
import ContextMenu from "./ContextMenu.vue";
import { useTabListContext } from "./context";
import TabItem from "./TabItem/TabItem.vue";

type LocalState = {
  hoverTabId: string;
  loading: boolean;
};

const tabStore = useSQLEditorTabStore();
const editorContext = useSQLEditorContext();
const context = useTabListContext();

const { t } = useI18n();
const dialog = useDialog();

const state = reactive<LocalState>({
  hoverTabId: "",
  loading: false,
});
const scrollbarRef = ref<InstanceType<typeof NScrollbar>>();
const tabListRef = ref<InstanceType<typeof Draggable>>();
const contextMenuRef = ref<InstanceType<typeof ContextMenu>>();

const scrollState = reactive({
  moreLeft: false,
  moreRight: false,
});

const scrollElement = computed((): HTMLElement | null | undefined => {
  return scrollbarRef.value?.scrollbarInstRef?.containerRef;
});

const handleAddTab = async () => {
  if (state.loading) {
    return;
  }

  state.loading = true;
  try {
    await editorContext.createWorksheet({});
    nextTick(() => {
      requestAnimationFrame(() => {
        const elem = scrollElement.value;
        if (elem) {
          elem.scrollTo(elem.scrollWidth, 0);
        }
        requestAnimationFrame(() => {
          recalculateScrollState();
        });
      });
    });
  } finally {
    state.loading = false;
  }
};

const handleRemoveTab = async (tab: SQLEditorTab, focusWhenConfirm = false) => {
  const _defer = defer<boolean>();
  if (tab.mode === "WORKSHEET" && tab.status === "DIRTY") {
    if (focusWhenConfirm) {
      tabStore.setCurrentTabId(tab.id);
    }
    const $dialog = dialog.create({
      title: t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.title"),
      content: t("sql-editor.hint-tips.confirm-to-close-unsaved-sheet.content"),
      type: "warning",
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      onPositiveClick() {
        remove();
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
    remove();
    _defer.resolve(true);
  }

  function remove() {
    tabStore.closeTab(tab.id);
    nextTick(recalculateScrollState);
  }

  return _defer.promise;
};

usePreventBackAndForward(scrollElement);
useResizeObserver(tabListRef, () => {
  recalculateScrollState();
});

const recalculateScrollState = () => {
  contextMenuRef.value?.hide();

  const element = scrollElement.value;
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
    const { openTabList } = tabStore;

    const remove = async (tab: SQLEditorTab) => {
      await handleRemoveTab(tab, true);
      await new Promise((r) => requestAnimationFrame(r));
    };

    const max = openTabList.length - 1;
    switch (action) {
      case "CLOSE": {
        await remove(tab);
        return;
      }
      case "CLOSE_OTHERS": {
        for (let i = max; i > index; i--) {
          await remove(openTabList[i]);
        }
        for (let i = index - 1; i >= 0; i--) {
          await remove(openTabList[i]);
        }
        return;
      }
      case "CLOSE_TO_THE_RIGHT": {
        for (let i = max; i > index; i--) {
          await remove(openTabList[i]);
        }
        return;
      }
      case "CLOSE_SAVED": {
        for (let i = max; i >= 0; i--) {
          const tab = openTabList[i];
          if (tab.status === "CLEAN") {
            await remove(tab);
          }
        }
        return;
      }
      case "CLOSE_ALL": {
        for (let i = max; i >= 0; i--) {
          await remove(openTabList[i]);
        }
        return;
      }
    }
  }
);
</script>

<style lang="postcss">
.bb-sql-editor-tab-list .ghost {
  opacity: 0.3;
  background-color: white;
}
.bb-sql-editor-tab-list .scrollbar .n-scrollbar-rail--horizontal--bottom {
  inset: auto 2px 2px 2px !important;
}
</style>
