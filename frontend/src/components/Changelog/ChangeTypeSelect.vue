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
import { Changelog_Type } from "@/types/proto-es/v1/database_service_pb";

defineProps<{
  changeType?: Changelog_Type;
}>();

const emit = defineEmits<{
  (event: "update:change-type", changeType?: Changelog_Type): void;
}>();

const options = computed(() => {
  return [
    {
      value: Changelog_Type.MIGRATE,
      label: "DDL",
    },
    {
      value: Changelog_Type.DATA,
      label: "DML",
    },
  ];
});

const updateSelectedKey = (value: Changelog_Type) => {
  emit("update:change-type", value);
};
</script>
