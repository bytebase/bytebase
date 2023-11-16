<template>
  <div v-if="scopeOption" class="flex flex-col">
    <div
      v-if="scopeOption.title"
      class="px-3 pt-2 pb-1 text-sm text-control font-semibold"
    >
      {{ scopeOption.title }}
    </div>
    <div v-if="valueOptions.length > 0" class="max-h-60 overflow-hidden">
      <VirtualList
        ref="virtualListRef"
        :items="valueOptions"
        :key-field="`value`"
        :item-resizable="false"
        :item-size="28"
      >
        <template #default="{ item: option, index }: ListItem">
          <div
            class="h-[28px] flex gap-x-2 px-3 py-1 items-center cursor-pointer hover:bg-gray-100 border-t border-block-border"
            :class="[index === menuIndex && 'bg-gray-100']"
            :data-index="index"
            :data-value="option.value"
            @mousedown.prevent.stop="onValueSelect(option.value)"
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
import { NEmpty } from "naive-ui";
import { computed, ref, watch } from "vue";
import { VirtualList } from "vueuc";
import { SearchParams } from "@/utils";
import { ScopeOption, ValueOption } from "./useSearchScopeOptions";

type ListItem = {
  item: ValueOption;
  index: number;
};

const props = defineProps<{
  params: SearchParams;
  scopeOption?: ScopeOption;
  valueOptions: ValueOption[];
  menuIndex: number;
}>();
const emit = defineEmits<{
  (event: "select-value", value: string): void;
}>();

const virtualListRef = ref<InstanceType<typeof VirtualList>>();

const onValueSelect = (value: string) => {
  emit("select-value", value);
};

const highlightedItem = computed((): ListItem | undefined => {
  const options = props.valueOptions;
  const index = props.menuIndex;
  const item = options[index];
  if (item) {
    return { item, index };
  }
  return undefined;
});

watch(
  highlightedItem,
  (item) => {
    if (!item) return;
    const virtualList = virtualListRef.value;
    if (!virtualList) return;
    virtualList.scrollTo({
      index: item.index,
    });
  },
  { immediate: true }
);
</script>
