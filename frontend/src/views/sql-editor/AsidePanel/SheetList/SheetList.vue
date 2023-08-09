<template>
  <div class="flex flex-col h-full overflow-hidden">
    <div class="px-2 py-2 gap-x-1 flex items-center">
      <NInput
        v-model:value="keyword"
        :disabled="isLoading"
        :placeholder="$t('sheet.search-sheets')"
        :clearable="true"
      >
        <template #prefix>
          <heroicons-outline:search class="h-5 w-5 text-gray-300" />
        </template>
      </NInput>
      <NButton quaternary style="--n-padding: 0 8px" @click="addSheet">
        <template #icon>
          <heroicons:plus />
        </template>
      </NButton>
    </div>
    <div
      class="flex-1 flex flex-col h-full overflow-y-auto"
      @scroll="dropdown = undefined"
    >
      <div
        v-if="!isLoading && filteredItemList.length === 0"
        class="flex flex-col items-center justify-center text-control-placeholder"
      >
        <p class="py-8">{{ $t("common.no-data") }}</p>
      </div>
      <template v-for="item in filteredItemList">
        <TabItem
          v-if="isTabItem(item)"
          :key="`tab-${item.target.name}`"
          :item="item"
          :is-current-item="isCurrentItem(item)"
          :keyword="keyword"
          @click="(item, e) => handleItemClick(item, e)"
        />
        <SheetItem
          v-else
          :key="`sheet-${item.target.name}`"
          :item="item"
          :is-current-item="isCurrentItem(item)"
          :keyword="keyword"
          :view="view"
          @click="(item, e) => handleItemClick(item, e)"
          @contextmenu="(item, e) => handleRightClick(item, e)"
        />
      </template>
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
import { orderBy } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { storeToRefs } from "pinia";
import scrollIntoView from "scroll-into-view-if-needed";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { useSheetAndTabStore, useTabStore } from "@/store";
import { Sheet } from "@/types/proto/v1/sheet_service";
import {
  SheetViewMode,
  openSheet,
  useSheetContextByView,
  Dropdown,
} from "@/views/sql-editor/Sheet";
import SheetItem from "./SheetItem.vue";
import TabItem from "./TabItem.vue";
import {
  DropdownState,
  MergedItem,
  domIDForItem,
  isSheetItem,
  isTabItem,
} from "./common";

const props = defineProps<{
  view: SheetViewMode;
}>();

const emit = defineEmits<{
  (event: "add-tab"): void;
}>();

const tabStore = useTabStore();
const { isInitialized, isLoading, sheetList, fetchSheetList } =
  useSheetContextByView(props.view);
const keyword = ref("");
const { currentSheet } = storeToRefs(useSheetAndTabStore());
const dropdown = ref<DropdownState>();

const sortedSheetList = computed(() => {
  return orderBy<Sheet>(
    sheetList.value,
    [
      (sheet) => (sheet.title ? 0 : 1), // Untitled sheets go behind.
      (sheet) => sheet.title,
    ],
    ["asc", "asc"]
  );
});

const mergedItemList = computed(() => {
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
  if (!isLoading.value) {
    // Sheets follow
    sortedSheetList.value.forEach((sheet) => {
      mergedList.push({
        type: "SHEET",
        target: sheet,
      });
    });
  }

  return mergedList;
});

const filteredItemList = computed(() => {
  const kw = keyword.value.toLowerCase().trim();
  if (!kw) return mergedItemList.value;
  return mergedItemList.value.filter((item) => {
    if (isTabItem(item)) {
      return item.target.name.toLowerCase().includes(kw);
    }
    if (isSheetItem(item)) {
      return item.target.title.toLowerCase().includes(kw);
    }
    throw new Error("should never reach this line.");
  });
});

const isCurrentItem = (item: MergedItem) => {
  if (isSheetItem(item)) {
    return item.target.name === currentSheet.value?.name;
  }
  // isTabItem
  return item.target.id === tabStore.currentTab.id;
};

const handleItemClick = (item: MergedItem, e: MouseEvent) => {
  if (isTabItem(item)) {
    tabStore.setCurrentTabId(item.target.id);
  } else {
    openSheet(item.target, e.metaKey || e.ctrlKey);
  }
};

const addSheet = () => {
  useTabStore().addTab();
  emit("add-tab");
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
