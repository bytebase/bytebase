<template>
  <NCheckbox
    :label="title"
    :checked="value"
    :disabled="disabled"
    @update:checked="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { RuleConfigComponent, RuleTemplate } from "@/types";
import { getRuleLocalizationKey } from "@/types/sqlReview";

const props = defineProps<{
  rule: RuleTemplate;
  config: RuleConfigComponent;
  value: boolean;
  disabled: boolean;
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
