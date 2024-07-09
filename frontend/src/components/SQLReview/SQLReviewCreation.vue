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
          :allow-change-attached-resource="false"
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
          :rule-map-by-engine="state.selectedRuleMapByEngine"
          @rule-change="change"
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
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
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
  RuleTemplateV2,
  SQLReviewPolicyTemplateV2,
  SQLReviewPolicy,
} from "@/types";
import {
  getRuleMapByEngine,
  convertRuleMapToPolicyRuleList,
  ruleIsAvailableInSubscription,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import SQLReviewConfig from "./SQLReviewConfig.vue";
import SQLReviewInfo from "./SQLReviewInfo.vue";
import { rulesToTemplate } from "./components";

interface LocalState {
  currentStep: number;
  name: string;
  resourceId: string;
  attachedResources: string[];
  selectedRuleMapByEngine: Map<Engine, Map<string, RuleTemplateV2>>;
  selectedTemplate: SQLReviewPolicyTemplateV2 | undefined;
  ruleUpdated: boolean;
  pendingApplyTemplate: SQLReviewPolicyTemplateV2 | undefined;
}

const props = withDefaults(
  defineProps<{
    policy?: SQLReviewPolicy;
    name?: string;
    selectedResources?: string[];
    selectedRuleList?: RuleTemplateV2[];
  }>(),
  {
    policy: undefined,
    name: "",
    selectedResources: () => [],
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
  selectedRuleMapByEngine: getRuleMapByEngine(props.selectedRuleList),
  selectedTemplate: props.policy
    ? rulesToTemplate(props.policy, false)
    : undefined,
  ruleUpdated: false,
  pendingApplyTemplate: undefined,
});

const isUpdate = computed(() => !!props.policy);

const onTemplateApply = (template: SQLReviewPolicyTemplateV2 | undefined) => {
  if (!template) {
    return;
  }
  state.selectedTemplate = template;
  state.pendingApplyTemplate = undefined;

  state.selectedRuleMapByEngine = getRuleMapByEngine(
    template.ruleList.map((rule) => ({
      ...rule,
      level: ruleIsAvailableInSubscription(
        rule.type,
        subscriptionStore.currentPlan
      )
        ? rule.level
        : SQLReviewRuleLevel.DISABLED,
    }))
  );
};

const onCancel = (newPolicy: SQLReviewPolicy | undefined = undefined) => {
  if (props.policy) {
    emit("cancel");
  } else {
    if (newPolicy) {
      router.push({
        name: WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
        params: {
          sqlReviewPolicySlug: sqlReviewPolicySlug(newPolicy),
        },
        query: {
          attachResourcePanel: newPolicy.resources.length === 0 ? 1 : undefined,
        },
      });
    } else {
      router.push({
        name: WORKSPACE_ROUTE_SQL_REVIEW,
      });
    }
  }
};

const allowNext = computed((): boolean => {
  switch (state.currentStep) {
    case BASIC_INFO_STEP:
      return (
        !!state.name &&
        !!state.resourceId &&
        state.selectedRuleMapByEngine.size > 0
      );
    case CONFIGURE_RULE_STEP:
      return state.selectedRuleMapByEngine.size > 0;
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

  const upsert = {
    title: state.name,
    ruleList: convertRuleMapToPolicyRuleList(state.selectedRuleMapByEngine),
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
    onCancel();
  } else {
    try {
      const policy = await store.createReviewPolicy({
        ...upsert,
        resources: state.attachedResources,
        id: `${reviewConfigNamePrefix}${state.resourceId}`,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-review.policy-created"),
      });
      onCancel(policy);
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("sql-review.policy-create-failed"),
      });
    }
  }
};

const tryApplyTemplate = (template: SQLReviewPolicyTemplateV2) => {
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

const change = (rule: RuleTemplateV2, overrides: Partial<RuleTemplateV2>) => {
  if (
    !ruleIsAvailableInSubscription(rule.type, subscriptionStore.currentPlan)
  ) {
    return;
  }

  const selectedRule = state.selectedRuleMapByEngine
    .get(rule.engine)
    ?.get(rule.type);
  if (!selectedRule) {
    return;
  }
  state.selectedRuleMapByEngine
    .get(rule.engine)
    ?.set(rule.type, Object.assign(selectedRule, overrides));

  state.ruleUpdated = true;
};
</script>
