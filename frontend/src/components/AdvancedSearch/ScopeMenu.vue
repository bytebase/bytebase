<template>
  <div v-if="options.length > 0" ref="scrollbarContainerRef" v-show="show">
    <NScrollbar
      ref="scrollbarRef"
      :style="{
        'max-height': `${maxListHeight}px`,
      }"
    >
      <div
        v-for="option, index in options"
        :key="`${option.id}-${index}`"
        class="flex gap-x-2 gap-y-0.5 px-3 py-2 cursor-pointer border-t text-sm"
        :class="[
          index === menuIndex && 'bg-gray-200/75',
          index === 0 ? 'border-transparent' : 'border-block-border',
          compactSize ? 'flex-col items-start' : 'flex-row items-center'
        ]"
        :data-index="index"
        :data-id="option.id"
        @mouseenter.prevent.stop="$emit('hover-item', index)"
        @mousedown.prevent.stop="$emit('select-scope', option.id)"
      >
        <span class="text-accent">
          {{ option.id }}
        </span>
        <NEllipsis>
          <span class="text-control-light">{{ option.description }}</span>
        </NEllipsis>
      </div>
    </NScrollbar>
  </div>
</template>

<script setup lang="ts">
import {
  useElementBounding,
  useElementSize,
  useWindowSize,
} from "@vueuse/core";
import { NEllipsis, NScrollbar } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import type { SearchScopeId } from "@/utils";
import type { ScopeOption } from "./types";

const props = defineProps<{
  show: boolean;
  options: ScopeOption[];
  menuIndex: number;
}>();

defineEmits<{
  (event: "select-scope", id: SearchScopeId): void;
  (event: "hover-item", index: number): void;
}>();

const scrollbarRef = ref<InstanceType<typeof NScrollbar>>();
const scrollbarContainerRef = ref<HTMLElement>();
const { top: containerTop } = useElementBounding(scrollbarContainerRef);
const { height: windowHeight } = useWindowSize();
const { width: scrollbarWidth } = useElementSize(scrollbarContainerRef);

const compactSize = computed(() => scrollbarWidth.value <= 550);

const maxListHeight = computed(() => {
  const MAX_HEIGHT = 480;
  const PADDING_BOTTOM = 16;

  return Math.min(
    MAX_HEIGHT,
    windowHeight.value - containerTop.value - PADDING_BOTTOM
  );
});

const highlightedIndex = computed(() => {
  if (!props.show) return undefined;
  const { options, menuIndex } = props;
  if (options[menuIndex]) {
    return menuIndex;
  }
  return undefined;
});

watch(
  [highlightedIndex, () => props.show],
  ([index, show]) => {
    if (!show) return;
    if (index === undefined) return;
    nextTick(() => {
      scrollbarRef.value?.scrollTo({
        top: index * (compactSize.value ? 59 : 38),
        behavior: "auto",
      });
    });
  },
  { immediate: true }
);
</script>
