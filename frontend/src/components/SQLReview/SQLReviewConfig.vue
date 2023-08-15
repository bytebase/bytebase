<template>
  <div>
    <SQLRuleFilter
      :rule-list="selectedRuleList"
      :params="filterParams"
      v-on="filterEvents"
    />
    <SQLRuleTable
      class="w-full"
      :rule-list="filteredRuleList"
      :editable="true"
      @level-change="onLevelChange"
      @payload-change="onPayloadChange"
      @comment-change="onCommentChange"
    />
  </div>
</template>

<script lang="ts" setup>
import { PropType, toRef } from "vue";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { RuleTemplate } from "@/types/sqlReview";
import {
  SQLRuleTable,
  SQLRuleFilter,
  useSQLRuleFilter,
  payloadValueListToComponentList,
} from "./components/";
import { PayloadForEngine } from "./components/RuleConfigComponents";

const props = defineProps({
  selectedRuleList: {
    required: true,
    type: Object as PropType<RuleTemplate[]>,
  },
});

const emit = defineEmits<{
  (event: "apply-template", index: number): void;
  (
    event: "payload-change",
    rule: RuleTemplate,
    update: Partial<RuleTemplate>
  ): void;
  (event: "level-change", rule: RuleTemplate, level: SQLReviewRuleLevel): void;
  (event: "comment-change", rule: RuleTemplate, comment: string): void;
}>();

const {
  params: filterParams,
  events: filterEvents,
  filteredRuleList,
} = useSQLRuleFilter(toRef(props, "selectedRuleList"));

const onPayloadChange = (rule: RuleTemplate, data: PayloadForEngine) => {
  if (!rule.componentList) {
    return;
  }
  emit("payload-change", rule, payloadValueListToComponentList(rule, data));
};

const onLevelChange = (rule: RuleTemplate, level: SQLReviewRuleLevel) => {
  emit("level-change", rule, level);
};

const onCommentChange = (rule: RuleTemplate, comment: string) => {
  emit("comment-change", rule, comment);
};
</script>
