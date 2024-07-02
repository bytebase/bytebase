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
        @level-change="onLevelChange"
        @payload-change="onPayloadChange"
        @comment-change="onCommentChange"
      />
      <NoDataPlaceholder v-else class="my-5" />
    </template>
  </SQLRuleFilter>
</template>

<script setup lang="ts">
import { watch } from "vue";
import {
  payloadValueListToComponentList,
  SQLRuleFilter,
  useSQLRuleFilter,
  SQLRuleTable,
} from "@/components/SQLReview/components";
import type { RuleTemplateV2 } from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import type { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import type { PayloadValueType } from "./RuleConfigComponents";

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

const onPayloadChange = (rule: RuleTemplateV2, data: PayloadValueType[]) => {
  if (!rule.componentList) {
    return;
  }
  emit("rule-change", rule, payloadValueListToComponentList(rule, data));
};

const onLevelChange = (rule: RuleTemplateV2, level: SQLReviewRuleLevel) => {
  emit("rule-change", rule, { level });
};

const onCommentChange = (rule: RuleTemplateV2, comment: string) => {
  emit("rule-change", rule, { comment });
};
</script>
