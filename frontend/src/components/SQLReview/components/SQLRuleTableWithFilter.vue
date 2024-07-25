<template>
  <SQLRuleFilter
    :rule-list="ruleList"
    :params="filterParams"
    :hide-level-filter="hideLevel"
    v-on="filterEvents"
  >
    <template
      #default="{
        ruleList: filteredRuleList,
      }: {
        ruleList: RuleListWithCategory[];
      }"
    >
      <SQLRuleTable
        v-if="filteredRuleList.length > 0"
        :rule-list="filteredRuleList"
        :editable="editable"
        :hide-level="hideLevel"
        :select-rule="selectRule"
        :selected-rule-keys="selectedRuleKeys"
        :size="size"
        @rule-upsert="onRuleChange"
        @update:selected-rule-keys="$emit('update:selectedRuleKeys', $event)"
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
import NoDataPlaceholder from "@/components/misc/NoDataPlaceholder.vue";
import type { RuleTemplateV2 } from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";

const props = withDefaults(
  defineProps<{
    engine: Engine;
    ruleList: RuleTemplateV2[];
    editable: boolean;
    hideLevel?: boolean;
    selectRule?: boolean;
    selectedRuleKeys?: string[];
    size?: "small" | "medium";
  }>(),
  {
    selectRule: false,
    hideLevel: false,
    selectedRuleKeys: () => [],
    size: "medium",
  }
);

const emit = defineEmits<{
  (
    event: "rule-upsert",
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ): void;
  (event: "update:selectedRuleKeys", keys: string[]): void;
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
  emit("rule-upsert", rule, update);
};
</script>
