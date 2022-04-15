<template>
  <BBBadge
    :text="$t(`schame-review.error-level.${level}`)"
    :can-remove="false"
    :theme="theme"
  />
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { RuleLevel } from "../../../types/schemaSystem";

const props = defineProps({
  level: {
    required: true,
    type: String as PropType<RuleLevel>,
  },
});

const theme = computed((): string => {
  switch (props.level) {
    case RuleLevel.Error:
      return "red";
    case RuleLevel.Warning:
      return "yellow";
    default:
      return "gray";
  }
});
</script>
