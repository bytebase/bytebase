<template>
  <SQLReviewTabsByEngine :rule-map-by-engine="ruleMapByEngine">
    <template
      #default="{
        ruleList: ruleListFilteredByEngine,
        engine,
      }: {
        ruleList: RuleTemplateV2[];
        engine: Engine;
      }"
    >
      <SQLRuleTableWithFilter
        :engine="engine"
        :rule-list="ruleListFilteredByEngine"
        :editable="true"
        @rule-change="onRuleChange"
      />
    </template>
  </SQLReviewTabsByEngine>
</template>

<script lang="ts" setup>
import type { Engine } from "@/types/proto/v1/common";
import type { RuleTemplateV2 } from "@/types/sqlReview";

defineProps<{
  ruleMapByEngine: Map<Engine, Map<string, RuleTemplateV2>>;
}>();

const emit = defineEmits<{
  (
    event: "rule-change",
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ): void;
}>();

const onRuleChange = (
  rule: RuleTemplateV2,
  overrides: Partial<RuleTemplateV2>
) => {
  emit("rule-change", rule, overrides);
};
</script>
