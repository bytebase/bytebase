<template>
  <NSelect
    :value="action"
    :options="options"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="$t('settings.sensitive-data.action.select-action')"
    @update:value="$emit('update:action', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MaskingExceptionPolicy_MaskingException_Action as Action } from "@/types/proto-es/v1/org_policy_service_pb";

const props = defineProps<{
  disabled?: boolean;
  clearable?: boolean;
  action?: Action;
  actionList: Action[];
}>();

defineEmits<{
  (event: "update:action", action: Action | undefined): void;
}>();

const { t } = useI18n();

// Helper function to convert Action enum to string
const actionToString = (action: Action): string => {
  switch (action) {
    case Action.ACTION_UNSPECIFIED:
      return "ACTION_UNSPECIFIED";
    case Action.QUERY:
      return "QUERY";
    case Action.EXPORT:
      return "EXPORT";
    default:
      return "UNKNOWN";
  }
};

const options = computed(() => {
  return props.actionList.map<SelectOption>((action) => ({
    label: t(
      `settings.sensitive-data.action.${actionToString(action).toLowerCase()}`
    ),
    value: action,
  }));
});
</script>
