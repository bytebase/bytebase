<template>
  <div class="flex justify-start items-center">
    <NPopconfirm v-if="status !== 'dropped'" @positive-click="$emit('drop')">
      <template #trigger>
        <MiniActionButton tag="div" :disabled="readonly">
          <TrashIcon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <template #default>
        <div class="flex flex-col">
          <p>
            {{ $t("branch.table-partition.drop-partition-confirm") }}
          </p>
          <p v-if="partition.subpartitions?.length > 0">
            {{
              $t(
                "branch.table-partition.drop-partition-with-subpartitions-confirm"
              )
            }}
          </p>
        </div>
      </template>
    </NPopconfirm>
    <NTooltip v-else trigger="hover" to="body">
      <template #trigger>
        <MiniActionButton tag="div" @click="$emit('restore')">
          <Undo2Icon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <span>{{ $t("schema-editor.actions.restore") }}</span>
    </NTooltip>
  </div>
</template>
<script lang="ts" setup>
import { TrashIcon, Undo2Icon } from "lucide-vue-next";
import { NPopconfirm, NTooltip } from "naive-ui";
import type { EditStatus } from "@/components/SchemaEditorLite";
import { MiniActionButton } from "@/components/v2";
import type { TablePartitionMetadata } from "@/types/proto/v1/database_service";

defineProps<{
  readonly: boolean;
  partition: TablePartitionMetadata;
  status: EditStatus;
}>();
defineEmits<{
  (event: "drop"): void;
  (event: "restore"): void;
  (event: "add-sub"): void;
}>();
</script>
