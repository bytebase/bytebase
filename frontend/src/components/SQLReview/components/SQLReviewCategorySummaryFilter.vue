<template>
  <div
    class="flex flex-col items-start 2xl:flex-row 2xl:items-center gap-y-5 gap-x-5"
  >
    <div class="flex items-center gap-x-5">
      <label
        v-for="stats in engineList"
        :key="stats.engine"
        class="flex items-center gap-x-1 text-sm text-gray-600"
      >
        <NCheckbox
          :id="engineToJSON(stats.engine)"
          :checked="isCheckedEngine(stats.engine)"
          @update:checked="
            (checked) => emit('toggle-checked-engine', stats.engine, checked)
          "
        />
        <EngineIcon
          :engine="engineFromJSON(stats.engine)"
          custom-class="ml-1"
        />
        <span
          class="items-center text-xs px-1 py-0.5 rounded-full bg-gray-200 text-gray-800"
        >
          {{ stats.count }}
        </span>
      </label>
    </div>
    <div class="hidden 2xl:block h-[1.5rem] border-l border-control-border" />
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
        />
        <SQLRuleLevelBadge :level="stats.level" :suffix="`(${stats.count})`" />
      </label>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { LEVEL_LIST, RuleTemplate } from "@/types";
import { engineFromJSON, Engine, engineToJSON } from "@/types/proto/v1/common";
import {
  SQLReviewRuleLevel,
  sQLReviewRuleLevelToJSON,
} from "@/types/proto/v1/org_policy_service";
import SQLRuleLevelBadge from "./SQLRuleLevelBadge.vue";

type EngineTypeStats = {
  engine: Engine;
  count: number;
};
type RuleLevelStats = {
  level: SQLReviewRuleLevel;
  count: number;
};

const props = withDefaults(
  defineProps<{
    ruleList: RuleTemplate[];
    isCheckedEngine?: (engine: Engine) => boolean;
    isCheckedLevel?: (level: SQLReviewRuleLevel) => boolean;
  }>(),
  {
    isCheckedEngine: () => false,
    isCheckedLevel: () => false,
  }
);

const emit = defineEmits<{
  (event: "toggle-checked-engine", engine: Engine, on: boolean): void;
  (event: "toggle-checked-level", level: SQLReviewRuleLevel, on: boolean): void;
}>();

const engineList = computed((): EngineTypeStats[] => {
  const tmp = props.ruleList.reduce((dict, rule) => {
    for (const engine of rule.engineList) {
      if (!dict[engine]) {
        dict[engine] = {
          engine: engine,
          count: 0,
        };
      }
      dict[engine].count += 1;
    }
    return dict;
  }, {} as { [id: string]: EngineTypeStats });
  return Object.values(tmp);
});

const errorLevelList = computed((): RuleLevelStats[] => {
  return LEVEL_LIST.map((level) => ({
    level,
    count: props.ruleList.filter((rule) => rule.level === level).length,
  }));
});
</script>
