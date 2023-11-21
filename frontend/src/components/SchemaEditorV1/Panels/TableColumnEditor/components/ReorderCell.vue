<template>
  <div class="flex justify-start items-center">
    <MiniActionButton
      :disabled="disabled || !allowMoveUp"
      @click="$emit('reorder', -1)"
    >
      <ChevronUpIcon class="w-4 h-4" />
    </MiniActionButton>
    <MiniActionButton
      :disabled="disabled || !allowMoveDown"
      @click="$emit('reorder', 1)"
    >
      <ChevronDownIcon class="w-4 h-4" />
    </MiniActionButton>
  </div>
</template>
<script setup lang="ts">
import { ChevronDownIcon, ChevronUpIcon } from "lucide-vue-next";
import { MiniActionButton } from "@/components/v2";
import { Column } from "@/types/v1/schemaEditor";

defineProps<{
  column: Column;
  allowMoveUp?: boolean;
  allowMoveDown?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "reorder", delta: -1 | 1): void;
}>();
</script>
