<template>
  <div
    v-if="scopeOption"
    v-show="show"
    ref="containerRef"
    class="flex flex-col overflow-hidden"
    :data-top="containerTop"
  >
    <div
      v-if="scopeOption.title"
      ref="titleRef"
      class="px-3 py-2 text-sm text-control font-semibold"
      :data-height="titleHeight"
    >
      {{ scopeOption.title }}
    </div>
    <div
      v-if="valueOptions.length > 0"
      class="flex-1 overflow-hidden"
      :style="{
        'max-height': `${maxListHeight}px`,
      }"
    >
      <VirtualList
        ref="virtualListRef"
        :items="valueOptions"
        :key-field="`value`"
        :item-resizable="false"
        :item-size="32"
      >
        <template #default="{ item: option, index }: ListItem">
          <div
            class="h-[32px] flex gap-x-2 px-3 items-center cursor-pointer border-t border-block-border"
            :class="[index === menuIndex && 'bg-gray-100']"
            :data-index="index"
            :data-value="option.value"
            @mouseenter.prevent.stop="$emit('hover-item', index)"
            @mousedown.prevent.stop="$emit('select-value', option.value)"
          >
            <component :is="option.render" class="text-control text-sm" />
            <span class="text-control-light text-sm">{{ option.value }}</span>
          </div>
        </template>
      </VirtualList>
    </div>
    <div v-if="valueOptions.length === 0" class="pb-2">
      <NEmpty />
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  useElementBounding,
  useElementSize,
  useWindowSize,
} from "@vueuse/core";
import { NEmpty } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import { VirtualList } from "vueuc";
import { SearchParams } from "@/utils";
import { ScopeOption, ValueOption } from "./useSearchScopeOptions";

type ListItem = {
  item: ValueOption;
  index: number;
};

const props = defineProps<{
  show: boolean;
  params: SearchParams;
  scopeOption?: ScopeOption;
  valueOptions: ValueOption[];
  menuIndex: number;
}>();
defineEmits<{
  (event: "select-value", value: string): void;
  (event: "hover-item", index: number): void;
}>();

const containerRef = ref<HTMLElement>();
const titleRef = ref<HTMLElement>();
const virtualListRef = ref<InstanceType<typeof VirtualList>>();
const { height: titleHeight } = useElementSize(titleRef, undefined, {
  box: "border-box",
});
const { top: containerTop } = useElementBounding(containerRef);
const { height: windowHeight } = useWindowSize();

const maxListHeight = computed(() => {
  const MAX_HEIGHT = 240;
  const PADDING_BOTTOM = 16;

  return Math.min(
    MAX_HEIGHT,
    windowHeight.value - titleHeight.value - containerTop.value - PADDING_BOTTOM
  );
});

const highlightedItem = computed((): ListItem | undefined => {
  if (props.show) return undefined;
  const options = props.valueOptions;
  const index = props.menuIndex;
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
