import { type Ref, computed, ref } from "vue";

const DEFAULT_DISPLAY_COUNT = 10;
export const VIRTUAL_SCROLL_THRESHOLD = 50;

export function useExpandableList<T>(items: Ref<T[]>) {
  const expanded = ref(false);
  const hasMore = computed(() => items.value.length > DEFAULT_DISPLAY_COUNT);
  const visibleItems = computed(() => {
    if (expanded.value || !hasMore.value) return items.value;
    return items.value.slice(0, DEFAULT_DISPLAY_COUNT);
  });
  const remainingCount = computed(
    () => items.value.length - DEFAULT_DISPLAY_COUNT
  );
  return { expanded, hasMore, visibleItems, remainingCount };
}

export { default as ExecutionHistorySection } from "./ExecutionHistorySection.vue";
export { default as LimitsSection } from "./LimitsSection.vue";
export { default as OptionsSection } from "./OptionsSection.vue";
export { default as TargetsSection } from "./TargetsSection.vue";
export { default as TasksSection } from "./TasksSection.vue";
