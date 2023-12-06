<template>
  <div class="flex justify-start items-center">
    <LabelsColumn :labels="columnConfig?.labels ?? {}" :show-count="1" />
    <MiniActionButton v-if="!readonly && !disabled" @click="$emit('edit')">
      <PencilIcon class="w-3 h-3" />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import { TableTabContext } from "@/components/SchemaEditorLite/types";
import { MiniActionButton } from "@/components/v2";
import LabelsColumn from "@/components/v2/Model/DatabaseV1Table/LabelsColumn.vue";
import { ColumnMetadata } from "@/types/proto/v1/database_service";

const props = defineProps<{
  column: ColumnMetadata;
  readonly?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "edit"): void;
}>();

const { currentTab, getColumnConfig } = useSchemaEditorContext();

const columnConfig = computed(() => {
  const tab = currentTab.value as TableTabContext;
  return getColumnConfig(tab.database, {
    ...tab.metadata,
    column: props.column,
  });
});
</script>
