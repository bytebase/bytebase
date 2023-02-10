<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttention
      v-if="!hasAccessControlFeature"
      feature="bb.feature.access-control"
      :description="$t('subscription.features.bb-feature-access-control.desc')"
    />
    <div class="flex justify-between">
      <i18n-t
        tag="div"
        keypath="settings.access-control.description"
        class="textinfolabel"
      >
        <template #link>
          <LearnMoreLink
            url="https://www.bytebase.com/docs/administration/database-access-control"
          />
        </template>
      </i18n-t>

      <div>
        <button
          class="btn-primary whitespace-nowrap"
          :disabled="!allowAdmin || !hasAccessControlFeature"
          @click="state.showAddRuleModal = true"
        >
          {{ $t("settings.access-control.add-rule") }}
        </button>
      </div>
    </div>

    <div v-if="hasAccessControlFeature" class="relative min-h-[12rem]">
      <BBTable
        :column-list="COLUMN_LIST"
        :data-source="activePolicyList"
        :show-header="true"
        :left-bordered="true"
        :right-bordered="true"
        :row-clickable="false"
      >
        <template #body="{ rowData: policy }: { rowData: Policy }">
          <BBTableCell class="w-[25%]" :left-padding="4">
            <div class="flex items-center space-x-2">
              <span>{{ databaseOfPolicy(policy).name }}</span>
            </div>
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            {{ projectName(databaseOfPolicy(policy).project) }}
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <div class="flex items-center">
              {{
                environmentName(databaseOfPolicy(policy).instance.environment)
              }}
              <ProductionEnvironmentIcon
                class="ml-1"
                :environment="databaseOfPolicy(policy).instance.environment"
              />
            </div>
          </BBTableCell>
          <BBTableCell class="w-[15%]">
            <div class="flex flex-row items-center space-x-1">
              <InstanceEngineIcon
                :instance="databaseOfPolicy(policy).instance"
              />
              <span class="flex-1 whitespace-pre-wrap">
                {{ instanceName(databaseOfPolicy(policy).instance) }}
              </span>
            </div>
          </BBTableCell>
          <BBTableCell>
            <div class="flex items-center justify-center">
              <NPopconfirm @positive-click="handleRemove(policy)">
                <template #trigger>
                  <button
                    :disabled="!allowAdmin"
                    class="w-5 h-5 p-0.5 bg-white hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  >
                    <heroicons-outline:trash />
                  </button>
                </template>

                <div class="whitespace-nowrap">
                  {{ $t("settings.access-control.remove-policy-tips") }}
                </div>
              </NPopconfirm>
            </div>
          </BBTableCell>
        </template>
      </BBTable>

      <div
        v-if="!state.isLoading && activePolicyList.length === 0"
        class="w-full flex flex-col py-6 justify-start items-center"
      >
        <heroicons-outline:inbox class="w-12 h-auto text-gray-500" />
        <span class="text-sm leading-6 text-gray-500">{{
          $t("common.no-data")
        }}</span>
      </div>

      <template v-if="inactivePolicyList.length > 0">
        <div class="text-lg mt-6 mb-1">
          {{ $t("settings.access-control.inactive-rules") }}
        </div>
        <div class="textinfolabel max-w-xl mb-4">
          {{ $t("settings.access-control.inactive-rules-description") }}
        </div>
        <BBTable
          :column-list="COLUMN_LIST"
          :data-source="inactivePolicyList"
          :show-header="true"
          :left-bordered="true"
          :right-bordered="true"
          :row-clickable="false"
        >
          <template #body="{ rowData: policy }: { rowData: Policy }">
            <BBTableCell class="w-[25%]" :left-padding="4">
              <div class="flex items-center space-x-2">
                <span>{{ databaseOfPolicy(policy).name }}</span>
              </div>
            </BBTableCell>
            <BBTableCell class="w-[15%]">
              {{ projectName(databaseOfPolicy(policy).project) }}
            </BBTableCell>
            <BBTableCell class="w-[15%]">
              <div class="flex items-center">
                {{
                  environmentName(databaseOfPolicy(policy).instance.environment)
                }}
                <ProductionEnvironmentIcon
                  class="ml-1"
                  :environment="databaseOfPolicy(policy).instance.environment"
                />
              </div>
            </BBTableCell>
            <BBTableCell class="w-[15%]">
              <div class="flex flex-row items-center space-x-1">
                <InstanceEngineIcon
                  :instance="databaseOfPolicy(policy).instance"
                />
                <span class="flex-1 whitespace-pre-wrap">
                  {{ instanceName(databaseOfPolicy(policy).instance) }}
                </span>
              </div>
            </BBTableCell>
            <BBTableCell>
              <div class="flex items-center justify-center">
                <NPopconfirm @positive-click="handleRemove(policy)">
                  <template #trigger>
                    <button
                      :disabled="!allowAdmin"
                      class="w-5 h-5 p-0.5 bg-white hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                    >
                      <heroicons-outline:trash />
                    </button>
                  </template>

                  <div class="whitespace-nowrap">
                    {{ $t("settings.access-control.remove-policy-tips") }}
                  </div>
                </NPopconfirm>
              </div>
            </BBTableCell>
          </template>
        </BBTable>
      </template>

      <div
        v-if="state.isLoading || state.isUpdating"
        class="absolute w-full h-full inset-0 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>
    </div>

    <template v-else>
      <BBTable
        :column-list="COLUMN_LIST"
        :data-source="[]"
        :show-header="true"
        :left-bordered="true"
        :right-bordered="true"
        :row-clickable="false"
      />
      <div class="w-full h-full flex flex-col items-center justify-center">
        <img src="../assets/illustration/no-data.webp" class="max-h-[30vh]" />
      </div>
    </template>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.access-control"
    @cancel="state.showFeatureModal = false"
  />

  <BBModal
    v-if="state.showAddRuleModal"
    :title="$t('settings.access-control.add-rule')"
    @close="state.showAddRuleModal = false"
  >
    <AddRuleForm
      :policy-list="state.policyList"
      :database-list="state.databaseList"
      @cancel="state.showAddRuleModal = false"
      @add="handleAddRule"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { NPopconfirm } from "naive-ui";
