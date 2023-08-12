<template>
  <FeatureAttention custom-class="my-4" feature="bb.feature.sql-review" />
  <SQLReviewCreation
    v-if="state.editMode"
    key="sql-review-creation"
    :policy="reviewPolicy"
    :name="reviewPolicy.name"
    :selected-environment="reviewPolicy.environment"
    :selected-rule-list="ruleListOfPolicy"
    @cancel="state.editMode = false"
  />
  <div v-else class="mt-4">
    <div
      class="flex flex-col items-center space-x-2 justify-center md:flex-row"
    >
      <div class="flex-1 flex space-x-2 items-center justify-start">
        <BBBadge
          v-if="reviewPolicy.environment"
          :can-remove="false"
          :link="`/environment/${reviewPolicy.environment.uid}`"
        >
          <EnvironmentV1Name
            :environment="reviewPolicy.environment"
            :link="false"
          />
        </BBBadge>
        <div v-if="!reviewPolicy.enforce" class="whitespace-nowrap">
          <BBBadge
            :text="$t('sql-review.disabled')"
            :can-remove="false"
            :style="'DISABLED'"
          />
        </div>
        <BBTextField
          class="flex-1 text-3xl py-0.5 px-0.5 font-bold truncate"
          :disabled="!hasPermission"
          :required="true"
          :focus-on-mount="false"
          :ends-on-enter="true"
          :bordered="false"
          :value="reviewPolicy.name"
          @end-editing="changeName"
        />
      </div>
      <div v-if="hasPermission" class="flex space-x-2">
        <button
          v-if="reviewPolicy.enforce"
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="state.showDisableModal = true"
        >
          {{ $t("common.disable") }}
        </button>
        <button
          v-else
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="state.showEnableModal = true"
        >
          {{ $t("common.enable") }}
        </button>
        <button type="button" class="btn-primary" @click="onEdit">
          {{ $t("sql-review.create.configure-rule.change-template") }}
        </button>
      </div>
    </div>
    <BBAttention
      v-if="!reviewPolicy.environment"
      class="my-4"
      :style="`WARN`"
      :title="$t('sql-review.create.basic-info.no-linked-environments')"
    />
    <SQLRuleFilter
      :rule-list="state.ruleList"
      :params="filterParams"
      v-on="filterEvents"
    />

    <SQLRuleTable
      :class="[state.updating && 'pointer-events-none']"
      :rule-list="filteredRuleList"
      :editable="hasPermission"
      @level-change="onLevelChange"
      @payload-change="onPayloadChange"
      @comment-change="onCommentChange"
    />
    <BBButtonConfirm
      class="my-4"
      :disabled="!hasPermission"
      :style="'DELETE'"
      :button-text="$t('sql-review.delete')"
      :ok-text="$t('common.delete')"
      :confirm-title="$t('common.delete') + ` '${reviewPolicy.name}'?`"
      :require-confirm="true"
      @confirm="onRemove"
    />
    <div
      v-if="state.rulesUpdated"
      class="w-full mt-4 py-4 border-t border-block-border flex justify-between bg-white sticky bottom-0 z-10"
    >
      <button type="button" class="btn-normal" @click.prevent="onCancelChanges">
        <span> {{ $t("common.cancel") }}</span>
      </button>
      <button type="button" class="btn-primary" @click.prevent="onApplyChanges">
        {{ $t("common.confirm-and-update") }}
      </button>
    </div>
  </div>
  <BBAlert
    v-if="state.showDisableModal"
    :style="'CRITICAL'"
    :ok-text="$t('common.disable')"
    :title="$t('common.disable') + ` '${reviewPolicy.name}'?`"
    description=""
    @ok="
      () => {
        state.showDisableModal = false;
        onArchive();
      }
    "
    @cancel="state.showDisableModal = false"
  >
  </BBAlert>
  <BBAlert
    v-if="state.showEnableModal"
    :style="'INFO'"
    :ok-text="$t('common.enable')"
    :title="$t('common.enable') + ` '${reviewPolicy.name}'?`"
    description=""
    @ok="
      () => {
        state.showEnableModal = false;
        onRestore();
      }
    "
    @cancel="state.showEnableModal = false"
  >
  </BBAlert>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, reactive, toRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBTextField } from "@/bbkit";
import { PayloadValueType } from "@/components/SQLReview/components/RuleConfigComponents";
import { EnvironmentV1Name } from "@/components/v2";
import {
  pushNotification,
  useCurrentUserV1,
  useSQLReviewStore,
  useSubscriptionV1Store,
} from "@/store";
import {
  unknown,
  RuleLevel,
  RuleTemplate,
  SQLReviewPolicy,
  SchemaRuleEngineType,
  RuleType,
  TEMPLATE_LIST,
  convertPolicyRuleToRuleTemplate,
  ruleIsAvailableInSubscription,
  convertRuleTemplateToPolicyRule,
} from "@/types";
import { idFromSlug, hasWorkspacePermissionV1 } from "@/utils";
import {
  payloadValueListToComponentList,
  SQLRuleFilter,
  useSQLRuleFilter,
  SQLRuleTable,
} from "../components/SQLReview/components";

