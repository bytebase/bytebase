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
    @mousedown.left="$emit('select', tab, index)"
    @mouseenter="state.hovering = true"
    @mouseleave="state.hovering = false"
  >
    <div
      class="body flex items-center gap-x-2"
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
import { getConnectionForSQLEditorTab, hexToRgb } from "@/utils";
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
  const { database } = getConnectionForSQLEditorTab(props.tab);
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
  cursor: pointer;
  border-right-width: 1px;
  background-color: white;
  column-gap: 0.5rem;
  position: relative;
}
.hovering {
  background-color: rgb(var(--color-gray-50));
}
.tab-item.admin {
  background-color: rgb(var(--color-dark-bg)) !important;
}

.body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-left: 0.5rem;
  padding-right: 0.25rem;
  border-top-width: 1px;
  padding-top: 4px;
  height: 36px;
}
.current .body {
  position: relative;
  background-color: white;
  border-top-width: 3px;
  padding-top: 2px;
}

.tab-item.admin .body {
  color: rgb(var(--color-matrix-green-hover));
}
.tab-item.admin.current .body {
  background-color: rgb(var(--color-dark-bg)) !important;
  border-top-color: rgb(var(--color-matrix-green-hover));
}
</style>
