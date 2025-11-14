<template>
  <div
    v-if="scopeOption"
    v-show="show"
    ref="containerRef"
    class="flex flex-col overflow-hidden"
    :data-top="containerTop"
  >
    <div class="px-3 py-2">
      <div
        ref="titleRef"
        class="text-sm text-control font-semibold"
        :data-height="titleHeight"
      >
        {{ scopeOption.title }}
      </div>
      <div v-if="scopeOption.search" class="textinfolabel">
        {{ $t("issue.advanced-search.search") }}
      </div>
      <div v-else class="textinfolabel">{{ scopeOption.description }}</div>
    </div>
    <div v-if="valueOptions.length > 0">
      <NVirtualList
        ref="virtualListRef"
        :items="valueOptions"
        :key-field="`value`"
        :item-resizable="false"
        :item-size="38"
        :style="{
          'max-height': `${maxListHeight}px`,
        }"
      >
        <template
          #default="{
            item: option,
            index,
          }: {
            item: ValueOption;
            index: number;
          }"
        >
          <div class="border-t border-block-border">
            <div
              class="h-[38px] flex gap-x-2 px-3 items-center cursor-pointer overflow-hidden"
              :class="[index === menuIndex && 'bg-gray-200/75']"
              :data-index="index"
              :data-value="option.value"
              @mouseenter.prevent.stop="$emit('hover-item', index)"
              @mousedown.prevent.stop="$emit('select-value', option.value)"
            >
              <template v-if="option.render">
                <component :is="option.render" class="text-control text-sm" />
              </template>
              <span v-if="!option.custom" class="text-control-light text-sm">
                {{ option.value }}
              </span>
            </div>

            <div
              v-if="
                !!fetchState?.nextPageToken && index === valueOptions.length - 1
              "
              class="py-2 px-1 border-t border-block-border"
            >
              <NButton
                quaternary
                :size="'small'"
                :loading="fetchState?.loading"
                @click="() => $emit('fetch-next-page')"
              >
                <span class="textinfolabel">
                  {{ $t("common.load-more") }}
                </span>
              </NButton>
            </div>
          </div>
        </template>
      </NVirtualList>
    </div>
    <div v-else-if="showEmptyPlaceholder" class="pb-2">
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
import { NButton, NEmpty, NVirtualList } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import type { ScopeOption, ValueOption } from "./types";

const props = defineProps<{
  show: boolean;
  scopeOption?: ScopeOption;
  valueOptions: ValueOption[];
  menuIndex: number;
  showEmptyPlaceholder?: boolean;
  fetchState?: {
    loading: boolean;
    nextPageToken?: string;
  };
}>();

defineEmits<{
  (event: "select-value", value: string): void;
  (event: "hover-item", index: number): void;
  (event: "fetch-next-page"): void;
}>();

const containerRef = ref<HTMLElement>();
const titleRef = ref<HTMLElement>();
const virtualListRef = ref<InstanceType<typeof NVirtualList>>();
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

const highlightedItem = computed(() => {
  if (!props.show) return undefined;
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
