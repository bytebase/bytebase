<template>
  <BBBadge
    :text="$t(`schema-review-policy.error-level.${level.toLowerCase()}`)"
    :can-remove="false"
    :style="style"
  />
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { RuleLevel } from "@/types/schemaSystem";
import { BBBadgeStyle } from "@/bbkit/BBBadge.vue";

const props = defineProps({
  level: {
    required: true,
    type: String as PropType<RuleLevel>,
  },
});

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
</script>
