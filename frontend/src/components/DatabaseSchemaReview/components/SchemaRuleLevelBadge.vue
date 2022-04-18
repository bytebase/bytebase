<template>
  <BBBadge
    :text="$t(`schema-review.error-level.${level}`)"
    :can-remove="false"
    :style="style"
  />
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { RuleLevel } from "../../../types/schemaSystem";
import { BBAttentionStyle } from "../../../bbkit/types";

const props = defineProps({
  level: {
    required: true,
    type: String as PropType<RuleLevel>,
  },
});

const style = computed((): BBAttentionStyle => {
  switch (props.level) {
    case RuleLevel.Error:
      return "CRITICAL";
    case RuleLevel.Warning:
      return "WARN";
    default:
      return "INFO";
  }
});
</script>