const props = defineProps({
  sqlReviewPolicySlug: {
    required: true,
    type: String,
  },
});

interface LocalState {
  showDisableModal: boolean;
  showEnableModal: boolean;
  selectedCategory?: string;
  editMode: boolean;
  checkedEngine: Set<SchemaRuleEngineType>;
  checkedLevel: Set<RuleLevel>;
  ruleList: RuleTemplate[];
  rulesUpdated: boolean;
  updating: boolean;
}

const { t } = useI18n();
const store = useSQLReviewStore();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const ROUTE_NAME = "setting.workspace.sql-review";
const subscriptionStore = useSubscriptionV1Store();

const state = reactive<LocalState>({
  showDisableModal: false,
  showEnableModal: false,
  editMode: false,
  checkedEngine: new Set<SchemaRuleEngineType>(),
  checkedLevel: new Set<RuleLevel>(),
  ruleList: [],
  rulesUpdated: false,
  updating: false,
});

const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUserV1.value.userRole
  );
});

const reviewPolicy = computed((): SQLReviewPolicy => {
  return (
    store.getReviewPolicyByEnvironmentUID(
      String(idFromSlug(props.sqlReviewPolicySlug))
    ) || (unknown("SQL_REVIEW") as SQLReviewPolicy)
  );
});

const ruleListOfPolicy = computed((): RuleTemplate[] => {
  if (!reviewPolicy.value) {
    return [];
  }

  const ruleTemplateList: RuleTemplate[] = [];
  const ruleTemplateMap: Map<RuleType, RuleTemplate> = TEMPLATE_LIST.reduce(
    (map, template) => {
      for (const rule of template.ruleList) {
        map.set(rule.type, rule);
      }
      return map;
    },
    new Map<RuleType, RuleTemplate>()
  );

  for (const policyRule of reviewPolicy.value.ruleList) {
    const rule = ruleTemplateMap.get(policyRule.type);
    if (!rule) {
      continue;
    }

    const data = convertPolicyRuleToRuleTemplate(policyRule, rule);
    if (data) {
      ruleTemplateList.push(data);
    }
    ruleTemplateMap.delete(policyRule.type);
  }

  for (const rule of ruleTemplateMap.values()) {
    ruleTemplateList.push({
      ...rule,
      level: RuleLevel.DISABLED,
    });
  }

  return ruleTemplateList;
});

watch(
  ruleListOfPolicy,
  (ruleList) => {
    state.ruleList = cloneDeep(ruleList);
  },
  { immediate: true }
);

const {
  params: filterParams,
  events: filterEvents,
  filteredRuleList,
} = useSQLRuleFilter(toRef(state, "ruleList"));

const changeName = async (name: string) => {
  const policy = reviewPolicy.value;
  if (name === policy.name) {
    return;
  }
  const upsert = {
    name,
    ruleList: policy.ruleList,
  };

  await useSQLReviewStore().updateReviewPolicy({
    id: policy.id,
    ...upsert,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-updated"),
  });
};

const markChange = (rule: RuleTemplate, overrides: Partial<RuleTemplate>) => {
  if (
    !ruleIsAvailableInSubscription(rule.type, subscriptionStore.currentPlan)
  ) {
    return;
  }

  const index = state.ruleList.findIndex((r) => r.type === rule.type);
  if (index < 0) {
    return;
  }
  const newRule = {
    ...state.ruleList[index],
    ...overrides,
  };
  state.ruleList[index] = newRule;
  state.rulesUpdated = true;
};

const onPayloadChange = (rule: RuleTemplate, data: PayloadValueType[]) => {
  const componentList = payloadValueListToComponentList(rule, data);
  markChange(rule, { componentList });
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  markChange(rule, { level });
};

const onCommentChange = (rule: RuleTemplate, comment: string) => {
  markChange(rule, { comment });
};

const onCancelChanges = () => {
  state.ruleList = cloneDeep(ruleListOfPolicy.value);
  state.rulesUpdated = false;
};

const onApplyChanges = async () => {
  const policy = reviewPolicy.value;
  const upsert = {
    ruleList: state.ruleList.map((rule) =>
      convertRuleTemplateToPolicyRule(rule)
    ),
  };

  state.updating = true;
  try {
    await useSQLReviewStore().updateReviewPolicy({
      id: policy.id,
      name: policy.name,
      ...upsert,
    });
    state.rulesUpdated = false;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-updated"),
    });
  } finally {
    state.updating = false;
  }
};

const onEdit = () => {
  state.editMode = true;
  filterParams.searchText = "";
  filterParams.selectedCategory = undefined;
};

const onArchive = () => {
  store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    enforce: false,
  });
};

const onRestore = () => {
  store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    enforce: true,
  });
};

const onRemove = () => {
  store.removeReviewPolicy(reviewPolicy.value.id).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-removed"),
    });
  });
  router.replace({
    name: ROUTE_NAME,
  });
};
</script>
