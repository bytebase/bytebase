<template>
  <div class="flex flex-row items-center">
    <ClassificationLevelBadge
      :classification="table.classification"
      :classification-config="classificationConfig"
    />
    <template v-if="!readonly && !disabled">
      <MiniActionButton
        v-if="table.classification"
        @click.prevent="$emit('remove')"
      >
        <XIcon class="w-3 h-3" />
      </MiniActionButton>
      <MiniActionButton
        class="w-4 h-4 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer"
        @click.prevent="$emit('edit')"
      >
        <PencilIcon class="w-3 h-3" />
      </MiniActionButton>
    </template>
  </div>
</template>

<script lang="ts" setup>
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { Table } from "@/types/v1/schemaEditor";

defineProps<{
  table: Table;
  readonly?: boolean;
  disabled?: boolean;
  classificationConfig: DataClassificationConfig;
}>();
defineEmits<{
  (event: "edit"): void;
  (event: "remove"): void;
}>();
</script>
