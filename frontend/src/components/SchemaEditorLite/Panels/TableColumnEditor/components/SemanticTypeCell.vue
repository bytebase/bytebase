<template>
  <div class="flex flex-row flex-wrap whitespace-nowrap">
    {{ semanticType?.title }}
    <template v-if="!readonly && !disabled">
      <MiniActionButton
        v-if="semanticType"
        :disabled="disableAlterColumn(column)"
        @click.prevent="$emit('remove')"
      >
        <XIcon class="w-3 h-3" />
      </MiniActionButton>
      <MiniActionButton
        :disabled="disableAlterColumn(column)"
        @click.prevent="$emit('edit')"
      >
        <PencilIcon class="w-3 h-3" />
      </MiniActionButton>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, XIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useSchemaEditorContext } from "@/components/SchemaEditorLite/context";
import { TableTabContext } from "@/components/SchemaEditorLite/types";
import { MiniActionButton } from "@/components/v2";
import { ColumnMetadata } from "@/types/proto/v1/database_service";
import { SemanticTypeSetting_SemanticType as SemanticType } from "@/types/proto/v1/setting_service";

const props = defineProps<{
  column: ColumnMetadata;
  readonly?: boolean;
  disabled?: boolean;
  semanticTypeList: SemanticType[];
  disableAlterColumn: (column: ColumnMetadata) => boolean;
}>();
defineEmits<{
  (event: "edit"): void;
  (event: "remove"): void;
}>();

const { currentTab, getColumnConfig } = useSchemaEditorContext();

const columnConfig = computed(() => {
  const tab = currentTab.value as TableTabContext;
  return getColumnConfig(tab.database, {
    ...tab.metadata,
    column: props.column,
  });
});

const semanticType = computed(() => {
  const { semanticTypeList } = props;
  const config = columnConfig.value;
  if (!config?.semanticTypeId) {
    return;
  }
  return semanticTypeList.find((data) => data.id === config.semanticTypeId);
});
</script>
