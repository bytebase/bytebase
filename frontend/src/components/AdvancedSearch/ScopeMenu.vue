<template>
  <div v-if="options.length > 0" v-show="show">
    <NVirtualList
      ref="virtualListRef"
      :items="options"
      :key-field="`id`"
      :item-resizable="false"
      :item-size="38"
      :style="{
        'max-height': `${maxListHeight}px`,
      }"
    >
      <template
        #default="{ item: option, index }: { item: ScopeOption; index: number }"
      >
        <div
          class="h-[38px] flex gap-x-1 px-3 items-center cursor-pointer border-t text-sm"
          :class="[
            index === menuIndex && 'bg-gray-200/75',
            index === 0 ? 'border-transparent' : 'border-block-border',
          ]"
          :data-index="index"
          :data-id="option.id"
          @mouseenter.prevent.stop="$emit('hover-item', index)"
          @mousedown.prevent.stop="$emit('select-scope', option.id)"
        >
          <span class="text-accent">
            {{ option.id }}{{ Boolean(option.description) ? ":" : "" }}
          </span>
          <span class="text-control-light">{{ option.description }}</span>
        </div>
      </template>
    </NVirtualList>
  </div>
</template>

<script setup lang="ts">
import { useElementBounding, useWindowSize } from "@vueuse/core";
import { NVirtualList } from "naive-ui";
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

const containerRef = ref<HTMLElement>();
const virtualListRef = ref<InstanceType<typeof NVirtualList>>();
const { top: containerTop } = useElementBounding(containerRef);
const { height: windowHeight } = useWindowSize();

const maxListHeight = computed(() => {
  const MAX_HEIGHT = 480;
  const PADDING_BOTTOM = 16;

  return Math.min(
    MAX_HEIGHT,
    windowHeight.value - containerTop.value - PADDING_BOTTOM
  );
});

const highlightedItem = computed(() => {
  if (!props.show) return undefined;
  const { options, menuIndex: index } = props;
  const item = options[index];
  if (item) {
    return { item, index };
  }
  return undefined;
});

watch(
  [highlightedItem, () => props.show],
  ([item, show]) => {
    if (!show) return;
    if (!item) return;
    nextTick(() => {
      const virtualList = virtualListRef.value;
      if (!virtualList) return;
      virtualList.scrollTo({
        index: item.index,
      });
    });
  },
  { immediate: true }
);
</script>
