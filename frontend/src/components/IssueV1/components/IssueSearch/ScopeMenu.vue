<template>
  <div
    v-if="options.length > 0"
    v-show="show"
    class="overflow-hidden"
    :style="{
      'max-height': `${maxListHeight}px`,
    }"
  >
    <VirtualList
      ref="virtualListRef"
      :items="options"
      :key-field="`id`"
      :item-resizable="false"
      :item-size="32"
    >
      <template #default="{ item: option, index }: ListItem">
        <div
          class="h-[32px] flex gap-x-1 px-3 items-center cursor-pointer border-t text-sm"
          :class="[
            index === menuIndex && 'bg-gray-100',
            index === 0 ? 'border-transparent' : 'border-block-border',
          ]"
          :data-index="index"
          :data-id="option.id"
          @mouseenter.prevent.stop="$emit('hover-item', index)"
          @mousedown.prevent.stop="$emit('select-scope', option.id)"
        >
          <span class="text-accent">{{ option.id }}:</span>
          <span class="text-control-light">{{ option.description }}</span>
        </div>
      </template>
    </VirtualList>
  </div>
</template>

<script setup lang="ts">
import { useElementBounding, useWindowSize } from "@vueuse/core";
import { computed, nextTick, ref, watch } from "vue";
import { VirtualList } from "vueuc";
import { SearchParams, SearchScopeId } from "@/utils";
import { ScopeOption } from "./useSearchScopeOptions";

type ListItem = {
  item: ScopeOption;
  index: number;
};

const props = defineProps<{
  show: boolean;
  params: SearchParams;
  options: ScopeOption[];
  menuIndex: number;
}>();
defineEmits<{
  (event: "select-scope", id: SearchScopeId): void;
  (event: "hover-item", index: number): void;
}>();

const containerRef = ref<HTMLElement>();
const virtualListRef = ref<InstanceType<typeof VirtualList>>();
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

const highlightedItem = computed((): ListItem | undefined => {
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
