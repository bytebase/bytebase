<template>
  <div class="my-4 space-y-4 divide-y divide-block-border">
    <BBStepTab
      class="my-4"
      :step-item-list="stepList"
      :allow-next="allowNext"
      :finish-title="$t(`common.confirm-and-${id ? 'update' : 'add'}`)"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="onCancel"
    >
      <template #0>
        <SchemaGuideInfo
          :name="state.name"
          :selected-env-name-list="state.selectedEnvNameList"
          :environment-list="environmentList?.map((env) => env.name) ?? []"
          @name-change="(val) => (state.name = val)"
          class="py-5"
        />
      </template>
      <template #1>
        <SchemaGuideConfig
          class="py-5"
          :select-rule-list="state.selectedRuleList"
          @select="onRuleSelect"
          @remove="onRuleRemove"
          @change="onRuleChange"
          @apply-template="onTemplateApply"
        />
      </template>
      <template #2>
        <SchemaGuidePreview
          :name="state.name"
          :rule-list="state.selectedRuleList"
          class="py-5"
        />
      </template>
    </BBStepTab>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBStepTabItem } from "../bbkit/types";
import { Rule, RuleLevel, SelectedRule } from "../types/schemaSystem";
import { useEnvironmentList, useSchemaSystemStore } from "@/store";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvNameList: string[];
  selectedRuleList: SelectedRule[];
}

const props = defineProps({
  id: {
    required: false,
    default: undefined,
    type: Number,
  },
  name: {
    required: false,
    default: "Database schema guideline",
    type: String,
  },
  selectedEnvNameList: {
    required: false,
    default: [],
    type: Object as PropType<string[]>,
  },
  selectedRuleList: {
    required: false,
    default: [],
    type: Object as PropType<SelectedRule[]>,
  },
});

const emit = defineEmits(["cancel"]);

const { t } = useI18n();
const router = useRouter();
const store = useSchemaSystemStore();
const environmentList = useEnvironmentList(["NORMAL"]);

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.database-review-guide";

const stepList: BBStepTabItem[] = [
  { title: t("database-review-guide.create.basic-info.name") },
  { title: t("database-review-guide.create.configure-rule.name") },
  { title: t("database-review-guide.create.preview.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name,
  selectedEnvNameList: props.selectedEnvNameList,
  selectedRuleList: props.selectedRuleList,
});

const onCancel = () => {
  if (props.id) {
    emit("cancel");
  } else {
    router.push({
      name: ROUTE_NAME,
    });
  }
};

const allowNext = computed((): boolean => {
  switch (state.currentStep) {
    case BASIC_INFO_STEP:
      return !!state.name && state.selectedEnvNameList.length > 0;
    case CONFIGURE_RULE_STEP:
      return state.selectedRuleList.length > 0;
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

const tryFinishSetup = (allowChangeCallback: () => void) => {
  const envIds: number[] = [];
  for (const envName of state.selectedEnvNameList) {
    const id = environmentList.value.find((e) => e.name === envName)?.id;
    if (id) {
      envIds.push(id);
    }
  }

  const guide = {
    id: props.id ?? store.guideList.length + 1,
    name: state.name,
    environmentList: envIds,
    createdTs: new Date().getTime() / 1000,
    updatedTs: new Date().getTime() / 1000,
    ruleList: state.selectedRuleList.map((rule) => ({
      id: rule.id,
      level: rule.level,
      payload: rule.payload
        ? Object.entries(rule.payload).reduce((res, [key, val]) => {
            res[key] = val.value ?? val.default;
            return res;
          }, {} as { [key: string]: any })
        : undefined,
    })),
  };

  if (props.id) {
    store.updateGuideline(guide);
  } else {
    store.addGuideline(guide);
  }

  allowChangeCallback();
  onCancel();
};

const onTemplateApply = (ruleList: SelectedRule[]) => {
  state.selectedRuleList = [...ruleList];
};

const onRuleSelect = (rule: Rule) => {
  state.selectedRuleList.push({
    ...rule,
    level: RuleLevel.Error,
  });
};

const onRuleRemove = (rule: SelectedRule) => {
  const index = state.selectedRuleList.findIndex((r) => r.id === rule.id);
  state.selectedRuleList = [
    ...state.selectedRuleList.slice(0, index),
    ...state.selectedRuleList.slice(index + 1),
  ];
};

const onRuleChange = (rule: SelectedRule) => {
  const index = state.selectedRuleList.findIndex((r) => r.id === rule.id);
  state.selectedRuleList = [
    ...state.selectedRuleList.slice(0, index),
    rule,
    ...state.selectedRuleList.slice(index + 1),
  ];
};
</script>
