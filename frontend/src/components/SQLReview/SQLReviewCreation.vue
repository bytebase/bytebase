<template>
  <div class="-my-4">
    <StepTab
      :sticky="true"
      :current-index="state.currentStep"
      :step-list="STEP_LIST"
      :allow-next="allowNext"
      :finish-title="$t(`common.confirm-and-${policy ? 'update' : 'add'}`)"
      header-class="!-top-4"
      footer-class="!-bottom-4 !pt-4"
      pane-class="!mb-4"
      @update:current-index="changeStepIndex"
      @cancel="onCancel"
      @finish="tryFinishSetup"
    >
      <template #0>
        <SQLReviewInfo
          :name="state.name"
          :resource-id="state.resourceId"
          :attached-resources="state.attachedResources"
          :selected-template="
            state.pendingApplyTemplate || state.selectedTemplate
          "
          :is-edit="!!policy"
          :is-create="!isUpdate"
          :allow-change-attached-resource="allowChangeAttachedResource"
          @select-template="tryApplyTemplate"
          @name-change="(val: string) => (state.name = val)"
          @resource-id-change="(val: string) => (state.resourceId = val)"
          @attached-resources-change="
            (val: string[]) => (state.attachedResources = val)
          "
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
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/router/dashboard/workspaceRoutes";
import {
  useCurrentUserV1,
  pushNotification,
  useSQLReviewStore,
  useSubscriptionV1Store,
} from "@/store";
import {
  reviewConfigNamePrefix,
  getReviewConfigId,
} from "@/store/modules/v1/common";
import type {
  RuleTemplate,
  SQLReviewPolicyTemplate,
  SQLReviewPolicy,
  SchemaPolicyRule,
} from "@/types";
import {
  convertToCategoryList,
  convertRuleTemplateToPolicyRule,
  ruleIsAvailableInSubscription,
} from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import SQLReviewConfig from "./SQLReviewConfig.vue";
import SQLReviewInfo from "./SQLReviewInfo.vue";
import { rulesToTemplate } from "./components";

interface LocalState {
  currentStep: number;
  name: string;
  resourceId: string;
  attachedResources: string[];
  selectedRuleList: RuleTemplate[];
  selectedTemplate: SQLReviewPolicyTemplate | undefined;
  ruleUpdated: boolean;
  pendingApplyTemplate: SQLReviewPolicyTemplate | undefined;
}

const props = withDefaults(
  defineProps<{
    policy?: SQLReviewPolicy;
    name?: string;
    selectedResources: string[];
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

const STEP_LIST = [
  { title: t("sql-review.create.basic-info.name") },
  { title: t("sql-review.create.configure-rule.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name || t("sql-review.create.basic-info.display-name-default"),
  resourceId: props.policy ? getReviewConfigId(props.policy.id) : "",
  attachedResources: props.selectedResources,
  selectedRuleList: [...props.selectedRuleList],
  selectedTemplate: props.policy
    ? rulesToTemplate(props.policy, false /* withDisabled=false */)
    : undefined,
  ruleUpdated: false,
  pendingApplyTemplate: undefined,
});

const isUpdate = computed(() => !!props.policy);

const allowChangeAttachedResource = computed(() => {
  if (isUpdate.value) {
    return (props.policy?.resources ?? []).length === 0;
  }
  return props.selectedResources.length === 0;
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

const onCancel = () => {
  if (props.policy) {
    emit("cancel");
  } else {
    router.push({
      name: WORKSPACE_ROUTE_SQL_REVIEW,
    });
  }
};

const allowNext = computed((): boolean => {
  switch (state.currentStep) {
    case BASIC_INFO_STEP:
      return (
        !!state.name &&
        !!state.resourceId &&
        state.attachedResources.length > 0 &&
        state.selectedRuleList.length > 0
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
          state.currentStep = nextIndex;
        },
      });
      return;
    }
  }
  state.currentStep = nextIndex;
};

const tryFinishSetup = async () => {
  if (!hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update")) {
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
    title: state.name,
    ruleList,
    resources: state.attachedResources,
  };

  if (isUpdate.value) {
    await store.updateReviewPolicy({
      id: props.policy!.id,
      ...upsert,
    });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-updated"),
    });
  } else {
    if (state.attachedResources.length === 0) {
      return;
    }
    try {
      await store.createReviewPolicy({
        ...upsert,
        id: `${reviewConfigNamePrefix}${state.resourceId}`,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-review.policy-created"),
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-review.policy-create-failed"),
      });
    }
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
