<template>
  <BBBadge
    :text="text"
    :can-remove="false"
    :badge-style="style"
    :size="'small'"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBBadge } from "@/bbkit";
import type { BBBadgeStyle } from "@/bbkit/BBBadge.vue";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";

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

// Helper function to convert SQLReviewRuleLevel to string
const sqlReviewRuleLevelToString = (level: SQLReviewRuleLevel): string => {
  switch (level) {
    case SQLReviewRuleLevel.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case SQLReviewRuleLevel.ERROR:
      return "ERROR";
    case SQLReviewRuleLevel.WARNING:
      return "WARNING";
    case SQLReviewRuleLevel.DISABLED:
      return "DISABLED";
    default:
      return "UNKNOWN";
  }
};

const text = computed(() => {
  const { level, suffix } = props;
  const parts = [
    t(`sql-review.level.${sqlReviewRuleLevelToString(level).toLowerCase()}`),
  ];
  if (suffix) {
    parts.push(suffix);
  }
  return parts.join(" ");
});
</script>
