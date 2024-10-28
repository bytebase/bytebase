<template>
  <NSelect
    :value="action"
    :options="options"
    :disabled="disabled"
    :clearable="clearable"
    :placeholder="$t('settings.sensitive-data.action.selet-action')"
    @update:value="$emit('update:action', $event)"
  />
</template>

<script lang="ts" setup>
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  type MaskingExceptionPolicy_MaskingException_Action as Action,
  maskingExceptionPolicy_MaskingException_ActionToJSON as actionToJSON,
} from "@/types/proto/v1/org_policy_service";

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

const options = computed(() => {
  return props.actionList.map<SelectOption>((action) => ({
    label: t(
      `settings.sensitive-data.action.${actionToJSON(action).toLowerCase()}`
    ),
    value: action,
  }));
});
</script>
