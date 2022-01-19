<template>
  <div class="query-selector">
    <!-- query tab list-->
    <div
      class="query-selector-wrapper relative overflow-hidden"
      :class="{ 'is-scrolling': scrollState.isScrolling }"
    >
      <div
        ref="tablistRef"
        class="query-selector-tablist"
        @wheel="handleScollTabList"
      >
        <div
          v-for="tab in queryTabList"
          :key="tab.id"
          class="query-selector-tab"
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
            class="label max-w-5xl w-48 overflow-hidden whitespace-nowrap overflow-ellipsis"
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
                @keyup.esc="handleCancelInput"
              />
              <!-- this is a trick -->
              <span class="w-full h-full invisible line-camp-1">
                Edit {{ labelState.currentLabelName }}
              </span>
            </div>
            <span v-else>
              {{ tab.label }}
            </span>
          </div>
          <template v-if="enterTabId === tab.id && queryTabList.length > 1">
            <span
              class="suffix close hover:bg-gray-200 rounded-sm"
              @click.prevent.stop="handleRemoveTab(tab)"
            >
              <heroicons-solid:x class="icon" />
            </span>
          </template>
          <template v-else>
            <template v-if="!tab.isSaved">
              <span class="editing text-gray-400">
                <carbon:dot-mark class="h-4 w-4" />
              </span>
            </template>
            <template v-if="tab.id === activeTabId && queryTabList.length > 1">
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

    <div class="query-selector-add">
      <button
        class="p-1 hover:bg-gray-200 rounded-md"
        @click="handleAddTab({})"
      >
        <heroicons-solid:plus class="h-4 w-4" />
      </button>
    </div>
    <div class="query-selector-more">
      <NPopselect
        v-model:value="selectedTab"
        :options="tabList"
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
          View all Tabs
        </NTooltip>
      </NPopselect>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, reactive, nextTick, computed } from "vue";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";
import {
  useNamespacedGetters,
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";

import {
  TabInfo,
  AnyTabInfo,
  EditorSelectorGetters,
  EditorSelectorState,
  EditorSelectorActions,
  SqlEditorActions,
} from "../../types";
import { debounce } from "lodash-es";

const { currentTab } = useNamespacedGetters<EditorSelectorGetters>(
  "editorSelector",
  ["currentTab"]
);

const { activeTabId, queryTabList } = useNamespacedState<EditorSelectorState>(
  "editorSelector",
  ["activeTabId", "queryTabList"]
);
const { addTab, removeTab, setActiveTabId, updateActiveTab } =
  useNamespacedActions<EditorSelectorActions>("editorSelector", [
    "addTab",
    "removeTab",
    "setActiveTabId",
    "updateActiveTab",
  ]);
const { patchSavedQuery, checkSavedQueryExistById } =
  useNamespacedActions<SqlEditorActions>("sqlEditor", [
    "patchSavedQuery",
    "checkSavedQueryExistById",
  ]);

const store = useStore();
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
const tabList = computed(() => {
  return queryTabList.value.map((tab: TabInfo) => {
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
    if (currentTab.value.currentQueryId) {
      patchSavedQuery({
        id: currentTab.value.currentQueryId,
        name: labelState.currentLabelName,
      });
    }
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
const handleCancelInput = () => {
  labelState.currentLabelName = labelState.oldLabelName;
  updateActiveTab({
    label: labelState.currentLabelName,
  });
  if (currentTab.value.currentQueryId) {
    patchSavedQuery({
      id: currentTab.value.currentQueryId,
      name: labelState.currentLabelName,
    });
  }
  nextTick(() => {
    labelState.isEditingLabel = false;
    reComputedScrollWidth();
  });
};

const handleSelectTab = async (tab: TabInfo) => {
  setActiveTabId(tab.id);

  if (currentTab.value.currentQueryId) {
    const exist = await checkSavedQueryExistById(
      currentTab.value.currentQueryId
    );
    if (!exist) {
      updateActiveTab({
        currentQueryId: undefined,
      });
    }
  }
};
const handleAddTab = (tab: AnyTabInfo) => {
  addTab(tab);
  nextTick(() => {
    const tab = currentTab.value;
    handleEditLabel(tab);
    reComputedScrollWidth();
  });
};
const handleRemoveTab = (tab: TabInfo) => {
  removeTab(tab);
  nextTick(() => {
    reComputedScrollWidth();
  });
};
const handleSelectTabFromPopselect = (tabId: string) => {
  setActiveTabId(tabId);
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
</script>

<style scoped>
.query-selector {
  height: var(--tab-height);
  @apply flex box-border;
  @apply text-gray-500 text-sm;
  @apply border-b;
}

.query-selector-tablist {
  @apply flex overflow-auto;
  max-width: calc(100vw - 112px);
  scrollbar-width: none; /* firefox */
  -ms-overflow-style: none; /* IE 10+ */
}

.query-selector-wrapper.is-scrolling::before {
  @apply absolute top-0 left-0 w-4 h-full z-10;
  content: "";
  transition: box-shadow 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: inset 10px 0 8px -8px rgb(0 0 0 / 16%);
}
.query-selector-wrapper.is-scrolling::after {
  @apply absolute top-0 right-0 w-4 h-full z-10;
  content: "";
  transition: box-shadow 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: inset -10px 0 8px -8px rgb(0 0 0 / 16%);
}
.query-selector-tablist::-webkit-scrollbar {
  display: none; /* Chrome Safari */
}

.query-selector-tab {
  @apply inline-flex place-items-center;
  @apply cursor-pointer box-border;
  @apply px-2 border-r;
  @apply bg-gray-50;
  @apply whitespace-nowrap;
  transform: translateX(0);
  transition: transform 0.3s, -webkit-transform 0.3s;
}

.query-selector-tab.active {
  @apply cursor-text relative;
  @apply bg-white;
  @apply text-accent;
}

.query-selector-tab .label {
  @apply p-2;
}
.query-selector-tab .suffix {
  @apply flex justify-center items-center h-4 w-4;
}

.query-selector-tab .suffix.close {
  @apply cursor-pointer;
  @apply text-gray-500;
}

.query-selector-move-prev,
.query-selector-move-next,
.query-selector-add {
  @apply flex items-center;
  @apply cursor-pointer;
  @apply p-2;
}

.query-selector-more {
  @apply flex items-center justify-end flex-1;
  @apply cursor-pointer;
  @apply p-2;
}
</style>
