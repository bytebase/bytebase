<template>
  <ClassificationLevelBadge
    :classification="table.classification"
    :classification-config="classificationConfig"
  />
  <div v-if="!readonly && !disableChangeTable(table)" class="flex">
    <button
      v-if="table.classification"
      class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
      @click.prevent="$emit('remove')"
    >
      <heroicons-outline:x class="w-3 h-3" />
    </button>
    <button
      class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
      @click.prevent="$emit('edit')"
    >
      <heroicons-outline:pencil class="w-3 h-3" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { Table } from "@/types/v1/schemaEditor";
import { disableChangeTable } from "../common";

defineProps<{
  table: Table;
  readonly?: boolean;
  classificationConfig: DataClassificationConfig;
}>();
defineEmits<{
  (event: "edit"): void;
  (event: "remove"): void;
}>();
</script>
