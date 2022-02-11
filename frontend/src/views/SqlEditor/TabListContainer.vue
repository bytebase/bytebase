<template>
  <div class="tab-list-container">
    <!-- tab list-->
    <div
      class="tab-list-wrapper relative overflow-hidden"
      :class="{ 'is-scrolling': scrollState.isScrolling }"
    >
      <div
        ref="tablistRef"
        class="tab-list-tablist"
        @wheel="handleScollTabList"
      >
        <div
          v-for="tab in tabList"
          :key="tab.id"
          class="tab-list-tab"
          :class="{ active: tab.id === activeTabId }"
          :style="scrollState.style"
          @click="handleSelectTab(tab)"
          @mouseover="enterTabId = tab.id"
          @mouseleave="enterTabId = ''"
        >
          <span class="prefix">
            <carbon:code class="h-4 w-4" />
          </span>
          <div
            class="label max-w-5xl w-48 truncate"
            @dblclick="handleEditLabel(tab)"
          >
            <div
              v-if="
                labelState.isEditingLabel && labelState.editingTabId === tab.id
              "
              class="label-input relative"
            >
              <input
                ref="labelInputRef"
                v-model="labelState.currentLabelName"
                type="text"
                class="rounded px-2 py-0 text-sm w-full absolute left-0 bottom-0"
                @blur="(e: Event) => handleTryChangeLabel()"
                @keyup.enter="(e: Event) => handleTryChangeLabel()"
                @keyup.esc="handleCancelChangeLabel"
              />
              <!-- this is a trick -->
              <span class="w-full h-full invisible">
                Edit {{ labelState.currentLabelName }}
              </span>
            </div>
            <span v-else>
              {{ tab.label }}
            </span>
          </div>
          <template v-if="enterTabId === tab.id && tabList.length > 1">
            <span
              class="suffix close hover:bg-gray-200 rounded-sm"
              @click.prevent.stop="handleRemoveTab(tab)"
            >
              <heroicons-solid:x class="icon" />
            </span>
          </template>
          <template v-else>
            <template v-if="!tab.isSaved">
              <span class="suffix editing text-gray-400">
                <carbon:dot-mark class="h-4 w-4" />
              </span>
            </template>
            <template v-else-if="tab.id === activeTabId && tabList.length > 1">
              <span
                class="suffix close hover:bg-gray-200 rounded-sm"
                @click.prevent="handleRemoveTab(tab)"
              >
                <heroicons-solid:x class="icon" />
              </span>
            </template>
            <template v-else>
              <span class="suffix" />
            </template>
          </template>
        </div>
      </div>
    </div>

    <div class="tab-list-add">
      <NTooltip
        trigger="hover"
        placement="bottom-center"
        :disabled="!isDisconnected"
      >
        <template #trigger>
          <button
            class="p-1 hover:bg-gray-200 rounded-md"
            :class="{ 'cursor-not-allowed': isDisconnected }"
            @click="handleAddTab({})"
          >
            <heroicons-solid:plus class="h-4 w-4" />
          </button>
        </template>
        Please select connections
      </NTooltip>
    </div>
    <div class="tab-list-more">
      <NPopselect
        v-model:value="selectedTab"
        :options="localTabList"
        trigger="click"
        size="medium"
        scrollable
        @update:value="handleSelectTabFromPopselect"
      >
        <NTooltip trigger="hover" placement="left-center">
          <template #trigger>
            <button class="ml-8 p-1 hover:bg-gray-200 rounded-md">
              <heroicons-solid:chevron-down class="h-4 w-4" />
            </button>
          </template>
          {{ $t("sql-editor.view-all-tabs") }}
        </NTooltip>
      </NPopselect>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, reactive, nextTick, computed, onMounted } from "vue";
import { debounce, cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import {
  useNamespacedGetters,
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";

import {
  TabInfo,
  AnyTabInfo,
  SqlEditorGetters,
  SqlEditorActions,
  TabGetters,
  TabState,
  TabActions,
  SheetActions,
} from "../../types";
import { getDefaultTab } from "../../store/modules/tab";

// getters map
const { currentTab, hasTabs } = useNamespacedGetters<TabGetters>("tab", [
  "currentTab",
  "hasTabs",
]);
const { isDisconnected } = useNamespacedGetters<SqlEditorGetters>("sqlEditor", [
  "isDisconnected",
]);

// state map
const { activeTabId, tabList } = useNamespacedState<TabState>("tab", [
  "activeTabId",
  "tabList",
]);

// actions map
const { addTab, removeTab, setActiveTabId, updateActiveTab } =
  useNamespacedActions<TabActions>("tab", [
    "addTab",
    "removeTab",
    "setActiveTabId",
    "updateActiveTab",
  ]);
const { createSheet, patchSheetById } = useNamespacedActions<SheetActions>(
  "sheet",
  ["createSheet", "patchSheetById"]
);
const { setActiveConnectionByTab } = useNamespacedActions<SqlEditorActions>(
  "sqlEditor",
  ["setActiveConnectionByTab"]
);

const store = useStore();
const router = useRouter();
const { t } = useI18n();

const enterTabId = ref("");
const selectedTab = computed(() => activeTabId.value);
// edit label state
const labelState = reactive({
  isEditingLabel: false,
  currentLabelName: "",
  oldLabelName: "",
  editingTabId: "",
});
const labelInputRef = ref<HTMLInputElement>();

const localTabList = computed(() => {
  return tabList.value.map((tab: TabInfo) => {
    return {
      label: tab.label,
      value: tab.id,
    };
  });
});
// scroll label state
const tablistRef = ref<HTMLInputElement>();
const scrollState = reactive({
  style: {},
  isScrolling: false,
  scrollWidth: 0,
  offsetWidth: 0,
});

const scrollingDistance = computed(() => {
  return scrollState.scrollWidth - scrollState.offsetWidth;
});

const reComputedScrollWidth = () => {
  scrollState.scrollWidth = tablistRef.value?.scrollWidth as number;
  scrollState.offsetWidth = tablistRef.value?.offsetWidth as number;
  scrollState.isScrolling = scrollingDistance.value > 0;
};

const updateSheetName = () => {
  if (currentTab.value.sheetId) {
    patchSheetById({
      id: currentTab.value.sheetId,
      name: labelState.currentLabelName,
    });
  }
};

// Edit label logic
const handleEditLabel = (tab: TabInfo) => {
  labelState.isEditingLabel = true;
  labelState.editingTabId = tab.id;
};
const handleTryChangeLabel = () => {
  if (labelState.currentLabelName !== "") {
    labelState.isEditingLabel = false;
    updateActiveTab({
      label: labelState.currentLabelName,
    });

    updateSheetName();

    nextTick(() => {
      reComputedScrollWidth();
      scrollState.style = {
        transform: `translateX(0px)`,
      };
    });
  } else {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.please-input-the-tab-label"),
    });
  }
};
const handleCancelChangeLabel = () => {
  labelState.currentLabelName = labelState.oldLabelName;
  updateActiveTab({
    label: labelState.currentLabelName,
  });

  updateSheetName();

  nextTick(() => {
    labelState.isEditingLabel = false;
    reComputedScrollWidth();
  });
};

