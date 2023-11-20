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
import { MiniActionButton } from "@/components/v2";
import { SemanticTypeSetting_SemanticType as SemanticType } from "@/types/proto/v1/setting_service";
import { Column } from "@/types/v1/schemaEditor";

const props = defineProps<{
  column: Column;
  readonly?: boolean;
  disabled?: boolean;
  semanticTypeList: SemanticType[];
  disableAlterColumn: (column: Column) => boolean;
}>();
defineEmits<{
  (event: "edit"): void;
  (event: "remove"): void;
}>();

const semanticType = computed(() => {
  const { column, semanticTypeList } = props;
  if (!column.config.semanticTypeId) {
    return;
  }
  return semanticTypeList.find(
    (data) => data.id === column.config.semanticTypeId
  );
});
</script>
