<template>
  <div class="flex justify-start items-center">
    <NTooltip v-if="!dropped" trigger="hover" to="body">
      <template #trigger>
        <heroicons:trash
          class="w-4 h-auto text-gray-500 cursor-pointer hover:opacity-80"
          @click="$emit('drop')"
        />
      </template>
      <span>{{ $t("schema-editor.actions.drop-table") }}</span>
    </NTooltip>
    <NTooltip v-else trigger="hover" to="body">
      <template #trigger>
        <heroicons:arrow-uturn-left
          class="w-4 h-auto text-gray-500 cursor-pointer hover:opacity-80"
          @click="$emit('restore')"
        />
      </template>
      <span>{{ $t("schema-editor.actions.restore") }}</span>
    </NTooltip>
  </div>
</template>
<script lang="ts" setup>
import { Table } from "@/types/v1/schemaEditor";

defineProps<{
  table: Table;
  dropped?: boolean;
}>();
defineEmits<{
  (event: "drop"): void;
  (event: "restore"): void;
}>();
</script>
