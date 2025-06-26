<template>
  <div class="flex flex-row flex-wrap whitespace-nowrap">
    <span v-if="semanticType?.title">{{ semanticType?.title }}</span>
    <span v-else class="text-control-placeholder italic">N/A</span>
    <template v-if="!readonly">
      <MiniActionButton
        v-if="semanticType"
        :disabled="disabled"
        @click.prevent="$emit('remove')"
      >
        <XIcon class="w-3 h-3" />
      </MiniActionButton>
      <MiniActionButton :disabled="disabled" @click.prevent="$emit('edit')">
        <PencilIcon class="w-3 h-3" />
      </MiniActionButton>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, XIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import { MiniActionButton } from "@/components/v2";
import type { SemanticTypeSetting_SemanticType as SemanticType } from "@/types/proto-es/v1/setting_service_pb";

const props = defineProps<{
  database: string;
  schema: string;
  table: string;
  column: string;
  readonly?: boolean;
  disabled: boolean;
  semanticTypeList: SemanticType[];
}>();

defineEmits<{
  (event: "edit"): void;
  (event: "remove"): void;
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

const semanticType = computed(() => {
  const { semanticTypeList } = props;
  const catalog = columnCatalog.value;
  if (!catalog?.semanticType) {
    return;
  }
  return semanticTypeList.find((data) => data.id === catalog.semanticType);
});
</script>