import { uniq } from "lodash-es";

import {
  featureToRef,
  useCurrentUser,
  useDatabaseStore,
  usePolicyStore,
} from "@/store";
import {
  AccessControlPolicyPayload,
  Database,
  DatabaseId,
  DEFAULT_PROJECT_ID,
  Policy,
  PolicyUpsert,
} from "@/types";
import { BBTableColumn } from "@/bbkit/types";
import { hasWorkspacePermission } from "@/utils";
import AddRuleForm from "@/components/AccessControl/AddRuleForm.vue";

interface LocalState {
  showFeatureModal: boolean;
  showAddRuleModal: boolean;
  isLoading: boolean;
  isUpdating: boolean;
  policyList: Policy[];
  databaseList: Database[];
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showAddRuleModal: false,
  isLoading: false,
  isUpdating: false,
  policyList: [],
  databaseList: [],
});
const databaseStore = useDatabaseStore();
const policyStore = usePolicyStore();
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const currentUser = useCurrentUser();
const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-access-control",
    currentUser.value.role
  );
});

const databaseOfPolicy = (policy: Policy) => {
  return databaseStore.getDatabaseById(policy.resourceId as DatabaseId);
};

const activePolicyList = computed(() => {
  return state.policyList.filter(
    (policy) =>
      databaseOfPolicy(policy).instance.environment.tier === "PROTECTED"
  );
});

const inactivePolicyList = computed(() => {
  return state.policyList.filter(
    (policy) =>
      databaseOfPolicy(policy).instance.environment.tier === "UNPROTECTED"
  );
});

const prepareList = async () => {
  state.isLoading = true;

  const policyList = await policyStore.fetchPolicyListByTypeAndResourceType(
    "bb.policy.access-control",
    "DATABASE"
  );

  const allDatabaseList = await databaseStore.fetchDatabaseList();
  state.databaseList = allDatabaseList
    .filter((db) => db.instance.environment.tier === "PROTECTED")
    .filter((db) => db.project.id !== DEFAULT_PROJECT_ID);

  // For some policy related databases that are not in the state.databaseList,
  // fetch them.
  const databaseIdList = uniq(
    policyList
      .map((policy) => policy.resourceId as DatabaseId)
      .filter((databaseId) =>
        state.databaseList.findIndex((db) => db.id === databaseId)
      )
  );

  Promise.all(
    databaseIdList.map((databaseId) =>
      databaseStore.getOrFetchDatabaseById(databaseId)
    )
  );

  state.policyList = policyList;

  state.isLoading = false;
};

watchEffect(prepareList);

const handleAddRule = async (databaseList: Database[]) => {
  if (!hasAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.showAddRuleModal = false;
  state.isUpdating = true;
  try {
    for (let i = 0; i < databaseList.length; i++) {
      const database = databaseList[i];
      const payload: AccessControlPolicyPayload = {
        disallowRuleList: [],
      };
      const policyUpsert: PolicyUpsert = {
        inheritFromParent: false,
        payload,
      };
      await policyStore.upsertPolicyByDatabaseAndType({
        databaseId: database.id,
        type: "bb.policy.access-control",
        policyUpsert,
      });
    }
  } finally {
    state.isUpdating = false;
  }
  prepareList();
};

const handleRemove = async (policy: Policy) => {
  if (!hasAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.isUpdating = true;
  try {
    await policyStore.deletePolicyByDatabaseAndType({
      databaseId: policy.resourceId,
      type: "bb.policy.access-control",
    });

    prepareList();
  } finally {
    state.isUpdating = false;
  }
};

const COLUMN_LIST = computed((): BBTableColumn[] => [
  {
    title: t("common.database"),
  },
  {
    title: t("common.project"),
  },
  {
    title: t("common.environment"),
  },
  {
    title: t("common.instance"),
  },
  {
    title: t("common.operation"),
    center: true,
    nowrap: true,
  },
]);
</script>
