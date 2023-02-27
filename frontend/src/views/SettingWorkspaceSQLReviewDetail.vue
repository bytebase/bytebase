<template>
  <FeatureAttention
    v-if="!hasSQLReviewPolicyFeature"
    custom-class="my-5"
    feature="bb.feature.sql-review"
    :description="$t('subscription.features.bb-feature-sql-review.desc')"
  />
  <SQLReviewCreation
    v-if="state.editMode"
    key="sql-review-creation"
    :policy-id="reviewPolicy.id"
    :name="reviewPolicy.name"
    :selected-environment="reviewPolicy.environment"
    :selected-rule-list="selectedRuleList"
    @cancel="state.editMode = false"
  />
  <div v-else class="my-5">
    <div
      class="flex flex-col items-center space-x-2 justify-center md:flex-row"
    >
      <div class="flex-1 flex space-x-2 items-center justify-start">
        <BBBadge v-if="reviewPolicy.environment" :can-remove="false">
          {{ reviewPolicy.environment.name }}
          <ProductionEnvironmentIcon
            :environment="reviewPolicy.environment"
            class="!text-current ml-1"
          />
        </BBBadge>
        <div
          v-if="reviewPolicy.rowStatus == 'ARCHIVED'"
          class="whitespace-nowrap"
        >
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
          v-if="reviewPolicy.rowStatus === 'NORMAL'"
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
      v-if="
        !reviewPolicy.environment || reviewPolicy.environment.id === UNKNOWN_ID
      "
      class="my-5"
      :style="`WARN`"
      :title="$t('sql-review.create.basic-info.no-linked-environments')"
    />
    <SQLRuleFilter
      :rule-list="selectedRuleList"
      :params="filterParams"
      v-on="filterEvents"
    />
    <SQLReviewPreview
      key="sql-review-preview"
      :policy="reviewPolicy"
      :selected-rule-list="filteredRuleList"
      :editable="hasPermission"
    />
    <BBButtonConfirm
      :disabled="!hasPermission"
      :style="'DELETE'"
      :button-text="$t('sql-review.delete')"
      :ok-text="$t('common.delete')"
      :confirm-title="$t('common.delete') + ` '${reviewPolicy.name}'?`"
      :require-confirm="true"
      @confirm="onRemove"
    />
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
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { idFromSlug, hasWorkspacePermission } from "@/utils";
import {
  unknown,
  RuleLevel,
  RuleTemplate,
  SQLReviewPolicy,
  SchemaRuleEngineType,
  RuleType,
  TEMPLATE_LIST,
  convertPolicyRuleToRuleTemplate,
  UNKNOWN_ID,
} from "@/types";
import { BBTextField } from "@/bbkit";
import {
  featureToRef,
  useCurrentUser,
  pushNotification,
  useSQLReviewStore,
} from "@/store";
import {
  SQLRuleFilter,
  useSQLRuleFilter,
} from "../components/SQLReview/components";
import ProductionEnvironmentIcon from "@/components/Environment/ProductionEnvironmentIcon.vue";

const props = defineProps({
  sqlReviewPolicySlug: {
    required: true,
    type: String,
  },
});

interface LocalState {
  showDisableModal: boolean;
  showEnableModal: boolean;
  searchText: string;
  selectedCategory?: string;
  editMode: boolean;
  checkedEngine: Set<SchemaRuleEngineType>;
  checkedLevel: Set<RuleLevel>;
}

const { t } = useI18n();
const store = useSQLReviewStore();
const router = useRouter();
const currentUser = useCurrentUser();
const ROUTE_NAME = "setting.workspace.sql-review";

const state = reactive<LocalState>({
  showDisableModal: false,
  showEnableModal: false,
  searchText: "",
  selectedCategory: router.currentRoute.value.query.category
    ? (router.currentRoute.value.query.category as string)
    : undefined,
  editMode: false,
  checkedEngine: new Set<SchemaRuleEngineType>(),
  checkedLevel: new Set<RuleLevel>(),
});

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

const hasPermission = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUser.value.role
  );
});

const reviewPolicy = computed((): SQLReviewPolicy => {
  return (
    store.getReviewPolicyByEnvironmentId(
      idFromSlug(props.sqlReviewPolicySlug)
    ) || (unknown("SQL_REVIEW") as SQLReviewPolicy)
  );
});

const selectedRuleList = computed((): RuleTemplate[] => {
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

const {
  params: filterParams,
  events: filterEvents,
  filteredRuleList,
} = useSQLRuleFilter(selectedRuleList);

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

const onEdit = () => {
  state.editMode = true;
  state.searchText = "";
  state.selectedCategory = undefined;
};

const onArchive = () => {
  store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    rowStatus: "ARCHIVED",
  });
};

const onRestore = () => {
  store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    rowStatus: "NORMAL",
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
