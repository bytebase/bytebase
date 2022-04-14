<template>
  <div class="my-4 space-y-4 divide-y divide-block-border">
    <BBStepTab
      class="my-4"
      :step-item-list="stepList"
      :allow-next="allowNext"
      :finish-title="$t('common.confirm-and-add')"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="cancelSetup"
    >
      <template #0>
        <SchemaGuideInfo
          :name="state.name"
          :selected-env-name-list="state.selectedEnvNameList"
          :environment-list="environmentList?.map((env) => env.name) ?? []"
          class="py-5"
        />
      </template>
      <template #1>
        <SchemaGuideConfig
          class="py-5"
          :select-rule-list="state.ruleList"
          @select="onRuleSelect"
          @remove="onRuleRemove"
          @change="onRuleChange"
          @apply-template="onTemplateApply"
        />
      </template>
      <template #2>
        <SchemaGuidePreview
          :name="state.name"
          :rule-list="state.ruleList"
          class="py-5"
        />
      </template>
    </BBStepTab>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { BBStepTabItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { Rule, RuleLevel, SelectedRule } from "../types/schemaSystem";
import { useEnvironmentList } from "@/store";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvNameList: string[];
  ruleList: SelectedRule[];
}

const { t } = useI18n();
const router = useRouter();

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;

const stepList: BBStepTabItem[] = [
  { title: t("database-review-guide.create.basic-info.name") },
  { title: t("database-review-guide.create.configure-rule.name") },
  { title: t("database-review-guide.create.preview.name") },
];

const cancelSetup = () => {
  router.push({
    name: "setting.workspace.database-review-guide",
  });
};

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: "Database schema guideline",
  selectedEnvNameList: [],
  ruleList: [],
});

const environmentList = useEnvironmentList(["NORMAL"]);

const allowNext = computed((): boolean => {
  switch (state.currentStep) {
    case BASIC_INFO_STEP:
      return !!state.name && state.selectedEnvNameList.length > 0;
    case CONFIGURE_RULE_STEP:
      return state.ruleList.length > 0;
    case PREVIEW_STEP:
      return true;
  }
  return false;
});

const tryChangeStep = (
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  state.currentStep = newStep;
  allowChangeCallback();
};

const tryFinishSetup = (allowChangeCallback: () => void) => {};

const onTemplateApply = (ruleList: SelectedRule[]) => {
  state.ruleList = [...ruleList];
};

const onRuleSelect = (rule: Rule) => {
  state.ruleList.push({
    ...rule,
    level: RuleLevel.Error,
  });
};

const onRuleRemove = (rule: SelectedRule) => {
  const index = state.ruleList.findIndex((r) => r.id === rule.id);
  state.ruleList = [
    ...state.ruleList.slice(0, index),
    ...state.ruleList.slice(index + 1),
  ];
};

const onRuleChange = (rule: SelectedRule) => {
  const index = state.ruleList.findIndex((r) => r.id === rule.id);
  state.ruleList = [
    ...state.ruleList.slice(0, index),
    rule,
    ...state.ruleList.slice(index + 1),
  ];
};
</script>
