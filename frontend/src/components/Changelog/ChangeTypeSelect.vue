<template>
  <NSelect
    clearable
    :value="changeType"
    :options="options"
    :placeholder="$t('issue.advanced-search.scope.type.title')"
    @update:value="updateSelectedKey"
  />
</template>

<script setup lang="tsx">
import { NSelect } from "naive-ui";
import { computed } from "vue";

defineProps<{
  changeType?: string;
}>();

const emit = defineEmits<{
  (event: "update:change-type", changeType?: string): void;
}>();

const options = computed(() => {
  return [
    {
      value: "MIGRATE",
      label: "DDL",
    },
    {
      value: "DATA",
      label: "DML",
    },
  ];
});

const updateSelectedKey = (value: string) => {
  emit("update:change-type", value);
};
</script>
