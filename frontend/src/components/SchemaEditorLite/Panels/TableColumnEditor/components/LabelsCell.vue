<template>
  <div class="flex justify-start items-center">
    <LabelsCell :labels="columnCatalog?.labels ?? {}" :show-count="1" />
    <MiniActionButton v-if="!readonly && !disabled" @click="$emit('edit')">
      <PencilIcon class="w-3 h-3" />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import { MiniActionButton } from "@/components/v2";
import { LabelsCell } from "@/components/v2/Model/cells";

const props = defineProps<{
  database: string;
  schema: string;
  table: string;
  column: string;
  readonly?: boolean;
  disabled?: boolean;
}>();
defineEmits<{
  (event: "edit"): void;
}>();

const { getColumnCatalog } = useSchemaEditorContext();

const columnCatalog = computed(() => {
  return getColumnCatalog({
    database: props.database,
    schema: props.schema,
    table: props.table,
    column: props.column,
  });
});
</script>
