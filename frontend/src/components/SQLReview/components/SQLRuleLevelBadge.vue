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
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";

const props = withDefaults(
  defineProps<{
    level: SQLReviewRule_Level;
    suffix?: string;
  }>(),
  {
    suffix: "",
  }
);

const { t } = useI18n();

const style = computed((): BBBadgeStyle => {
  switch (props.level) {
    case SQLReviewRule_Level.ERROR:
      return "CRITICAL";
    case SQLReviewRule_Level.WARNING:
      return "WARN";
    default:
      return "DISABLED";
  }
});

// Helper function to convert SQLReviewRule_Level to string
const sqlReviewRuleLevelToString = (level: SQLReviewRule_Level): string => {
  switch (level) {
    case SQLReviewRule_Level.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case SQLReviewRule_Level.ERROR:
      return "ERROR";
    case SQLReviewRule_Level.WARNING:
      return "WARNING";
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
