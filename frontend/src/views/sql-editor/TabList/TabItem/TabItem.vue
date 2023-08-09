<template>
  <div
    class="tab-item"
    :class="{
      current: isCurrentTab,
      temp: isTempTab(tab),
      hovering: state.hovering,
      admin: tab.mode === TabMode.Admin,
    }"
    @mousedown="$emit('select', tab, index)"
    @mouseenter="state.hovering = true"
    @mouseleave="state.hovering = false"
  >
    <div class="body">
      <Prefix :tab="tab" :index="index" />
      <Label v-if="tab.mode === TabMode.ReadOnly" :tab="tab" :index="index" />
      <AdminLabel v-else :tab="tab" :index="index" />
      <Suffix :tab="tab" :index="index" @close="$emit('close', tab, index)" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, PropType, reactive } from "vue";
import { useTabStore } from "@/store";
import type { TabInfo } from "@/types";
import { TabMode } from "@/types";
import { isTempTab } from "@/utils";
import AdminLabel from "./AdminLabel.vue";
import Label from "./Label.vue";
import Prefix from "./Prefix.vue";
import Suffix from "./Suffix.vue";

type LocalState = {
  hovering: boolean;
};

const props = defineProps({
  tab: {
    type: Object as PropType<TabInfo>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

defineEmits<{
  (e: "select", tab: TabInfo, index: number): void;
  (e: "close", tab: TabInfo, index: number): void;
}>();

const state = reactive<LocalState>({
  hovering: false,
});

const tabStore = useTabStore();
const { currentTabId } = storeToRefs(tabStore);

const isCurrentTab = computed(() => props.tab.id === currentTabId.value);
</script>

<style scoped lang="postcss">
.tab-item {
  @apply cursor-pointer border-r bg-white gap-x-2;
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
  @apply text-[var(--color-matrix-green-hover)];
}
.tab-item.admin.current .body {
  @apply !bg-dark-bg border-[var(--color-matrix-green-hover)];
}
</style>
