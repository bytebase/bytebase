<template>
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
      <SchemaReviewInfo
        :name="state.name"
        :selected-env-name-list="state.selectedEnvNameList"
        :environment-list="environmentList"
        @name-change="(val) => (state.name = val)"
        class="py-5"
      />
    </template>
    <template #1>
      <SchemaReviewConfig
        class="py-5"
        :select-rule-list="state.selectedRuleList"
        :template-list="templateList"
        @change="onRuleChange"
        @apply-template="tryApplyTemplate"
      />
    </template>
    <template #2>
      <SchemaReviewPreview
        :name="state.name"
        :rule-list="state.selectedRuleList"
        class="py-5"
      />
    </template>
  </BBStepTab>
  <BBAlert
    v-if="state.showAlertModal"
    style="CRITICAL"
    :ok-text="$t('common.confirm')"
    :title="$t('schame-review.create.configure-rule.confirm-override-title')"
    :description="
      $t('schame-review.create.configure-rule.confirm-override-description')
    "
    @ok="
      () => {
        state.showAlertModal = false;
        state.ruleUpdated = false;
        onTemplateApply(state.pendingApplyTemplateIndex);
      }
    "
    @cancel="state.showAlertModal = false"
  >
  </BBAlert>
</template>

<script lang="ts" setup>
import { reactive, computed, withDefaults } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBStepTabItem } from "../../bbkit/types";
import {
  ruleList,
  RuleLevel,
  SelectedRule,
  SchemaReviewTemplate,
} from "../../types/schemaSystem";
import {
  pushNotification,
  useEnvironmentList,
  useSchemaSystemStore,
} from "@/store";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvNameList: string[];
  selectedRuleList: SelectedRule[];
  ruleUpdated: boolean;
  showAlertModal: boolean;
  pendingApplyTemplateIndex: number;
}

const props = withDefaults(
  defineProps<{
    id?: number;
    name?: string;
    selectedEnvNameList?: string[];
    selectedRuleList?: SelectedRule[];
  }>(),
  {
    id: undefined,
    name: "Database schema review",
    selectedEnvNameList: () => [],
    selectedRuleList: () => [],
  }
);

const emit = defineEmits(["cancel"]);

const { t } = useI18n();
const router = useRouter();
const store = useSchemaSystemStore();
const environmentList = useEnvironmentList(["NORMAL"]);

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.schame-review";

const stepList: BBStepTabItem[] = [
  { title: t("schame-review.create.basic-info.name") },
  { title: t("schame-review.create.configure-rule.name") },
  { title: t("schame-review.create.preview.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name,
  selectedEnvNameList: props.selectedEnvNameList,
  selectedRuleList: props.selectedRuleList,
  ruleUpdated: false,
  showAlertModal: false,
  pendingApplyTemplateIndex: -1,
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

  const review = {
    name: state.name,
    environmentList: envIds,
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
    store.updateReview(props.id, review);
  } else {
    store.addReview(review);
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(`schame-review.${props.id ? "update" : "create"}-review`),
  });

  allowChangeCallback();
  onCancel();
};

const getRuleListWithLevel = (
  idList: string[],
  level: RuleLevel
): SelectedRule[] => {
  return idList.reduce((res, id) => {
    const rule = ruleList.find((r) => r.id === id);
    if (!rule) {
      return res;
    }
    res.push({
      ...rule,
      level,
    });
    return res;
  }, [] as SelectedRule[]);
};

const templateList: SchemaReviewTemplate[] = [
  {
    name: "Schema review for Prod",
    image: new URL("../../assets/plan-enterprise.png", import.meta.url).href,
    ruleList: [
      ...getRuleListWithLevel(["engine.mysql.use-innodb"], RuleLevel.Error),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.pk",
          "naming.index.uk",
          "naming.index.idx",
        ],
        RuleLevel.Warning
      ),
      ...getRuleListWithLevel(
        [
          "query.select.no-select-all",
          "query.where.require",
          "query.where.no-leading-wildcard-like",
          "table.require-pk",
        ],
        RuleLevel.Error
      ),
      ...getRuleListWithLevel(
        ["column.required", "column.no-null"],
        RuleLevel.Warning
      ),
    ],
  },
  {
    name: "Schema review for Dev",
    image: new URL("../../assets/plan-free.png", import.meta.url).href,
    ruleList: [
      ...getRuleListWithLevel(["engine.mysql.use-innodb"], RuleLevel.Error),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.pk",
          "naming.index.uk",
          "naming.index.idx",
          "query.select.no-select-all",
          "query.where.require",
          "query.where.no-leading-wildcard-like",
        ],
        RuleLevel.Warning
      ),
      ...getRuleListWithLevel(["table.require-pk"], RuleLevel.Error),
      ...getRuleListWithLevel(
        ["column.required", "column.no-null"],
        RuleLevel.Warning
      ),
    ],
  },
];

const onTemplateApply = (index: number) => {
  if (index < 0 || index >= templateList.length) {
    return;
  }
  state.selectedRuleList = [...templateList[index].ruleList];
};

const tryApplyTemplate = (index: number) => {
  if (state.ruleUpdated || props.id) {
    state.showAlertModal = true;
    state.pendingApplyTemplateIndex = index;
    return;
  }
  onTemplateApply(index);
};

const onRuleChange = (rule: SelectedRule) => {
  const index = state.selectedRuleList.findIndex((r) => r.id === rule.id);
  state.selectedRuleList = [
    ...state.selectedRuleList.slice(0, index),
    rule,
    ...state.selectedRuleList.slice(index + 1),
  ];
  state.ruleUpdated = true;
};
</script>
