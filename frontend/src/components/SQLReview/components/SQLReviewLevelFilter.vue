<template>
  <div
    class="flex flex-col items-start 2xl:flex-row 2xl:items-center gap-y-5 gap-x-5"
  >
    <div class="flex items-center gap-x-5">
      <label
        v-for="[level, count] in errorLevelList.entries()"
        :key="level"
        class="flex items-center gap-x-2 text-sm text-gray-600"
      >
        <NCheckbox
          :id="sqlReviewRuleLevelToString(level)"
          :checked="isCheckedLevel(level)"
          @update:checked="
            (checked) => $emit('toggle-checked-level', level, checked)
          "
        >
          <SQLRuleLevelBadge :level="level" :suffix="`(${count})`" />
        </NCheckbox>
      </label>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";
import SQLRuleLevelBadge from "./SQLRuleLevelBadge.vue";

const LEVEL_LIST = [
  SQLReviewRuleLevel.ERROR,
  SQLReviewRuleLevel.WARNING,
  SQLReviewRuleLevel.DISABLED,
];

const props = withDefaults(
  defineProps<{
    ruleList: RuleListWithCategory[];
    isCheckedLevel?: (level: SQLReviewRuleLevel) => boolean;
  }>(),
  {
    isCheckedLevel: () => false,
  }
);

defineEmits<{
  (event: "toggle-checked-level", level: SQLReviewRuleLevel, on: boolean): void;
}>();

const errorLevelList = computed(() => {
  const map = LEVEL_LIST.reduce((m, level) => {
    m.set(level, 0);
    return m;
  }, new Map<SQLReviewRuleLevel, number>());

  for (const ruleWithCategory of props.ruleList) {
    for (const rule of ruleWithCategory.ruleList) {
      const count = map.get(rule.level) ?? 0;
      map.set(rule.level, count + 1);
    }
  }
  return map;
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
</script>
