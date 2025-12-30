<template>
  <SQLReviewCreation
    v-if="state.editMode"
    key="sql-review-creation"
    class="mt-1"
    :policy="reviewPolicy"
    :name="reviewPolicy.name"
    :selected-rule-list="ruleListOfPolicy"
    :selected-resources="reviewPolicy.resources"
    @cancel="state.editMode = false"
  />
  <div v-else>
    <BBAttention v-if="!reviewPolicy.enforce" type="warning" class="mb-4">
      {{ $t("sql-review.disabled") }}
    </BBAttention>
    <div
      class="flex flex-col gap-y-2 items-start md:items-center gap-x-2 justify-center md:flex-row"
    >
      <BBTextField
        class="flex-1 text-xl! pl-0! px-0.5 font-bold truncate sql-review-title"
        :disabled="!hasUpdateReviwConfigPermission"
        :required="true"
        :focus-on-mount="false"
        :ends-on-enter="true"
        :bordered="state.editingTitle"
        :value="reviewPolicy.name"
        @on-focus="state.editingTitle = true"
        @end-editing="changeName"
      />
      <div v-if="hasUpdateReviwConfigPermission" class="flex gap-x-2">
        <NButton
          v-if="reviewPolicy.enforce"
          @click.prevent="state.showDisableModal = true"
        >
          {{ $t("common.disable") }}
        </NButton>
        <NButton v-else @click.prevent="state.showEnableModal = true">
          {{ $t("common.enable") }}
        </NButton>
        <NButton
          v-if="reviewPolicy.resources.length > 0 && hasTagPolicyPermission"
          @click.prevent="state.showResourcePanel = true"
        >
          {{ $t("sql-review.attach-resource.change-resources") }}
        </NButton>
        <NButton type="primary" @click="onEdit">
          {{ $t("sql-review.create.configure-rule.change-template") }}
        </NButton>
      </div>
    </div>
    <div class="mt-4 flex flex-col gap-y-4">
      <BBAttention
        v-if="reviewPolicy.resources.length === 0"
        type="warning"
        :title="$t('sql-review.attach-resource.no-linked-resources')"
        :description="$t('sql-review.attach-resource.label')"
        :action-text="$t('sql-review.attach-resource.self')"
        @click="state.showResourcePanel = true"
      />
      <div class="flex flex-wrap gap-y-2 gap-x-2">
        <NTag
          v-for="resource in reviewPolicy.resources"
          :key="resource"
          type="primary"
        >
          <Resource :resource="resource" :show-prefix="true" />
        </NTag>
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
          :editable="hasUpdateReviwConfigPermission"
          @rule-upsert="markChange"
          @rule-remove="removeRule"
        />
      </template>
    </SQLReviewTabsByEngine>


    <NDivider />
    <BBButtonConfirm
      :disabled="!hasDeleteReviwConfigPermission"
      :type="'DELETE'"
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

  <SQLReviewAttachResourcesPanel
    :show="state.showResourcePanel"
    :review="reviewPolicy"
    @close="state.showResourcePanel = false"
  />
</template>

<script lang="tsx" setup>
import { useTitle } from "@vueuse/core";
import { NButton, NDivider, NTag } from "naive-ui";
import {
  computed,
  nextTick,
  onMounted,
  reactive,
  watch,
  watchEffect,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAlert, BBAttention, BBButtonConfirm, BBTextField } from "@/bbkit";
import { SQLReviewCreation } from "@/components/SQLReview";
import SQLReviewAttachResourcesPanel from "@/components/SQLReview/components/SQLReviewAttachResourcesPanel.vue";
import SQLReviewTabsByEngine from "@/components/SQLReview/components/SQLReviewTabsByEngine.vue";
import SQLRuleTableWithFilter from "@/components/SQLReview/components/SQLRuleTableWithFilter.vue";
import { rulesToTemplate } from "@/components/SQLReview/components/utils";
import Resource from "@/components/v2/ResourceOccupiedModal/Resource.vue";
import { WORKSPACE_ROUTE_SQL_REVIEW } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useSQLReviewStore } from "@/store";
import type { RuleTemplateV2 } from "@/types";
import {
  convertRuleMapToPolicyRuleList,
  getRuleMapByEngine,
  UNKNOWN_ID,
  unknown,
} from "@/types";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import { hasWorkspacePermissionV2, sqlReviewNameFromSlug } from "@/utils";

const props = defineProps<{
  sqlReviewPolicySlug: string;
}>();

interface LocalState {
  showDisableModal: boolean;
  showEnableModal: boolean;
  selectedCategory?: string;
  editMode: boolean;
  ruleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
  rulesUpdated: boolean;
  updating: boolean;
  editingTitle: boolean;
  showResourcePanel: boolean;
}

const { t } = useI18n();
const store = useSQLReviewStore();
const router = useRouter();
const route = useRoute();

const state = reactive<LocalState>({
  showDisableModal: false,
  showEnableModal: false,
  editMode: false,
  ruleMapByEngine: new Map(),
  rulesUpdated: false,
  updating: false,
  editingTitle: false,
  showResourcePanel: false,
});

const hasUpdateReviwConfigPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.reviewConfigs.update");
});

const hasDeleteReviwConfigPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.reviewConfigs.delete");
});

const hasTagPolicyPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
});

const sqlReviewName = computed(() =>
  sqlReviewNameFromSlug(props.sqlReviewPolicySlug)
);

watchEffect(async () => {
  await store.getOrFetchReviewPolicyByName(sqlReviewName.value);
});

onMounted(() => {
  if (route.query.attachResourcePanel && hasTagPolicyPermission.value) {
    nextTick(() => {
      state.showResourcePanel = true;
    });
  }
});

const reviewPolicy = computed(() => {
  return (
    store.getReviewPolicyByName(sqlReviewName.value) ?? unknown("SQL_REVIEW")
  );
});

const ruleListOfPolicy = computed((): RuleTemplateV2[] => {
  if (reviewPolicy.value.id === `${UNKNOWN_ID}`) {
    return [];
  }
  return rulesToTemplate(reviewPolicy.value).ruleList;
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

  await store.upsertReviewPolicy({
    id: policy.id,
    title,
  });
  pushUpdatedNotify();
};

const markChange = (
  rule: RuleTemplateV2,
  overrides: Partial<RuleTemplateV2>
) => {
  const selectedRule = state.ruleMapByEngine.get(rule.engine)?.get(rule.type);
  if (!selectedRule) {
    return;
  }
  state.ruleMapByEngine
    .get(rule.engine)
    ?.set(rule.type, Object.assign(selectedRule, overrides));

  state.rulesUpdated = true;
};

const removeRule = (rule: RuleTemplateV2) => {
  state.ruleMapByEngine.get(rule.engine)?.delete(rule.type);
  if (state.ruleMapByEngine.get(rule.engine)?.size === 0) {
    state.ruleMapByEngine.delete(rule.engine);
  }
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
    await store.upsertReviewPolicy({
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
  await store.upsertReviewPolicy({
    id: reviewPolicy.value.id,
    enforce: false,
  });
  pushUpdatedNotify();
};

const onRestore = async () => {
  await store.upsertReviewPolicy({
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

<style lang="postcss" scoped>
.sql-review-title :deep(.n-input-wrapper) {
  padding-left: 0.3rem !important;
}
</style>
