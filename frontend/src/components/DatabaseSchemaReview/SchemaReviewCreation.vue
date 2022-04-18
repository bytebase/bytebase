<template>
  <div>
    <BBStepTab
      class="my-4"
      :step-item-list="stepList"
      :allow-next="allowNext"
      :finish-title="$t(`common.confirm-and-${reviewId ? 'update' : 'add'}`)"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="onCancel"
    >
      <template #0>
        <SchemaReviewInfo
          :name="state.name"
          :selected-environment-list="state.selectedEnvironmentList"
          :available-environment-list="availableEnvironmentList"
          :template-list="templateList"
          :selected-template-index="state.templateIndex"
          :is-edit="!!reviewId"
          @select-template="tryApplyTemplate"
          @name-change="(val) => (state.name = val)"
          @toggle-env="(env) => onEnvToggle(env)"
          class="py-5"
        />
      </template>
      <template #1>
        <SchemaReviewConfig
          class="py-5"
          :select-rule-list="state.selectedRuleList"
          :template-list="templateList"
          :selected-template-index="state.templateIndex"
          @change="onRuleChange"
          @apply-template="tryApplyTemplate"
        />
      </template>
      <template #2>
        <SchemaReviewPreview
          :name="state.name"
          :selected-rule-list="state.selectedRuleList"
          :selected-environment-list="state.selectedEnvironmentList"
          class="py-5"
        />
      </template>
    </BBStepTab>
    <BBAlert
      v-if="state.showAlertModal"
      style="CRITICAL"
      :ok-text="$t('common.confirm')"
      :title="$t('schema-review.create.configure-rule.confirm-override-title')"
      :description="
        $t('schema-review.create.configure-rule.confirm-override-description')
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
  </div>
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
  Environment,
  SchemaReviewTemplate,
} from "../../types";
import {
  pushNotification,
  useEnvironmentList,
  useSchemaSystemStore,
} from "@/store";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvironmentList: Environment[];
  selectedRuleList: SelectedRule[];
  ruleUpdated: boolean;
  showAlertModal: boolean;
  templateIndex: number;
  pendingApplyTemplateIndex: number;
}

const props = withDefaults(
  defineProps<{
    reviewId?: number;
    name?: string;
    selectedEnvironmentList?: Environment[];
    selectedRuleList?: SelectedRule[];
  }>(),
  {
    reviewId: undefined,
    name: "Database schema review",
    selectedEnvironmentList: () => [],
    selectedRuleList: () => [],
  }
);

const emit = defineEmits(["cancel"]);

const { t } = useI18n();
const router = useRouter();
const store = useSchemaSystemStore();

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.schema-review";

const stepList: BBStepTabItem[] = [
  { title: t("schema-review.create.basic-info.name") },
  { title: t("schema-review.create.configure-rule.name") },
  { title: t("schema-review.create.preview.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name,
  selectedEnvironmentList: props.selectedEnvironmentList,
  selectedRuleList: props.selectedRuleList,
  ruleUpdated: false,
  showAlertModal: false,
  templateIndex: -1,
  pendingApplyTemplateIndex: -1,
});

const availableEnvironmentList = computed((): Environment[] => {
  const environmentList = useEnvironmentList(["NORMAL"]);

  const filteredList = store.availableEnvironments(
    environmentList.value,
    props.reviewId
  );

  return filteredList;
});

const onCancel = () => {
  if (props.reviewId) {
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
      return !!state.name && state.selectedRuleList.length > 0;
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
  const review = {
    name: state.name,
    environmentList: state.selectedEnvironmentList.map((env) => env.id),
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

  if (props.reviewId) {
    store.updateReview(props.reviewId, review);
  } else {
    store.addReview(review);
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(`schema-review.${props.reviewId ? "update" : "create"}-review`),
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

const onEnvToggle = (env: Environment) => {
  const index = state.selectedEnvironmentList.findIndex((e) => e.id === env.id);

  if (index >= 0) {
    state.selectedEnvironmentList = [
      ...state.selectedEnvironmentList.slice(0, index),
      ...state.selectedEnvironmentList.slice(index + 1),
    ];
  } else {
    state.selectedEnvironmentList.push(env);
  }
};

const onTemplateApply = (index: number) => {
  if (index < 0 || index >= templateList.length) {
    return;
  }
  state.templateIndex = index;
  state.pendingApplyTemplateIndex = -1;
  state.selectedRuleList = [...templateList[index].ruleList];
};

const tryApplyTemplate = (index: number) => {
  if (state.ruleUpdated || props.reviewId) {
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
