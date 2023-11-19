<template>
  <div class="flex justify-start items-center">
    <NTooltip v-if="!isDroppedColumn(column)" trigger="hover" to="body">
      <template #trigger>
        <MiniActionButton tag="div" :disabled="disabled" @click="$emit('drop')">
          <TrashIcon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <span>{{ $t("schema-editor.actions.drop-table") }}</span>
    </NTooltip>
    <NTooltip v-else trigger="hover" to="body">
      <template #trigger>
        <MiniActionButton
          tag="div"
          :disabled="disabled"
          @click="$emit('restore')"
        >
          <Undo2Icon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <span>{{ $t("schema-editor.actions.restore") }}</span>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { TrashIcon } from "lucide-vue-next";
import { Undo2Icon } from "lucide-vue-next";
import { MiniActionButton } from "@/components/v2";
import { Column } from "@/types/v1/schemaEditor";
import { isDroppedColumn } from "../common";

defineProps<{
  column: Column;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "drop"): void;
  (event: "restore"): void;
}>();
</script>
