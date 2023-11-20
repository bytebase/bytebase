<template>
  <div class="flex flex-row items-center">
    <ClassificationLevelBadge
      :classification="column.classification"
      :classification-config="classificationConfig"
    />
    <template v-if="!readonly && !disabled">
      <MiniActionButton
        v-if="column.classification"
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
import { PencilIcon, XIcon } from "lucide-vue-next";
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import { MiniActionButton } from "@/components/v2";
import { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { Column } from "@/types/v1/schemaEditor";

defineProps<{
  column: Column;
  readonly?: boolean;
  disabled?: boolean;
  classificationConfig: DataClassificationConfig;
}>();
defineEmits<{
  (event: "edit"): void;
  (event: "remove"): void;
}>();
</script>
