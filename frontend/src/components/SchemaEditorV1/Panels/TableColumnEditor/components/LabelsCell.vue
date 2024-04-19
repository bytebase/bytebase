<template>
  <div class="flex justify-start items-center">
    <DatabaseLabelsCell :labels="column.config.labels" :show-count="1" />
    <MiniActionButton v-if="!readonly && !disabled" @click="$emit('edit')">
      <PencilIcon class="w-3 h-3" />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { MiniActionButton } from "@/components/v2";
import { DatabaseLabelsCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import type { Column } from "@/types/v1/schemaEditor";

defineProps<{
  column: Column;
  readonly?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "edit"): void;
}>();
</script>
