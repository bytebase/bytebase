<template>
  <div class="flex justify-start items-center">
    <NPopconfirm @positive-click="$emit('drop')">
      <template #trigger>
        <MiniActionButton tag="div">
          <TrashIcon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <template #default>
        {{ $t("branch.table-partition.drop-partition-conform") }}
      </template>
    </NPopconfirm>
  </div>
</template>
<script lang="ts" setup>
import { TrashIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { MiniActionButton } from "@/components/v2";
import type { TablePartitionMetadata } from "@/types/proto/v1/database_service";

defineProps<{
  partition: TablePartitionMetadata;
}>();
defineEmits<{
  (event: "drop"): void;
}>();
</script>
