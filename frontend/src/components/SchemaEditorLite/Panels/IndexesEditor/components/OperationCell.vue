<template>
  <div class="flex justify-start items-center">
    <NPopconfirm @positive-click="$emit('drop')">
      <template #trigger>
        <MiniActionButton tag="div">
          <TrashIcon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <template #default> asdf </template>
    </NPopconfirm>
  </div>
</template>
<script lang="ts" setup>
import { TrashIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { MiniActionButton } from "@/components/v2";
import { IndexMetadata } from "@/types/proto/v1/database_service";

defineProps<{
  index: IndexMetadata;
}>();
defineEmits<{
  (event: "drop"): void;
}>();
</script>
