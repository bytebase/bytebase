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
        <SchemaReviewInfo
          :name="state.name"
          :selected-environment="state.selectedEnvironment"
          :available-environment-list="availableEnvironmentList"
          :template-list="TEMPLATE_LIST"
          :selected-template-index="state.templateIndex"
          :is-edit="!!policyId"
          @select-template="tryApplyTemplate"
          @name-change="(val) => (state.name = val)"
          @env-change="(env) => onEnvChange(env)"
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
import { BBStepTabItem } from "@/bbkit/types";
import {
  RuleType,
  ruleTemplateList,
  RuleLevel,
  Environment,
  RuleTemplate,
  SchemaReviewPolicyTemplate,
  convertRuleTemplateToPolicyRule,
} from "@/types";
import {
  useCurrentUser,
  pushNotification,
  useEnvironmentList,
  useSchemaSystemStore,
} from "@/store";
import { isOwner, isDBA } from "@/utils";

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
const store = useSchemaSystemStore();
const currentUser = useCurrentUser();

const hasPermission = computed(() => {
  return isOwner(currentUser.value.role) || isDBA(currentUser.value.role);
});

const getRuleListWithLevel = (
  typeList: RuleType[],
  level: RuleLevel
): RuleTemplate[] => {
  return typeList.reduce((res, type) => {
    const rule = ruleTemplateList.find((r) => r.type === type);
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

const TEMPLATE_LIST: SchemaReviewPolicyTemplate[] = [
  {
    title: t("schema-review-policy.template.policy-for-prod.title"),
    imagePath: new URL("../../assets/plan-enterprise.png", import.meta.url)
      .href,
    ruleList: [
      ...getRuleListWithLevel(["engine.mysql.use-innodb"], RuleLevel.ERROR),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.uk",
          "naming.index.fk",
          "naming.index.idx",
        ],
        RuleLevel.WARNING
      ),
      ...getRuleListWithLevel(
        [
          "statement.select.no-select-all",
          "statement.where.require",
          "statement.where.no-leading-wildcard-like",
          "table.require-pk",
        ],
        RuleLevel.ERROR
      ),
      ...getRuleListWithLevel(
        ["column.required", "column.no-null"],
        RuleLevel.WARNING
      ),
      ...getRuleListWithLevel(
        ["schema.backward-compatibility"],
        RuleLevel.ERROR
      ),
    ],
  },
  {
    title: t("schema-review-policy.template.policy-for-dev.title"),
    imagePath: new URL("../../assets/plan-free.png", import.meta.url).href,
    ruleList: [
      ...getRuleListWithLevel(["engine.mysql.use-innodb"], RuleLevel.ERROR),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.uk",
          "naming.index.fk",
          "naming.index.idx",
          "statement.select.no-select-all",
          "statement.where.require",
          "statement.where.no-leading-wildcard-like",
        ],
        RuleLevel.WARNING
      ),
      ...getRuleListWithLevel(["table.require-pk"], RuleLevel.ERROR),
      ...getRuleListWithLevel(
        ["column.required", "column.no-null", "schema.backward-compatibility"],
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
  selectedEnvironment: props.selectedEnvironment,
  selectedRuleList: props.selectedRuleList.length
    ? [...props.selectedRuleList]
    : [...TEMPLATE_LIST[DEFAULT_TEMPLATE_INDEX].ruleList],
  ruleUpdated: false,
  showAlertModal: false,
  templateIndex: props.policyId ? -1 : DEFAULT_TEMPLATE_INDEX,
  pendingApplyTemplateIndex: -1,
});

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
  if (!hasPermission.value) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-review-policy.no-permission"),
    });
  }
  const upsert = {
    name: state.name,
    ruleList: state.selectedRuleList.map((rule) =>
      convertRuleTemplateToPolicyRule(rule)
    ),
  };

  if (props.policyId) {
    store.updateReviewPolicy({
      id: props.policyId,
      ...upsert,
    });
  } else {
    store.addReviewPolicy({
      ...upsert,
      environmentId: state.selectedEnvironment?.id,
    });
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(
      `schema-review-policy.policy-${props.policyId ? "updated" : "created"}`
    ),
  });

  allowChangeCallback();
  onCancel();
};

const onEnvChange = (env: Environment) => {
  state.selectedEnvironment = env;
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
  if (state.ruleUpdated || props.policyId) {
    state.showAlertModal = true;
    state.pendingApplyTemplateIndex = index;
    return;
  }
  onTemplateApply(index);
};

const onRuleChange = (rule: RuleTemplate) => {
  const index = state.selectedRuleList.findIndex((r) => r.type === rule.type);
  state.selectedRuleList = [
    ...state.selectedRuleList.slice(0, index),
    rule,
    ...state.selectedRuleList.slice(index + 1),
  ];
  state.ruleUpdated = true;
};
</script>
