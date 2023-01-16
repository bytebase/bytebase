<template>
  <div v-if="summary.length > 0" class="inline-flex gap-x-1">
    <SQLRuleLevelBadge
      v-for="item in summary"
      :key="item.level"
      :level="item.level"
      :suffix="`(${item.count})`"
    />
  </div>
</template>

<script lang="ts" setup>
import { groupBy } from "lodash-es";
import { computed } from "vue";

import { RuleLevel, RuleTemplate } from "@/types";
import SQLRuleLevelBadge from "./SQLRuleLevelBadge.vue";

const props = withDefaults(
  defineProps<{
    ruleList?: RuleTemplate[];
  }>(),
  {
    ruleList: () => [],
  }
);

const summary = computed(() => {
  const groups = groupBy(props.ruleList, (rule) => rule.level);

  return Object.keys(groups).map((level) => {
    const group = groups[level];
    return {
      level: level as RuleLevel,
      count: group.length,
    };
  });
});
</script>
