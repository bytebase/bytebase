<template>
  <div class="flex justify-start items-center">
    <NPopconfirm v-if="allowDrop" @positive-click="$emit('drop')">
      <template #trigger>
        <MiniActionButton tag="div">
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
    <NTooltip v-if="status === 'dropped'" trigger="hover" to="body">
      <template #trigger>
        <MiniActionButton tag="div" @click="$emit('restore')">
          <Undo2Icon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <span>{{ $t("schema-editor.actions.restore") }}</span>
    </NTooltip>
    <NTooltip v-if="allowAddSub" trigger="hover" to="body">
      <template #trigger>
        <MiniActionButton tag="div" @click="$emit('add-sub')">
          <CirclePlusIcon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <span>{{ $t("schema-editor.table-partition.add-sub-partition") }}</span>
    </NTooltip>
  </div>
</template>
<script lang="ts" setup>
import { CirclePlusIcon, TrashIcon, Undo2Icon } from "lucide-vue-next";
import { NPopconfirm, NTooltip } from "naive-ui";
import { computed } from "vue";
import type { EditStatus } from "@/components/SchemaEditorLite";
import { MiniActionButton } from "@/components/v2";
import type { TablePartitionMetadata } from "@/types/proto-es/v1/database_service_pb";
import { PartitionTypesSupportSubPartition } from "../common";

const props = defineProps<{
  partition: TablePartitionMetadata;
  parent?: TablePartitionMetadata;
  tableStatus: EditStatus;
  status: EditStatus;
}>();
defineEmits<{
  (event: "drop"): void;
  (event: "restore"): void;
  (event: "add-sub"): void;
}>();

const allowDrop = computed(() => {
  return props.tableStatus === "created" || props.status === "created";
});

const allowAddSub = computed(() => {
  const { partition, parent } = props;
  if (parent) {
    // partitions cannot nest (only 1-level parent-sub relationships)
    return false;
  }
  return PartitionTypesSupportSubPartition.includes(partition.type);
});
</script>
