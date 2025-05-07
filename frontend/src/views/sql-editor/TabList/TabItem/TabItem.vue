<template>
  <div
    class="tab-item"
    :class="[
      {
        current: isCurrentTab,
        hovering: state.hovering,
      },
      tab.status.toLowerCase(),
    ]"
    :data-status="tab.status"
    :data-sheet="tab.worksheet"
    :data-connection="JSON.stringify(tab.connection)"
    @mousedown.left="$emit('select', tab, index)"
    @mouseenter="state.hovering = true"
    @mouseleave="state.hovering = false"
  >
    <div
      class="body"
      :style="
        backgroundColorRgb
          ? {
              backgroundColor: `rgba(${backgroundColorRgb}, 0.1)`,
              borderTopColor: `rgb(${backgroundColorRgb})`,
              color: `rgb(${backgroundColorRgb})`,
            }
          : {}
      "
    >
      <Prefix :tab="tab" />
      <Label v-if="tab.mode === 'WORKSHEET'" :tab="tab" />
      <AdminLabel v-else :tab="tab" />
      <Suffix :tab="tab" @close="$emit('close', tab, index)" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useSQLEditorTabStore } from "@/store";
import { type SQLEditorTab, UNKNOWN_ID } from "@/types";
import { connectionForSQLEditorTab, hexToRgb } from "@/utils";
import AdminLabel from "./AdminLabel.vue";
import Label from "./Label.vue";
import Prefix from "./Prefix.vue";
import Suffix from "./Suffix.vue";

type LocalState = {
  hovering: boolean;
};

const props = defineProps<{
  tab: SQLEditorTab;
  index: number;
}>();

defineEmits<{
  (e: "select", tab: SQLEditorTab, index: number): void;
  (e: "close", tab: SQLEditorTab, index: number): void;
}>();

const state = reactive<LocalState>({
  hovering: false,
});

const tabStore = useSQLEditorTabStore();

const isCurrentTab = computed(() => props.tab.id === tabStore.currentTabId);

const environment = computed(() => {
  const { database } = connectionForSQLEditorTab(props.tab);
  const environment = database?.effectiveEnvironmentEntity;
  if (environment?.id === String(UNKNOWN_ID)) {
    return;
  }
  return environment;
});

const backgroundColorRgb = computed(() => {
  if (!isCurrentTab.value) {
    return "";
  }
  if (!environment.value || !environment.value.color) {
    return hexToRgb("#4f46e5").join(", ");
  }
  return hexToRgb(environment.value.color).join(", ");
});
</script>

<style scoped lang="postcss">
.tab-item {
  @apply cursor-pointer border-r bg-white gap-x-2 relative;
}
.hovering {
  @apply bg-gray-50;
}
.tab-item.admin {
  @apply !bg-dark-bg;
}

.body {
  @apply flex items-center justify-between gap-x-1 pl-2 pr-1 border-t pt-[4px] h-[36px];
}
.current .body {
  @apply relative bg-white border-t-[3px] pt-[2px];
}

.tab-item.admin .body {
  @apply text-matrix-green-hover;
}
.tab-item.admin.current .body {
  @apply !bg-dark-bg border-matrix-green-hover;
}
</style>
