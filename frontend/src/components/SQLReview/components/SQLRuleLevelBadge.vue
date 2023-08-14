<template>
  <BBBadge :text="text" :can-remove="false" :style="style" />
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBBadgeStyle } from "@/bbkit/BBBadge.vue";
import { RuleLevel } from "@/types/sqlReview";

const props = defineProps({
  level: {
    required: true,
    type: String as PropType<RuleLevel>,
  },
  suffix: {
    type: String,
    default: "",
  },
});

const { t } = useI18n();

const style = computed((): BBBadgeStyle => {
  switch (props.level) {
    case RuleLevel.ERROR:
      return "CRITICAL";
    case RuleLevel.WARNING:
      return "WARN";
    default:
      return "DISABLED";
  }
});

const text = computed(() => {
  const { level, suffix } = props;
  const parts = [t(`sql-review.level.${level.toLowerCase()}`)];
  if (suffix) {
    parts.push(suffix);
  }
  return parts.join(" ");
});
</script>
