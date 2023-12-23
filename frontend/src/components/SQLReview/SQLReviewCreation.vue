<template>
  <div>
    <StepTab
      :sticky="true"
      :current-index="state.currentStep"
      :step-list="STEP_LIST"
      :allow-next="allowNext"
      :finish-title="$t(`common.confirm-and-${policy ? 'update' : 'add'}`)"
      @update:current-index="changeStepIndex"
      @cancel="onCancel"
      @finish="tryFinishSetup"
    >
      <template #0>
        <SQLReviewInfo
          :name="state.name"
          :selected-environment="props.selectedEnvironment"
          :available-environment-list="availableEnvironmentList"
          :selected-template="
            state.pendingApplyTemplate || state.selectedTemplate
          "
          :is-edit="!!policy"
          @select-template="tryApplyTemplate"
          @name-change="(val: string) => (state.name = val)"
        />
      </template>
      <template #1>
        <SQLReviewConfig
          :selected-rule-list="state.selectedRuleList"
          @level-change="onLevelChange"
          @payload-change="onPayloadChange"
          @comment-change="onCommentChange"
        />
      </template>
    </StepTab>
  </div>
</template>

<script lang="ts" setup>
import { useDialog } from "naive-ui";
import { reactive, computed, withDefaults } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { StepTab } from "@/components/v2";
import {
  useCurrentUserV1,
  pushNotification,
  useSQLReviewStore,
  useSubscriptionV1Store,
  useEnvironmentV1List,
} from "@/store";
import {
  RuleTemplate,
  convertToCategoryList,
  convertRuleTemplateToPolicyRule,
  ruleIsAvailableInSubscription,
  SQLReviewPolicyTemplate,
  SQLReviewPolicy,
  SchemaPolicyRule,
} from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import SQLReviewConfig from "./SQLReviewConfig.vue";
import SQLReviewInfo from "./SQLReviewInfo.vue";
import { rulesToTemplate } from "./components";

interface LocalState {
  currentStep: number;
  name: string;
  selectedRuleList: RuleTemplate[];
  selectedTemplate: SQLReviewPolicyTemplate | undefined;
  ruleUpdated: boolean;
  pendingApplyTemplate: SQLReviewPolicyTemplate | undefined;
}

const props = withDefaults(
  defineProps<{
    policy?: SQLReviewPolicy;
    name?: string;
    selectedEnvironment: Environment;
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    policy: undefined,
    name: "",
    selectedRuleList: () => [],
  }
);

const emit = defineEmits(["cancel"]);

const dialog = useDialog();
const { t } = useI18n();
const router = useRouter();
const store = useSQLReviewStore();
const currentUserV1 = useCurrentUserV1();
const subscriptionStore = useSubscriptionV1Store();

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.sql-review";

const STEP_LIST = [
  { title: t("sql-review.create.basic-info.name") },
  { title: t("sql-review.create.configure-rule.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name || t("sql-review.create.basic-info.display-name-default"),
  selectedRuleList: [...props.selectedRuleList],
  selectedTemplate: props.policy
    ? rulesToTemplate(props.policy, false /* withDisabled=false */)
    : undefined,
  ruleUpdated: false,
  pendingApplyTemplate: undefined,
});

const onTemplateApply = (template: SQLReviewPolicyTemplate | undefined) => {
  if (!template) {
    return;
  }
  state.selectedTemplate = template;
  state.pendingApplyTemplate = undefined;

  const categoryList = convertToCategoryList(template.ruleList);
  state.selectedRuleList = categoryList.reduce<RuleTemplate[]>(
    (res, category) => {
      res.push(
        ...category.ruleList.map((rule) => ({
          ...rule,
          level: ruleIsAvailableInSubscription(
            rule.type,
            subscriptionStore.currentPlan
          )
            ? rule.level
            : SQLReviewRuleLevel.DISABLED,
        }))
      );
      return res;
    },
    []
  );
};

const availableEnvironmentList = computed((): Environment[] => {
  const environmentList = useEnvironmentV1List();
  const filteredList = store.availableEnvironments(
    environmentList.value,
    props.policy?.id
  );

  return filteredList;
});

const onCancel = () => {
  if (props.policy) {
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
        !!props.selectedEnvironment
      );
    case CONFIGURE_RULE_STEP:
      return state.selectedRuleList.length > 0;
    case PREVIEW_STEP:
      return true;
  }
  return false;
});

const changeStepIndex = (nextIndex: number) => {
  if (state.currentStep === 0 && nextIndex === 1) {
    if (state.pendingApplyTemplate) {
      dialog.warning({
        title: t("sql-review.create.configure-rule.confirm-override-title"),
        content: t(
          "sql-review.create.configure-rule.confirm-override-description"
        ),
        positiveText: t("common.confirm"),
        onPositiveClick: (_: MouseEvent) => {
          onTemplateApply(state.pendingApplyTemplate);
        },
      });
      return;
    }
  }
  state.currentStep = nextIndex;
};

const tryFinishSetup = () => {
  if (
    !hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-sql-review-policy",
      currentUserV1.value.userRole
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }

  const ruleList: SchemaPolicyRule[] = [];
  for (const rule of state.selectedRuleList) {
    ruleList.push(...convertRuleTemplateToPolicyRule(rule));
  }

  const upsert = {
    name: state.name,
    ruleList,
  };

  if (props.policy) {
    store
      .updateReviewPolicy({
        id: props.policy.id,
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
    if (!props.selectedEnvironment) {
      return;
    }
    store
      .addReviewPolicy({
        ...upsert,
        environmentPath: props.selectedEnvironment.name,
      })
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("sql-review.policy-created"),
        });
      });
  }

  onCancel();
};

const tryApplyTemplate = (template: SQLReviewPolicyTemplate) => {
  if (state.ruleUpdated || props.policy) {
    if (template.id === state.selectedTemplate?.id) {
      state.pendingApplyTemplate = undefined;
      return;
    }
    state.pendingApplyTemplate = template;
    return;
  }
  onTemplateApply(template);
};

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
  state.ruleUpdated = true;
};

const onPayloadChange = (rule: RuleTemplate, update: Partial<RuleTemplate>) => {
  change(rule, update);
};

const onLevelChange = (rule: RuleTemplate, level: SQLReviewRuleLevel) => {
  change(rule, { level });
};

const onCommentChange = (rule: RuleTemplate, comment: string) => {
  change(rule, { comment });
};
</script>
