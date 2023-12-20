<template>
  <NSelect
    v-bind="$attrs"
    multiple
    :value="selected"
    :options="options"
    :disabled="disabled"
    max-tag-count="responsive"
    :placeholder="$t('audit-log.table.select-type')"
    @update:value="$emit('update:selected', $event)"
  />
</template>

<script lang="ts" setup>
import { NSelect, SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { AuditActivityTypeI18nNameMap } from "@/types";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";

defineProps<{
  selected: LogEntity_Action[];
  disabled: boolean;
}>();
defineEmits<{
  (event: "update:selected", selected: LogEntity_Action[]): void;
}>();

const { t } = useI18n();

const options = computed(() => {
  const resp: SelectOption[] = [];
  for (const [key, val] of AuditActivityTypeI18nNameMap.entries()) {
    resp.push({
      value: key,
      label: t(val),
    });
  }
  return resp;
});
</script>
