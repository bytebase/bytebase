<template>
  <DropdownInput
    :value="column.type || null"
    :allow-input-value="allowInputValue"
    :options="columnTypeOptions"
    :consistent-menu-width="false"
    placeholder="column type"
    style="--n-padding-left: 6px; --n-padding-right: 22px; --n-font-size: 14px"
    class="bb-schema-editor--column-data-type-select"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { SelectOption } from "naive-ui";
import { computed } from "vue";
import { DropdownInput } from "@/components/v2";
import { Engine } from "@/types/proto/v1/common";
import { Column } from "@/types/v1/schemaEditor";
import { getDataTypeSuggestionList } from "@/utils";

const props = defineProps<{
  column: Column;
  readonly?: boolean;
  engine: Engine;
  schemaTemplateColumnTypes: string[];
}>();
defineEmits<{
  (event: "update:value", value: string): void;
}>();

const allowInputValue = computed(() => {
  return props.schemaTemplateColumnTypes.length === 0;
});

const columnTypeOptions = computed(() => {
  const { schemaTemplateColumnTypes, engine } = props;
  if (allowInputValue.value) {
    return getDataTypeSuggestionList(engine).map<SelectOption>((dataType) => {
      return {
        value: dataType,
        label: dataType,
      };
    });
  }
  return schemaTemplateColumnTypes.map<SelectOption>((columnType) => {
    return {
      value: columnType,
      label: columnType,
    };
  });
});
</script>

<style lang="postcss" scoped>
.bb-schema-editor--column-data-type-select :deep(.n-base-selection) {
  --n-padding-single: 0 22px 0 6px !important;
}
</style>
