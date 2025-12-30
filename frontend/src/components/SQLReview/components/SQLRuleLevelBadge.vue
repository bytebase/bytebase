<template>
  <NTag
    :type="style"
    :size="'small'"
  >
    {{ text }}
  </NTag>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
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

const style = computed(() => {
  switch (props.level) {
    case SQLReviewRule_Level.ERROR:
      return "error";
    case SQLReviewRule_Level.WARNING:
      return "warning";
    default:
      return "default";
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
