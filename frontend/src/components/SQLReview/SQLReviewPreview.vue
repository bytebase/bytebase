<template>
  <SQLRuleTable
    key="sql-review-preview-rule-table"
    :class="[state.updating && 'pointer-events-none']"
    :rule-list="state.selectedRuleList"
    :editable="editable"
    @level-change="onLevelChange"
    @payload-change="onPayloadChange"
  />
</template>

<script lang="ts" setup>
import {
  pushNotification,
  useSQLReviewStore,
  useSubscriptionStore,
} from "@/store";
import {
  convertRuleTemplateToPolicyRule,
  ruleIsAvailableInSubscription,
  RuleLevel,
  RuleTemplate,
  SQLReviewPolicy,
} from "@/types";
import { cloneDeep } from "lodash-es";
import { reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { payloadValueListToComponentList, SQLRuleTable } from "./components";
import { PayloadValueType } from "./components/RuleConfigComponents";

interface LocalState {
  selectedRuleList: RuleTemplate[];
  rulesUpdated: boolean;
  updating: boolean;
}

const props = withDefaults(
  defineProps<{
    policy: SQLReviewPolicy;
    selectedRuleList?: RuleTemplate[];
    editable?: boolean;
  }>(),
  {
    selectedRuleList: () => [],
    editable: true,
  }
);

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedRuleList: cloneDeep(props.selectedRuleList),
  rulesUpdated: false,
  updating: false,
});
const subscriptionStore = useSubscriptionStore();

const change = (rule: RuleTemplate, overrides: Partial<RuleTemplate>) => {
  if (
    !ruleIsAvailableInSubscription(rule.type, subscriptionStore.currentPlan)
  ) {
    return;
  }

  const index = state.selectedRuleList.findIndex((r) => r.type === rule.type);
  if (index < 0) {
    return;
  }
  const newRule = {
    ...state.selectedRuleList[index],
    ...overrides,
  };
  state.selectedRuleList[index] = newRule;
  state.rulesUpdated = true;
};

const onPayloadChange = (rule: RuleTemplate, data: PayloadValueType[]) => {
  const componentList = payloadValueListToComponentList(rule, data);
  change(rule, { componentList });
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  change(rule, { level });
};

watch(
  () => state.rulesUpdated,
  async (updated) => {
    if (!updated) return;

    const upsert = {
      name: props.policy.name,
      ruleList: state.selectedRuleList.map((rule) =>
        convertRuleTemplateToPolicyRule(rule)
      ),
    };

    state.updating = true;
    await useSQLReviewStore().updateReviewPolicy({
      id: props.policy.id,
      ...upsert,
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-updated"),
    });

    state.updating = false;
    state.rulesUpdated = false;
  }
);

watch(
  () => props.selectedRuleList,
  (ruleList) => {
    state.selectedRuleList = cloneDeep(ruleList);
    state.rulesUpdated = false;
  }
);
</script>
