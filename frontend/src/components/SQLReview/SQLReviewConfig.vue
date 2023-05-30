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
import {
  RuleLevel,
  RuleTemplate,
  RuleConfigComponent,
} from "@/types/sqlReview";
import {
  SQLRuleTable,
  SQLRuleFilter,
  useSQLRuleFilter,
  payloadValueListToComponentList,
} from "./components/";

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
    componentList: RuleConfigComponent[]
  ): void;
  (event: "level-change", rule: RuleTemplate, level: RuleLevel): void;
  (event: "comment-change", rule: RuleTemplate, comment: string): void;
}>();

const {
  params: filterParams,
  events: filterEvents,
  filteredRuleList,
} = useSQLRuleFilter(toRef(props, "selectedRuleList"));

const onPayloadChange = (
  rule: RuleTemplate,
  data: (string | number | boolean | string[])[]
) => {
  if (!rule.componentList) {
    return;
  }
  const componentList = payloadValueListToComponentList(rule, data);
  emit("payload-change", rule, componentList);
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  emit("level-change", rule, level);
};

const onCommentChange = (rule: RuleTemplate, comment: string) => {
  emit("comment-change", rule, comment);
};
</script>
