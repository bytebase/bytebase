<template>
  <BBBadge
    :text="`${level[0].toUpperCase()}${level.slice(1).toLowerCase()}`"
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
