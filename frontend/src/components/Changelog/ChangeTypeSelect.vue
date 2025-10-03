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
import { Changelog_MigrationType } from "@/types/proto-es/v1/database_service_pb";

defineProps<{
  changeType?: Changelog_MigrationType;
}>();

const emit = defineEmits<{
  (event: "update:change-type", changeType?: Changelog_MigrationType): void;
}>();

const options = computed(() => {
  return [
    {
      value: Changelog_MigrationType.DDL,
      label: "DDL",
    },
    {
      value: Changelog_MigrationType.DML,
      label: "DML",
    },
  ];
});

const updateSelectedKey = (value: Changelog_MigrationType) => {
  emit("update:change-type", value);
};
</script>
