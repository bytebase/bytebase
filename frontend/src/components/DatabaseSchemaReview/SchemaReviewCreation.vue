<template>
  <div>
    <BBStepTab
      class="my-4"
      :step-item-list="STEP_LIST"
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
          :template-list="TEMPLATE_LIST"
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
          :template-list="TEMPLATE_LIST"
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
      :title="
        $t('schema-review-policy.create.configure-rule.confirm-override-title')
      "
      :description="
        $t(
          'schema-review-policy.create.configure-rule.confirm-override-description'
        )
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
  RuleTemplate,
  Environment,
  SchemaReviewPolicyTemplate,
  convertRuleTemplateToPolicyRule,
} from "../../types";
import {
  useCurrentUser,
  pushNotification,
  useEnvironmentList,
  useSchemaSystemStore,
} from "@/store";
import { isOwner, isDBA } from "../../utils";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvironmentList: Environment[];
  selectedRuleList: RuleTemplate[];
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
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    reviewId: undefined,
    name: "",
    selectedEnvironmentList: () => [],
    selectedRuleList: () => [],
  }
);

const emit = defineEmits(["cancel"]);

const { t } = useI18n();
const router = useRouter();
const store = useSchemaSystemStore();
const currentUser = useCurrentUser();

const hasPermission = computed(() => {
  return isOwner(currentUser.value.role) || isDBA(currentUser.value.role);
});

const getRuleListWithLevel = (
  idList: string[],
  level: RuleLevel
): RuleTemplate[] => {
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
  }, [] as RuleTemplate[]);
};

// TODO: i18n
const TEMPLATE_LIST: SchemaReviewPolicyTemplate[] = [
  {
    name: "Schema review for Prod",
    imagePath: new URL("../../assets/plan-enterprise.png", import.meta.url)
      .href,
    ruleList: [
      ...getRuleListWithLevel(["engine.mysql.use-innodb"], RuleLevel.ERROR),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.pk",
          "naming.index.uk",
          "naming.index.idx",
        ],
        RuleLevel.WARNING
      ),
      ...getRuleListWithLevel(
        [
          "query.select.no-select-all",
          "query.where.require",
          "query.where.no-leading-wildcard-like",
          "table.require-pk",
        ],
        RuleLevel.ERROR
      ),
      ...getRuleListWithLevel(
        ["column.required", "column.no-null"],
        RuleLevel.WARNING
      ),
    ],
  },
  {
    name: "Schema review for Dev",
    imagePath: new URL("../../assets/plan-free.png", import.meta.url).href,
    ruleList: [
      ...getRuleListWithLevel(["engine.mysql.use-innodb"], RuleLevel.ERROR),
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
        RuleLevel.WARNING
      ),
      ...getRuleListWithLevel(["table.require-pk"], RuleLevel.ERROR),
      ...getRuleListWithLevel(
        ["column.required", "column.no-null"],
        RuleLevel.WARNING
      ),
    ],
  },
];

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.schema-review-policy";
const DEFAULT_TEMPLATE_INDEX = 0;

const STEP_LIST: BBStepTabItem[] = [
  { title: t("schema-review-policy.create.basic-info.name") },
  { title: t("schema-review-policy.create.configure-rule.name") },
  { title: t("schema-review-policy.create.preview.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name:
    props.name ||
    t("schema-review-policy.create.basic-info.display-name-default"),
  selectedEnvironmentList: props.selectedEnvironmentList,
  selectedRuleList: props.selectedRuleList.length
    ? [...props.selectedRuleList]
    : [...TEMPLATE_LIST[DEFAULT_TEMPLATE_INDEX].ruleList],
  ruleUpdated: false,
  showAlertModal: false,
  templateIndex: props.reviewId ? -1 : DEFAULT_TEMPLATE_INDEX,
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
  if (!hasPermission.value) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-review-policy.no-permission"),
    });
  }
  const review = {
    name: state.name,
    environmentIdList: state.selectedEnvironmentList.map((env) => env.id),
    ruleList: state.selectedRuleList.map((rule) =>
      convertRuleTemplateToPolicyRule(rule)
    ),
  };

  if (props.reviewId) {
    store.updateReviewPolicy(props.reviewId, review);
  } else {
    store.addReviewPolicy(review);
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(
      `schema-review-policy.${props.reviewId ? "update" : "create"}-review`
    ),
  });

  allowChangeCallback();
  onCancel();
};

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
  if (index < 0 || index >= TEMPLATE_LIST.length) {
    return;
  }
  state.templateIndex = index;
  state.pendingApplyTemplateIndex = -1;
  state.selectedRuleList = [...TEMPLATE_LIST[index].ruleList];
};

const tryApplyTemplate = (index: number) => {
  if (state.ruleUpdated || props.reviewId) {
    state.showAlertModal = true;
    state.pendingApplyTemplateIndex = index;
    return;
  }
  onTemplateApply(index);
};

const onRuleChange = (rule: RuleTemplate) => {
  const index = state.selectedRuleList.findIndex((r) => r.id === rule.id);
  state.selectedRuleList = [
    ...state.selectedRuleList.slice(0, index),
    rule,
    ...state.selectedRuleList.slice(index + 1),
  ];
  state.ruleUpdated = true;
};
</script>
