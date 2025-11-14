<template>
  <SQLRuleFilter
    :rule-list="ruleList"
    :selected-rule-count="selectedRuleKeys.length"
    :params="filterParams"
    :hide-level-filter="hideLevel"
    :support-select="supportSelect"
    @toggle-select-all="toggleSelectAll"
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
        :support-select="supportSelect"
        :selected-rule-keys="selectedRuleKeys"
        :size="size"
        @rule-upsert="onRuleChange"
        @rule-remove="$emit('rule-remove', $event)"
        @update:selected-rule-keys="$emit('update:selectedRuleKeys', $event)"
      />
      <NEmpty v-else class="py-12 border rounded-sm" />
    </template>
  </SQLRuleFilter>
</template>

<script setup lang="ts">
import { NEmpty } from "naive-ui";
import { watch } from "vue";
import {
  SQLRuleFilter,
  SQLRuleTable,
  useSQLRuleFilter,
} from "@/components/SQLReview/components";
import type { RuleTemplateV2 } from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";
import { getRuleKey } from "./utils";

const props = withDefaults(
  defineProps<{
    engine: Engine;
    ruleList: RuleTemplateV2[];
    editable: boolean;
    hideLevel?: boolean;
    supportSelect?: boolean;
    selectedRuleKeys?: string[];
    size?: "small" | "medium";
  }>(),
  {
    supportSelect: false,
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
  (event: "rule-remove", rule: RuleTemplateV2): void;
  (event: "update:selectedRuleKeys", keys: string[]): void;
}>();

const { params: filterParams, events: filterEvents } = useSQLRuleFilter();

watch(
  () => props.engine,
  () => filterEvents.reset()
);

const toggleSelectAll = (select: boolean) => {
  if (!select) {
    emit("update:selectedRuleKeys", []);
  } else {
    emit("update:selectedRuleKeys", props.ruleList.map(getRuleKey));
  }
};

const onRuleChange = (
  rule: RuleTemplateV2,
  update: Partial<RuleTemplateV2>
) => {
  emit("rule-upsert", rule, update);
};
</script>
