<template>
  <div v-if="scopeOption" class="flex flex-col">
    <div
      v-if="scopeOption.title"
      class="px-3 pt-2 pb-1 text-sm text-control font-semibold"
    >
      {{ scopeOption.title }}
    </div>
    <div class="max-h-60 overflow-y-auto divide-block-border">
      <div
        v-for="(option, index) in valueOptions"
        :key="option.value"
        class="flex gap-x-2 px-3 py-1 items-center cursor-pointer hover:bg-gray-100"
        :class="index === menuIndex && 'bg-gray-100'"
        @mousedown.prevent.stop="onValueSelect(option.value)"
      >
        <component :is="option.render" class="text-control text-sm" />
        <span class="text-control-light text-sm">{{ option.value }}</span>
      </div>
      <div
        v-if="valueOptions.length === 0"
        class="px-3 py-1 text-control text-sm"
      >
        N/A
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { SearchParams } from "@/utils";
import { ScopeOption, ValueOption } from "./useSearchScopeOptions";

defineProps<{
  params: SearchParams;
  scopeOption?: ScopeOption;
  valueOptions: ValueOption[];
  menuIndex: number;
}>();
const emit = defineEmits<{
  (event: "select-value", value: string): void;
}>();

const onValueSelect = (value: string) => {
  emit("select-value", value);
};
</script>