const handleSelectTab = async (tab: TabInfo) => {
  setActiveTabId(tab.id);
  setActiveConnectionByTab(router);
};
const handleAddTab = (tab: AnyTabInfo) => {
  if (isDisconnected.value) return;

  addTab(tab);

  nextTick(async () => {
    const tab = cloneDeep(currentTab.value);
    handleEditLabel(tab);

    // make a relation between the new sheet and the current tab
    const newSheet = await createSheet();

    updateActiveTab({
      sheetId: newSheet.id,
    });

    reComputedScrollWidth();
  });
};
const handleRemoveTab = async (tab: TabInfo) => {
  await removeTab(tab);
  setActiveConnectionByTab(router);
  nextTick(() => {
    reComputedScrollWidth();
  });
};
const handleSelectTabFromPopselect = (tabId: string) => {
  setActiveTabId(tabId);
  setActiveConnectionByTab(router);
};

const handleScollTabList = debounce((e: WheelEvent) => {
  console.log(e);
  console.log(e.deltaX > 0 ? "Move to right" : "Move to left");
  console.log(e.offsetX);
}, 333);

watch(
  () => labelState.isEditingLabel,
  (newVal) => {
    if (newVal) {
      labelState.currentLabelName = currentTab.value.label;
      labelState.oldLabelName = currentTab.value.label;
      nextTick(() => {
        labelInputRef.value?.focus();
      });
    } else {
      nextTick(() => {
        labelState.currentLabelName = "";
        labelState.editingTabId = "";
        labelState.oldLabelName = "";
      });
    }
  }
);

onMounted(async () => {
  if (!hasTabs.value) {
    addTab(getDefaultTab());
    if (!isDisconnected.value) {
      // make a relation between the new sheet and the current tab
      const newSheet = await createSheet();

      updateActiveTab({
        sheetId: newSheet.id,
      });

      reComputedScrollWidth();
    }
  }
});
</script>

<style scoped>
.tab-list-container {
  height: var(--tab-height);
  @apply flex box-border;
  @apply text-gray-500 text-sm;
  @apply border-b;
}

.tab-list-tablist {
  @apply flex overflow-auto;
  max-width: calc(100vw - 112px);
  scrollbar-width: none; /* firefox */
  -ms-overflow-style: none; /* IE 10+ */
}

.tab-list-wrapper.is-scrolling::before {
  @apply absolute top-0 left-0 w-4 h-full z-10;
  content: "";
  transition: box-shadow 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: inset 10px 0 8px -8px rgb(0 0 0 / 16%);
}
.tab-list-wrapper.is-scrolling::after {
  @apply absolute top-0 right-0 w-4 h-full z-10;
  content: "";
  transition: box-shadow 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: inset -10px 0 8px -8px rgb(0 0 0 / 16%);
}
.tab-list-tablist::-webkit-scrollbar {
  display: none; /* Chrome Safari */
}

.tab-list-tab {
  @apply inline-flex place-items-center;
  @apply cursor-pointer box-border;
  @apply px-2 border-r;
  @apply bg-gray-50;
  @apply whitespace-nowrap;
  transform: translateX(0);
  transition: transform 0.3s, -webkit-transform 0.3s;
}

.tab-list-tab.active {
  @apply cursor-text relative;
  @apply bg-white;
  @apply text-accent;
}

.tab-list-tab .label {
  @apply p-2;
}
.tab-list-tab .suffix {
  @apply flex justify-center items-center h-4 w-4;
}

.tab-list-tab .suffix.close {
  @apply cursor-pointer;
  @apply text-gray-500;
}

.tab-list-move-prev,
.tab-list-move-next,
.tab-list-add {
  @apply flex items-center;
  @apply cursor-pointer;
  @apply p-2;
}

.tab-list-more {
  @apply flex items-center justify-end flex-1;
  @apply cursor-pointer;
  @apply p-2;
}
</style>
