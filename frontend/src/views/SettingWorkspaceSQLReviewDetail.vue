<template>
  <FeatureAttention custom-class="mb-4" feature="bb.feature.sql-review" />
  <SQLReviewCreation
    v-if="state.editMode"
    key="sql-review-creation"
    class="mt-1"
    :policy="reviewPolicy"
    :name="reviewPolicy.name"
    :selected-resources="reviewPolicy.resources"
    :selected-rule-list="ruleListOfPolicy"
    @cancel="state.editMode = false"
  />
  <div v-else>
    <div
      class="flex flex-col gap-y-2 items-start md:items-center gap-x-2 justify-center md:flex-row"
    >
      <div class="flex-1 flex space-x-2 items-center justify-start">
        <div v-if="!reviewPolicy.enforce" class="whitespace-nowrap">
          <BBBadge
            :text="$t('sql-review.disabled')"
            :can-remove="false"
            :badge-style="'DISABLED'"
          />
        </div>
        <BBTextField
          class="flex-1 !text-xl md:!text-2xl py-0.5 px-0.5 font-bold truncate"
          :disabled="!hasPermission"
          :required="true"
          :focus-on-mount="false"
          :ends-on-enter="true"
          :bordered="state.editingTitle"
          :value="reviewPolicy.name"
          size="large"
          @on-focus="state.editingTitle = true"
          @end-editing="changeName"
        />
      </div>
      <div v-if="hasPermission" class="flex gap-x-2">
        <NButton
          v-if="reviewPolicy.enforce"
          @click.prevent="state.showDisableModal = true"
        >
          {{ $t("common.disable") }}
        </NButton>
        <NButton v-else @click.prevent="state.showEnableModal = true">
          {{ $t("common.enable") }}
        </NButton>
        <NButton type="primary" @click="onEdit">
          {{ $t("sql-review.create.configure-rule.change-template") }}
        </NButton>
      </div>
    </div>
    <div class="mt-4 space-y-4">
      <BBAttention v-if="reviewPolicy.resources.length === 0" type="warning">
        {{ $t("sql-review.no-linked-resources") }}
      </BBAttention>
      <div class="flex space-x-2 items-center">
        <BBBadge
          v-for="resource in reviewPolicy.resources"
          :key="resource"
          :can-remove="false"
        >
          <SQLReviewAttachedResource :resource="resource" :show-prefix="true" />
        </BBBadge>
      </div>
    </div>

    <SQLReviewTabsByEngine
      class="mt-5"
      :rule-map-by-engine="state.ruleMapByEngine"
    >
      <template
        #default="{ ruleList: ruleListFilteredByEngine, engine }: { ruleList: RuleTemplateV2[]; engine: Engine; }"
      >
        <SQLRuleTableWithFilter
          :engine="engine"
          :rule-list="ruleListFilteredByEngine"
          :editable="hasPermission"
          @rule-change="markChange"
        />
      </template>
    </SQLReviewTabsByEngine>

    <BBButtonConfirm
      class="!my-4"
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
      class="w-full mt-4 py-4 border-t border-block-border flex justify-between bg-white sticky -bottom-4 z-10"
    >
      <NButton @click.prevent="onCancelChanges">
        <span> {{ $t("common.cancel") }}</span>
      </NButton>
      <NButton type="primary" @click.prevent="onApplyChanges">
        {{ $t("common.confirm-and-update") }}
      </NButton>
    </div>
  </div>
  <BBAlert
    v-model:show="state.showDisableModal"
    type="warning"
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
  />
  <BBAlert
    v-model:show="state.showEnableModal"
    type="info"
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
  />
</template>

<script lang="tsx" setup>
import { useTitle } from "@vueuse/core";
import { computed, reactive, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBTextField } from "@/bbkit";
import { rulesToTemplate } from "@/components/SQLReview/components/utils";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useCurrentUserV1,
  useSQLReviewStore,
  useSubscriptionV1Store,
} from "@/store";
import type { RuleTemplateV2 } from "@/types";
import {
  unknown,
  UNKNOWN_ID,
  getRuleMapByEngine,
  ruleIsAvailableInSubscription,
  convertRuleMapToPolicyRuleList,
} from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  sqlReviewPolicySlug: string;
}>();

interface LocalState {
  showDisableModal: boolean;
  showEnableModal: boolean;
  selectedCategory?: string;
  editMode: boolean;
  ruleMapByEngine: Map<Engine, Map<string, RuleTemplateV2>>;
  rulesUpdated: boolean;
  updating: boolean;
  editingTitle: boolean;
}

const { t } = useI18n();
const store = useSQLReviewStore();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const subscriptionStore = useSubscriptionV1Store();

const state = reactive<LocalState>({
  showDisableModal: false,
  showEnableModal: false,
  editMode: false,
  ruleMapByEngine: new Map(),
  rulesUpdated: false,
  updating: false,
  editingTitle: false,
});

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUserV1.value, "bb.policies.update");
});

watchEffect(async () => {
  await store.getOrFetchReviewPolicyByName(props.sqlReviewPolicySlug);
});

const reviewPolicy = computed(() => {
  return (
    store.getReviewPolicyByName(props.sqlReviewPolicySlug) ??
    unknown("SQL_REVIEW")
  );
});

const ruleListOfPolicy = computed((): RuleTemplateV2[] => {
  if (reviewPolicy.value.id === `${UNKNOWN_ID}`) {
    return [];
  }
  return rulesToTemplate(reviewPolicy.value, true).ruleList;
});

watch(
  ruleListOfPolicy,
  (ruleList) => {
    state.ruleMapByEngine = getRuleMapByEngine(ruleList);
  },
  { immediate: true }
);

const pushUpdatedNotify = () => {
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-updated"),
  });
};

const changeName = async (title: string) => {
  state.editingTitle = false;
  const policy = reviewPolicy.value;
  if (title === policy.name) {
    return;
  }
  const upsert = {
    title,
    ruleList: policy.ruleList,
  };

  await store.updateReviewPolicy({
    id: policy.id,
    ...upsert,
  });
  pushUpdatedNotify();
};

const markChange = (
  rule: RuleTemplateV2,
  overrides: Partial<RuleTemplateV2>
) => {
  if (
    !ruleIsAvailableInSubscription(rule.type, subscriptionStore.currentPlan)
  ) {
    return;
  }

  const selectedRule = state.ruleMapByEngine.get(rule.engine)?.get(rule.type);
  if (!selectedRule) {
    return;
  }
  state.ruleMapByEngine
    .get(rule.engine)
    ?.set(rule.type, Object.assign(selectedRule, overrides));

  state.rulesUpdated = true;
};

const onCancelChanges = () => {
  state.ruleMapByEngine = getRuleMapByEngine(ruleListOfPolicy.value);
  state.rulesUpdated = false;
};

const onApplyChanges = async () => {
  const policy = reviewPolicy.value;

  state.updating = true;
  try {
    await store.updateReviewPolicy({
      id: policy.id,
      title: policy.name,
      ruleList: convertRuleMapToPolicyRuleList(state.ruleMapByEngine),
    });
    state.rulesUpdated = false;
    pushUpdatedNotify();
  } finally {
    state.updating = false;
  }
};

const onEdit = () => {
  state.editMode = true;
};

const onArchive = async () => {
  await store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    enforce: false,
  });
  pushUpdatedNotify();
};

const onRestore = async () => {
  await store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    enforce: true,
  });
  pushUpdatedNotify();
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
    name: WORKSPACE_ROUTE_SQL_REVIEW,
  });
};

useTitle(computed(() => reviewPolicy.value.name));
</script>
