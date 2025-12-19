<template>
  <div class="w-full h-full flex flex-col">
    <StepTab
      :sticky="true"
      :current-index="state.currentStep"
      :step-list="STEP_LIST"
      :allow-next="allowNext"
      :finish-title="finishTitle"
      class="flex-1 overflow-hidden flex flex-col"
      pane-class="flex-1 overflow-y-auto"
      @update:current-index="changeStepIndex"
      @cancel="onCancel"
      @finish="tryFinishSetup"
    >
      <template #0>
        <SQLReviewInfo
          :name="state.name"
          :resource-id="state.resourceId"
          :selected-template-id="
            state.pendingApplyTemplate?.id ?? state.selectedTemplateId
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
          @rule-upsert="upsertRule"
          @rule-remove="removeRole"
        />
      </template>
    </StepTab>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { useDialog } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { StepTab } from "@/components/v2";
import {
  WORKSPACE_ROUTE_SQL_REVIEW,
  WORKSPACE_ROUTE_SQL_REVIEW_DETAIL,
} from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useSQLReviewStore } from "@/store";
import {
  getReviewConfigId,
  reviewConfigNamePrefix,
} from "@/store/modules/v1/common";
import type {
  RuleTemplateV2,
  SQLReviewPolicy,
  SQLReviewPolicyTemplateV2,
} from "@/types";
import {
  TEMPLATE_LIST_V2 as builtInTemplateList,
  convertRuleMapToPolicyRuleList,
  getRuleMapByEngine,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import { hasWorkspacePermissionV2, sqlReviewPolicySlug } from "@/utils";
import { getTemplateId } from "./components";
import SQLReviewConfig from "./SQLReviewConfig.vue";
import SQLReviewInfo from "./SQLReviewInfo.vue";

interface LocalState {
  currentStep: number;
  name: string;
  resourceId: string;
  attachedResources: string[];
  selectedRuleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
  selectedTemplateId: string | undefined;
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

const finishTitle = computed(() => {
  if (props.policy) {
    return t("common.confirm-and-update");
  }
  return t("common.confirm-and-add");
});

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
  selectedTemplateId: props.policy
    ? getTemplateId(props.policy)
    : builtInTemplateList[0]?.id,
  ruleUpdated: false,
  pendingApplyTemplate: undefined,
});

const isUpdate = computed(() => !!props.policy);

const onTemplateApply = (template: SQLReviewPolicyTemplateV2 | undefined) => {
  if (!template) {
    return;
  }
  state.selectedTemplateId = template.id;
  state.pendingApplyTemplate = undefined;

  state.selectedRuleMapByEngine = getRuleMapByEngine(template.ruleList);
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
      return !!state.name && !!state.resourceId;
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
  if (
    !hasWorkspacePermissionV2(
      isUpdate.value ? "bb.reviewConfigs.update" : "bb.reviewConfigs.create"
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }

  try {
    const policy = await store.upsertReviewPolicy({
      title: state.name,
      ruleList: convertRuleMapToPolicyRuleList(state.selectedRuleMapByEngine),
      resources: isEqual(props.selectedResources, state.attachedResources)
        ? undefined
        : state.attachedResources,
      id: `${reviewConfigNamePrefix}${state.resourceId}`,
      enforce: isUpdate.value ? undefined : true,
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: isUpdate.value
        ? t("sql-review.policy-updated")
        : t("sql-review.policy-created"),
    });
    onCancel(policy);
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: isUpdate.value
        ? t("sql-review.policy-update-failed")
        : t("sql-review.policy-create-failed"),
    });
  }
};

const tryApplyTemplate = (template: SQLReviewPolicyTemplateV2) => {
  if (state.ruleUpdated || props.policy) {
    if (template.id === state.selectedTemplateId) {
      state.pendingApplyTemplate = undefined;
      return;
    }
    state.pendingApplyTemplate = template;
    return;
  }
  onTemplateApply(template);
};

const removeRole = (rule: RuleTemplateV2) => {
  state.selectedRuleMapByEngine.get(rule.engine)?.delete(rule.type);
  if (state.selectedRuleMapByEngine.get(rule.engine)?.size === 0) {
    state.selectedRuleMapByEngine.delete(rule.engine);
  }
};

const upsertRule = (
  rule: RuleTemplateV2,
  overrides: Partial<RuleTemplateV2>
) => {
  if (!state.selectedRuleMapByEngine.has(rule.engine)) {
    state.selectedRuleMapByEngine.set(
      rule.engine,
      new Map<SQLReviewRule_Type, RuleTemplateV2>()
    );
  }

  const selectedRule = state.selectedRuleMapByEngine
    .get(rule.engine)
    ?.get(rule.type);
  if (!selectedRule) {
    // Adding a new rule - use the rule's level from the template
    state.selectedRuleMapByEngine.get(rule.engine)?.set(rule.type, {
      ...rule,
      ...overrides,
    });
    state.ruleUpdated = true;
    return;
  }
  // Updating existing rule with overrides
  state.selectedRuleMapByEngine
    .get(rule.engine)
    ?.set(rule.type, Object.assign(selectedRule, overrides));

  state.ruleUpdated = true;
};
</script>
