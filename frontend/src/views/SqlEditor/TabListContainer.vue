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
          v-for="tab in tabStore.tabList"
          :key="tab.id"
          class="tab-list-tab"
          :class="{ active: tab.id === tabStore.currentTabId }"
          :style="scrollState.style"
          @click="handleSelectTab(tab)"
          @mouseover="enterTabId = tab.id"
          @mouseleave="enterTabId = ''"
        >
          <div
            class="label max-w-5xl w-48 truncate"
            @dblclick="handleEditLabel(tab)"
          >
            <div
              v-if="labelState.editingTabId === tab.id"
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
            <span v-else class="flex items-center space-x-2">
              <heroicons-outline:user-group
                v-if="sheet.visibility === 'PROJECT'"
                class="w-4 h-4"
              />
              <heroicons-outline:globe
                v-if="sheet.visibility === 'PUBLIC'"
                class="w-4 h-4"
              />
              <span>{{ tab.name }}</span>
            </span>
          </div>
          <template v-if="enterTabId === tab.id && tabStore.tabList.length > 1">
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
            <template
              v-else-if="
                tab.id === tabStore.currentTabId && tabStore.tabList.length > 1
              "
            >
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
      <button class="p-1 hover:bg-gray-200 rounded-md" @click="handleAddTab">
        <heroicons-solid:plus class="h-4 w-4" />
      </button>
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
import { ref, reactive, nextTick, computed, onMounted, onUnmounted } from "vue";
import { debounce } from "lodash-es";
import { useI18n } from "vue-i18n";
import { useDialog } from "naive-ui";

import { pushNotification, useTabStore, useSheetStore } from "@/store";
import { TabInfo } from "@/types";
import { useSQLEditorConnection } from "@/composables/useSQLEditorConnection";

const tabStore = useTabStore();
const sheetStore = useSheetStore();

const sheet = computed(() => sheetStore.currentSheet);

const { t } = useI18n();
const { setConnectionContextFromCurrentTab } = useSQLEditorConnection();
const dialog = useDialog();

const enterTabId = ref("");
const selectedTab = computed(() => tabStore.currentTabId);
// edit label state
const labelState = reactive({
  currentLabelName: "",
  editingTabId: "",
});
const labelInputRef = ref<HTMLInputElement>();

const localTabList = computed(() => {
  return tabStore.tabList.map((tab: TabInfo) => {
    return {
      label: tab.name,
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

const recalculateScrollWidth = () => {
  scrollState.scrollWidth = tablistRef.value?.scrollWidth as number;
  scrollState.offsetWidth = tablistRef.value?.offsetWidth as number;
  scrollState.isScrolling = scrollingDistance.value > 0;
};

const updateSheetName = () => {
  if (tabStore.currentTab.sheetId) {
    sheetStore.patchSheetById({
      id: tabStore.currentTab.sheetId,
      name: labelState.currentLabelName,
    });
  }
};

const handleEditLabel = (tab: TabInfo) => {
  labelState.editingTabId = tab.id;
  labelState.currentLabelName = tab.name;
  nextTick(() => {
    labelInputRef.value?.focus();
  });
};

const handleTryChangeLabel = () => {
  if (labelState.editingTabId) {
    if (labelState.currentLabelName !== "") {
      tabStore.updateCurrentTab({
        name: labelState.currentLabelName,
      });

      updateSheetName();
      nextTick(() => {
        recalculateScrollWidth();
        scrollState.style = {
          transform: `translateX(0px)`,
        };
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-editor.please-input-the-tab-label"),
      });
    }
    handleCancelChangeLabel();
  }
};

const handleCancelChangeLabel = () => {
  labelState.editingTabId = "";
  labelState.currentLabelName = "";
};

const handleSelectTab = async (tab: TabInfo) => {
  tabStore.setCurrentTabId(tab.id);
  setConnectionContextFromCurrentTab();
};
const handleAddTab = () => {
  tabStore.addTab();
  nextTick(recalculateScrollWidth);
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
    nextTick(() => {
      recalculateScrollWidth();
    });
  }
};
const handleSelectTabFromPopselect = (tabId: string) => {
  tabStore.setCurrentTabId(tabId);
  setConnectionContextFromCurrentTab();
};

const handleScollTabList = debounce((e: WheelEvent) => {
  console.log(e);
  console.log(e.deltaX > 0 ? "Move to right" : "Move to left");
  console.log(e.offsetX);
}, 333);

onMounted(async () => {
  if (!tabStore.hasTabs) {
    handleAddTab();
  }
});

// add listener to confirm confrim if close the tab.
onMounted(() => {
  window.onbeforeunload = () => {
    return "false";
  };
});
// remove if unmount view
onUnmounted(() => {
  window.onbeforeunload = null;
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
