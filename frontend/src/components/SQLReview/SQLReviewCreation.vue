<template>
  <div>
    <BBStepTab
      class="my-4"
      :step-item-list="STEP_LIST"
      :allow-next="allowNext"
      :finish-title="$t(`common.confirm-and-${policyId ? 'update' : 'add'}`)"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="onCancel"
    >
      <template #0>
        <SQLReviewInfo
          :name="state.name"
          :selected-environment="state.selectedEnvironment"
          :available-environment-list="availableEnvironmentList"
          :template-list="TEMPLATE_LIST"
          :selected-template-index="state.templateIndex"
          :is-edit="!!policyId"
          class="py-5"
          @select-template="tryApplyTemplate"
          @name-change="(val: string) => (state.name = val)"
          @env-change="(env: Environment) => onEnvChange(env)"
        />
      </template>
      <template #1>
        <SQLReviewConfig
          class="py-5"
          :selected-rule-list="state.selectedRuleList"
          :template-list="TEMPLATE_LIST"
          :selected-template-index="state.templateIndex"
          @change="onRuleChange"
          @apply-template="tryApplyTemplate"
        />
      </template>
      <template #2>
        <SQLReviewPreview
          :name="state.name"
          :is-preview-step="true"
          :selected-rule-list="state.selectedRuleList"
          :selected-environment="state.selectedEnvironment"
          class="py-5"
        />
      </template>
    </BBStepTab>
    <BBAlert
      v-if="state.showAlertModal"
      style="CRITICAL"
      :ok-text="$t('common.confirm')"
      :title="$t('sql-review.create.configure-rule.confirm-override-title')"
      :description="
        $t('sql-review.create.configure-rule.confirm-override-description')
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
import { BBStepTabItem } from "@/bbkit/types";
import {
  RuleLevel,
  Environment,
  RuleTemplate,
  TEMPLATE_LIST,
  convertToCategoryList,
  convertRuleTemplateToPolicyRule,
  ruleIsAvailableInSubscription,
} from "@/types";
import {
  useCurrentUser,
  pushNotification,
  useEnvironmentList,
  useSQLReviewStore,
  useSubscriptionStore,
} from "@/store";
import { hasWorkspacePermission } from "@/utils";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvironment: Environment;
  selectedRuleList: RuleTemplate[];
  ruleUpdated: boolean;
  showAlertModal: boolean;
  templateIndex: number;
  pendingApplyTemplateIndex: number;
}

const props = withDefaults(
  defineProps<{
    policyId?: number;
    name?: string;
    selectedEnvironment?: Environment;
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    policyId: undefined,
    name: "",
    selectedEnvironment: undefined,
    selectedRuleList: () => [],
  }
);

const emit = defineEmits(["cancel"]);

const { t } = useI18n();
const router = useRouter();
const store = useSQLReviewStore();
const currentUser = useCurrentUser();
const subscriptionStore = useSubscriptionStore();

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.sql-review";
const DEFAULT_TEMPLATE_INDEX = 0;

const STEP_LIST: BBStepTabItem[] = [
  { title: t("sql-review.create.basic-info.name") },
  { title: t("sql-review.create.configure-rule.name") },
  { title: t("sql-review.create.preview.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name || t("sql-review.create.basic-info.display-name-default"),
  selectedEnvironment: props.selectedEnvironment,
  selectedRuleList: [...props.selectedRuleList],
  ruleUpdated: false,
  showAlertModal: false,
  templateIndex: props.policyId ? -1 : DEFAULT_TEMPLATE_INDEX,
  pendingApplyTemplateIndex: -1,
});

const onTemplateApply = (index: number) => {
  if (index < 0 || index >= TEMPLATE_LIST.length) {
    return;
  }
  state.templateIndex = index;
  state.pendingApplyTemplateIndex = -1;

  const categoryList = convertToCategoryList(TEMPLATE_LIST[index].ruleList);
  state.selectedRuleList = categoryList.reduce((res, category) => {
    res.push(
      ...category.ruleList.map((rule) => ({
        ...rule,
        level: ruleIsAvailableInSubscription(
          rule.type,
          subscriptionStore.currentPlan
        )
          ? rule.level
          : RuleLevel.DISABLED,
      }))
    );
    return res;
  }, [] as RuleTemplate[]);
};

if (state.selectedRuleList.length === 0) {
  onTemplateApply(DEFAULT_TEMPLATE_INDEX);
}

const availableEnvironmentList = computed((): Environment[] => {
  const environmentList = useEnvironmentList(["NORMAL"]);

  const filteredList = store.availableEnvironments(
    environmentList.value,
    props.policyId
  );

  return filteredList;
});

const onCancel = () => {
  if (props.policyId) {
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
      return (
        !!state.name &&
        state.selectedRuleList.length > 0 &&
        !!state.selectedEnvironment
      );
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
  if (
    !hasWorkspacePermission(
      "bb.permission.workspace.manage-sql-review-policy",
      currentUser.value.role
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }
  const upsert = {
    name: state.name,
    ruleList: state.selectedRuleList.map((rule) =>
      convertRuleTemplateToPolicyRule(rule)
    ),
  };

  if (props.policyId) {
    store
      .updateReviewPolicy({
        id: props.policyId,
        ...upsert,
      })
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("sql-review.policy-updated"),
        });
      });
  } else {
    store
      .addReviewPolicy({
        ...upsert,
        environmentId: state.selectedEnvironment?.id,
      })
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("sql-review.policy-created"),
        });
      });
  }

  allowChangeCallback();
  onCancel();
};

const onEnvChange = (env: Environment) => {
  state.selectedEnvironment = env;
};

const tryApplyTemplate = (index: number) => {
  if (state.ruleUpdated || props.policyId) {
    state.showAlertModal = true;
    state.pendingApplyTemplateIndex = index;
    return;
  }
  onTemplateApply(index);
};

const onRuleChange = (rule: RuleTemplate) => {
  if (
    !ruleIsAvailableInSubscription(rule.type, subscriptionStore.currentPlan)
  ) {
    return;
  }

  const index = state.selectedRuleList.findIndex((r) => r.type === rule.type);
  state.selectedRuleList = [
    ...state.selectedRuleList.slice(0, index),
    rule,
    ...state.selectedRuleList.slice(index + 1),
  ];
  state.ruleUpdated = true;
};
</script>
