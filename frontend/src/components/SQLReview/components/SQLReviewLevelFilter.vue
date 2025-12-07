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
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";
import SQLRuleLevelBadge from "./SQLRuleLevelBadge.vue";

const LEVEL_LIST = [SQLReviewRule_Level.ERROR, SQLReviewRule_Level.WARNING];

const props = withDefaults(
  defineProps<{
    ruleList: RuleListWithCategory[];
    isCheckedLevel?: (level: SQLReviewRule_Level) => boolean;
  }>(),
  {
    isCheckedLevel: () => false,
  }
);

defineEmits<{
  (
    event: "toggle-checked-level",
    level: SQLReviewRule_Level,
    on: boolean
  ): void;
}>();

const errorLevelList = computed(() => {
  const map = LEVEL_LIST.reduce((m, level) => {
    m.set(level, 0);
    return m;
  }, new Map<SQLReviewRule_Level, number>());

  for (const ruleWithCategory of props.ruleList) {
    for (const rule of ruleWithCategory.ruleList) {
      const count = map.get(rule.level) ?? 0;
      map.set(rule.level, count + 1);
    }
  }
  return map;
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
</script>
