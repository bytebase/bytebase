<template>
  <BBCheckbox
    :title="title"
    :value="value"
    :disabled="disabled || !editable"
    @toggle="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBCheckbox } from "@/bbkit";
import type { RuleConfigComponent, RuleTemplate } from "@/types";
import { getRuleLocalizationKey } from "@/types/sqlReview";

const props = defineProps<{
  rule: RuleTemplate;
  config: RuleConfigComponent;
  value: boolean;
  disabled: boolean;
  editable: boolean;
}>();

defineEmits<{
  (event: "update:value", value: boolean): void;
}>();

const { t } = useI18n();

const title = computed(() => {
  const { rule, config } = props;
  const key = `sql-review.rule.${getRuleLocalizationKey(rule.type)}.component.${
    config.key
  }.title`;
  return t(key);
});
</script>
