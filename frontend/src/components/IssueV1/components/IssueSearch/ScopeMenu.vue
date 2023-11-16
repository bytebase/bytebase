<template>
  <div class="flex flex-col">
    <div
      v-for="(option, index) in options"
      :key="option.id"
      class="flex gap-x-1 px-3 py-1 items-center cursor-pointer hover:bg-gray-100"
      :class="index === menuIndex && 'bg-gray-100'"
      @mousedown.prevent.stop="onScopeSelect(option.id)"
    >
      <div class="space-x-1 text-sm">
        <span class="text-accent">{{ option.id }}:</span>
        <span class="text-control-light">{{ option.description }}</span>
      </div>
    </div>
    <div v-if="options.length === 0" class="px-3 py-1 text-control text-sm">
      N/A
    </div>
  </div>
</template>

<script setup lang="ts">
import { SearchParams, SearchScopeId } from "@/utils";
import { ScopeOption } from "./useSearchScopeOptions";

defineProps<{
  inputText: string;
  params: SearchParams;
  options: ScopeOption[];
  menuIndex: number;
}>();
const emit = defineEmits<{
  (event: "select-scope", id: SearchScopeId): void;
}>();

const onScopeSelect = (id: SearchScopeId) => {
  emit("select-scope", id);
};
</script>
