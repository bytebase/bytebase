<template>
  <div
    class="tab-item"
    :class="[
      {
        current: isCurrentTab,
        hovering: state.hovering,
        admin: tab.mode === 'ADMIN',
      },
      tab.status.toLowerCase(),
    ]"
    :data-status="tab.status"
    :data-sheet="tab.sheet"
    :data-connection="JSON.stringify(tab.connection)"
    @mousedown.left="$emit('select', tab, index)"
    @mouseenter="state.hovering = true"
    @mouseleave="state.hovering = false"
  >
    <div class="body">
      <Prefix :tab="tab" :index="index" />
      <Label
        v-if="tab.mode === 'READONLY' || tab.mode === 'STANDARD'"
        :tab="tab"
        :index="index"
      />
      <AdminLabel v-else :tab="tab" :index="index" />
      <Suffix :tab="tab" :index="index" @close="$emit('close', tab, index)" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import AdminLabel from "./AdminLabel.vue";
import Label from "./Label.vue";
import Prefix from "./Prefix.vue";
import Suffix from "./Suffix.vue";

type LocalState = {
  hovering: boolean;
};

const props = defineProps({
  tab: {
    type: Object as PropType<SQLEditorTab>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

defineEmits<{
  (e: "select", tab: SQLEditorTab, index: number): void;
  (e: "close", tab: SQLEditorTab, index: number): void;
}>();

const state = reactive<LocalState>({
  hovering: false,
});

const tabStore = useSQLEditorTabStore();

const isCurrentTab = computed(() => props.tab.id === tabStore.currentTabId);
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
  @apply flex items-center justify-between gap-x-1 pl-2 pr-1 pb-1 border-t pt-[4px];
}
.current .body {
  @apply relative bg-white text-accent border-t-accent border-t-2 pt-[3px];
}

.tab-item.admin .body {
  @apply text-matrix-green-hover;
}
.tab-item.admin.current .body {
  @apply !bg-dark-bg border-matrix-green-hover;
}
</style>
