<template>
  <NSelect
    v-bind="$attrs"
    :value="value"
    :multiple="true"
    :clearable="true"
    :options="options"
    :max-tag-count="'responsive'"
    :filterable="true"
    :placeholder="$t('schema-editor.schema.select')"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="tsx" setup>
import { NSelect } from "naive-ui";
import { computed } from "vue";
import type { SchemaMetadata } from "@/types/proto/v1/database_service";

const props = defineProps<{
  value: string[];
  schemas: SchemaMetadata[];
}>();

defineEmits<{
  (event: "update:value", val: string[]): void;
}>();

const options = computed(() => {
  return props.schemas.map((schema) => {
    return {
      value: schema.name,
      label: schema.name,
    };
  });
});
</script>
