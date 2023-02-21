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
    />
  </div>
</template>

<script lang="ts" setup>
import { PropType, toRef } from "vue";
import {
  RuleLevel,
  RuleTemplate,
  RuleConfigComponent,
  SQLReviewPolicyTemplate,
} from "@/types/sqlReview";
import { SQLRuleTable, SQLRuleFilter, useSQLRuleFilter } from "./components/";

const props = defineProps({
  selectedRuleList: {
    required: true,
    type: Object as PropType<RuleTemplate[]>,
  },
  templateList: {
    required: true,
    type: Object as PropType<SQLReviewPolicyTemplate[]>,
  },
  selectedTemplateIndex: {
    required: true,
    type: Number,
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

  const componentList = rule.componentList.reduce<RuleConfigComponent[]>(
    (list, component, index) => {
      switch (component.payload.type) {
        case "STRING_ARRAY":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string[],
            },
          });
          break;
        case "NUMBER":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as number,
            },
          });
          break;
        case "BOOLEAN":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as boolean,
            },
          });
          break;
        default:
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string,
            },
          });
          break;
      }
      return list;
    },
    []
  );

  emit("payload-change", rule, componentList);
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  emit("level-change", rule, level);
};
</script>
