<template>
  <div class="flex justify-start items-center">
    <template v-if="allowReorder">
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
    </template>
    <NTooltip v-if="!dropped" trigger="hover" to="body">
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
import {
  ChevronDownIcon,
  ChevronUpIcon,
  TrashIcon,
  Undo2Icon,
} from "lucide-vue-next";
import { MiniActionButton } from "@/components/v2";
import { Column } from "@/types/v1/schemaEditor";

defineProps<{
  column: Column;
  allowReorder?: boolean;
  allowMoveUp?: boolean;
  allowMoveDown?: boolean;
  dropped?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "drop"): void;
  (event: "restore"): void;
  (event: "reorder", delta: -1 | 1): void;
}>();
</script>
