<template>
  <div class="flex flex-row flex-wrap items-center break-all">
    <span v-if="hasForeignKey" @click="$emit('click')">
      {{ referencedForeignKeyName }}
    </span>
    <span v-else class="italic text-control-placeholder">EMPTY</span>

    <MiniActionButton
      v-if="!readonly"
      :disabled="disabled"
      @click="$emit('edit')"
    >
      <PenSquareIcon class="w-4 h-4" />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PenSquareIcon } from "lucide-vue-next";
import { MiniActionButton } from "@/components/v2";
import { Column } from "@/types/v1/schemaEditor";

defineProps<{
  column: Column;
  readonly?: boolean;
  disabled: boolean;
  hasForeignKey: boolean;
  referencedForeignKeyName: string;
}>();
defineEmits<{
  (event: "click"): void;
  (event: "edit"): void;
}>();
</script>
