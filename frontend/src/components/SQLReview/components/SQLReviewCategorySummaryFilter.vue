<template>
  <div
    class="flex flex-col items-start 2xl:flex-row 2xl:items-center gap-y-5 gap-x-5"
  >
    <div class="flex items-center gap-x-5">
      <label
        v-for="stats in errorLevelList"
        :key="stats.level"
        class="flex items-center gap-x-2 text-sm text-gray-600"
      >
        <NCheckbox
          :id="sQLReviewRuleLevelToJSON(stats.level)"
          :checked="isCheckedLevel(stats.level)"
          @update:checked="
            (checked) => $emit('toggle-checked-level', stats.level, checked)
          "
        >
          <SQLRuleLevelBadge
            :level="stats.level"
            :suffix="`(${stats.count})`"
          />
        </NCheckbox>
      </label>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import type { RuleTemplateV2 } from "@/types";
import { LEVEL_LIST } from "@/types";
import type { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { sQLReviewRuleLevelToJSON } from "@/types/proto/v1/org_policy_service";
import SQLRuleLevelBadge from "./SQLRuleLevelBadge.vue";

type RuleLevelStats = {
  level: SQLReviewRuleLevel;
  count: number;
};

const props = withDefaults(
  defineProps<{
    ruleList: RuleTemplateV2[];
    isCheckedLevel?: (level: SQLReviewRuleLevel) => boolean;
  }>(),
  {
    isCheckedLevel: () => false,
  }
);

defineEmits<{
  (event: "toggle-checked-level", level: SQLReviewRuleLevel, on: boolean): void;
}>();

const errorLevelList = computed((): RuleLevelStats[] => {
  return LEVEL_LIST.map((level) => ({
    level,
    count: props.ruleList.filter((rule) => rule.level === level).length,
  }));
});
</script>
