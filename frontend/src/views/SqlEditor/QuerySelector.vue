<template>
  <div class="query-selector border-b">
    <div
      v-for="tab in queryTabList"
      :key="tab.id"
      class="query-selector--tab"
      :class="{ active: tab.id === activeTabId }"
      @click="handleSelectTab(tab)"
      @mouseover="enterTabId = tab.id"
      @mouseleave="enterTabId = ''"
    >
      <span class="prefix">
        <carbon:code class="h-4 w-4" />
      </span>
      <span class="label">
        {{ tab.label }}
      </span>
      <template v-if="enterTabId === tab.id && queryTabList.length > 1">
        <span
          class="suffix close hover:bg-gray-200 rounded-sm"
          @click.prevent="handleRemoveTab(tab)"
        >
          <heroicons-solid:x class="icon" />
        </span>
      </template>
      <template v-else>
        <!-- <template v-if="!tab.isSaved">
          <span class="editing text-gray-400">
            <carbon:dot-mark class="h-4 w-4" />
          </span>
        </template> -->
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
    <button
      v-if="isDev()"
      class="query-selector--added"
      @click="handleAddTab({})"
    >
      <heroicons-solid:plus class="h-4 w-4 hover:bg-gray-200 rounded-sm" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";

import {
  EditorSelectorState,
  TabInfo,
  AnyTabInfo,
  EditorSelectorActions,
} from "../../types";
import { isDev } from "../../utils";

const { activeTabId, queryTabList } = useNamespacedState<EditorSelectorState>(
  "editorSelector",
  ["activeTabId", "queryTabList"]
);
const { addTab, removeTab, setActiveTabId } =
  useNamespacedActions<EditorSelectorActions>("editorSelector", [
    "addTab",
    "removeTab",
    "setActiveTabId",
  ]);

const enterTabId = ref("");

const handleSelectTab = (tab: TabInfo) => {
  setActiveTabId(tab.id);
};
const handleAddTab = (tab: AnyTabInfo) => {
  addTab(tab);
};
const handleRemoveTab = (tab: TabInfo) => {
  removeTab(tab);
};
</script>

<style scoped>
.query-selector {
  height: var(--tab-height);
  @apply flex box-border;
  @apply text-gray-500;
  @apply text-sm;
}

.query-selector--tab {
  @apply inline-flex place-items-center;
  @apply cursor-pointer box-border;
  @apply px-2 border-r;
  @apply bg-gray-50;
}

.query-selector--tab.active {
  @apply cursor-text relative;
  @apply bg-white;
  @apply text-accent;
}

.query-selector--tab .label {
  @apply p-2;
}
.query-selector--tab .suffix {
  @apply flex justify-center items-center h-4 w-4;
}

.query-selector--tab .suffix.close {
  @apply cursor-pointer;
  @apply text-gray-500;
}

.query-selector--added {
  @apply cursor-pointer;
  @apply p-2;
}
</style>
