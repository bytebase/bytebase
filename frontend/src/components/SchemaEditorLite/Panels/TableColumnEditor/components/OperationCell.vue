<template>
  <div class="flex justify-start items-center">
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
import { TrashIcon, Undo2Icon } from "lucide-vue-next";
import { MiniActionButton } from "@/components/v2";
import { ColumnMetadata } from "@/types/proto/v1/database_service";

defineProps<{
  column: ColumnMetadata;
  dropped?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "drop"): void;
  (event: "restore"): void;
}>();
</script>
