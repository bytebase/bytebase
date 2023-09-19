<template>
  <BBBadge :text="text" :can-remove="false" :badge-style="style" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBBadgeStyle } from "@/bbkit/BBBadge.vue";
import {
  SQLReviewRuleLevel,
  sQLReviewRuleLevelToJSON,
} from "@/types/proto/v1/org_policy_service";

const props = withDefaults(
  defineProps<{
    level: SQLReviewRuleLevel;
    suffix?: string;
  }>(),
  {
    suffix: "",
  }
);

const { t } = useI18n();

const style = computed((): BBBadgeStyle => {
  switch (props.level) {
    case SQLReviewRuleLevel.ERROR:
      return "CRITICAL";
    case SQLReviewRuleLevel.WARNING:
      return "WARN";
    default:
      return "DISABLED";
  }
});

const text = computed(() => {
  const { level, suffix } = props;
  const parts = [
    t(`sql-review.level.${sQLReviewRuleLevelToJSON(level).toLowerCase()}`),
  ];
  if (suffix) {
    parts.push(suffix);
  }
  return parts.join(" ");
});
</script>
