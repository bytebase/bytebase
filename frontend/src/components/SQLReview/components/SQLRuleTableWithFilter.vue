<template>
  <SQLRuleFilter
    :rule-list="ruleList"
    :params="filterParams"
    v-on="filterEvents"
  >
    <template
      #default="{ ruleList: filteredRuleList }: { ruleList: RuleTemplateV2[] }"
    >
      <SQLRuleTable
        v-if="filteredRuleList.length > 0"
        :rule-list="filteredRuleList"
        :editable="editable"
        @rule-change="onRuleChange"
      />
      <NoDataPlaceholder v-else class="my-5" />
    </template>
  </SQLRuleFilter>
</template>

<script setup lang="ts">
import { watch } from "vue";
import {
  SQLRuleFilter,
  useSQLRuleFilter,
  SQLRuleTable,
} from "@/components/SQLReview/components";
import type { RuleTemplateV2 } from "@/types";
import type { Engine } from "@/types/proto/v1/common";

const props = defineProps<{
  engine: Engine;
  ruleList: RuleTemplateV2[];
  editable: boolean;
}>();

const emit = defineEmits<{
  (
    event: "rule-change",
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ): void;
}>();

const { params: filterParams, events: filterEvents } = useSQLRuleFilter();

watch(
  () => props.engine,
  () => filterEvents.reset()
);

const onRuleChange = (
  rule: RuleTemplateV2,
  update: Partial<RuleTemplateV2>
) => {
  emit("rule-change", rule, update);
};
</script>
